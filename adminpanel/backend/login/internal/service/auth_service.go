package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/ivankudzin/tgapp/adminpanel/backend/login/internal/repo"
	"github.com/ivankudzin/tgapp/adminpanel/backend/login/internal/security"
)

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrForbidden         = errors.New("forbidden")
	ErrAccountLocked     = errors.New("account locked")
	ErrChallengeExpired  = errors.New("challenge expired or not found")
	ErrChallengeStep     = errors.New("invalid challenge step")
	ErrTOTPNotConfigured = errors.New("2fa is not configured")
	ErrSessionExpired    = errors.New("session expired")
)

type Service struct {
	users            *repo.AdminUserRepo
	challenges       *repo.ChallengeRepo
	setupTokens      *repo.TOTPSetupTokenRepo
	sessions         *repo.SessionRepo
	tokens           *security.TokenManager
	secretCipher     *security.SecretCipher
	telegramBotToken string
	telegramMaxAge   time.Duration
	challengeTTL     time.Duration
	totpSetupTTL     time.Duration
	sessionIdleTTL   time.Duration
	sessionMaxTTL    time.Duration
	maxAttempts      int
	lockDuration     time.Duration
	issuer           string
	devMode          bool
}

type TelegramStartInput struct {
	InitData  string
	IP        string
	UserAgent string
}

type TelegramStartResult struct {
	ChallengeID string `json:"challenge_id"`
	NextStep    string `json:"next_step"`
	Username    string `json:"username,omitempty"`
}

type VerifyTOTPResult struct {
	ChallengeID string `json:"challenge_id"`
	NextStep    string `json:"next_step"`
}

type VerifyPasswordResult struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
	Admin       AdminInfo `json:"admin"`
}

type AdminInfo struct {
	ID          int64  `json:"id"`
	TelegramID  int64  `json:"telegram_id"`
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Role        string `json:"role"`
}

type TOTPSetupStartResult struct {
	SetupID       string    `json:"setup_id"`
	TelegramID    int64     `json:"telegram_id"`
	OTPAuthURL    string    `json:"otpauth_url"`
	Secret        string    `json:"secret"`
	QRCodeDataURL string    `json:"qr_code_data_url"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func NewService(
	users *repo.AdminUserRepo,
	challenges *repo.ChallengeRepo,
	setupTokens *repo.TOTPSetupTokenRepo,
	sessions *repo.SessionRepo,
	tokens *security.TokenManager,
	secretCipher *security.SecretCipher,
	telegramBotToken string,
	telegramMaxAge time.Duration,
	challengeTTL time.Duration,
	totpSetupTTL time.Duration,
	sessionIdleTTL time.Duration,
	sessionMaxTTL time.Duration,
	maxAttempts int,
	lockDuration time.Duration,
	issuer string,
	devMode bool,
) *Service {
	return &Service{
		users:            users,
		challenges:       challenges,
		setupTokens:      setupTokens,
		sessions:         sessions,
		tokens:           tokens,
		secretCipher:     secretCipher,
		telegramBotToken: telegramBotToken,
		telegramMaxAge:   telegramMaxAge,
		challengeTTL:     challengeTTL,
		totpSetupTTL:     totpSetupTTL,
		sessionIdleTTL:   sessionIdleTTL,
		sessionMaxTTL:    sessionMaxTTL,
		maxAttempts:      maxAttempts,
		lockDuration:     lockDuration,
		issuer:           issuer,
		devMode:          devMode,
	}
}

func (s *Service) TelegramStart(ctx context.Context, in TelegramStartInput) (TelegramStartResult, error) {
	identity, err := security.ParseAndValidateInitData(in.InitData, s.telegramBotToken, s.telegramMaxAge, s.devMode)
	if err != nil {
		return TelegramStartResult{}, ErrUnauthorized
	}

	user, err := s.users.FindByTelegramID(ctx, identity.UserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return TelegramStartResult{}, ErrUnauthorized
		}
		return TelegramStartResult{}, fmt.Errorf("find admin user: %w", err)
	}
	if !user.IsActive {
		return TelegramStartResult{}, ErrForbidden
	}
	if isLocked(user.LockedUntil) {
		return TelegramStartResult{}, ErrAccountLocked
	}
	if !user.TOTPEnabled || strings.TrimSpace(user.TOTPSecret) == "" {
		return TelegramStartResult{}, ErrTOTPNotConfigured
	}

	challenge, err := s.challenges.Create(ctx, user.ID, s.challengeTTL, in.IP, in.UserAgent)
	if err != nil {
		return TelegramStartResult{}, fmt.Errorf("create login challenge: %w", err)
	}

	username := user.Username
	if username == "" {
		username = identity.Username
	}

	return TelegramStartResult{
		ChallengeID: challenge.ID.String(),
		NextStep:    "2fa",
		Username:    username,
	}, nil
}

func (s *Service) VerifyTOTP(ctx context.Context, challengeID, code string) (VerifyTOTPResult, error) {
	id, err := uuid.Parse(strings.TrimSpace(challengeID))
	if err != nil {
		return VerifyTOTPResult{}, ErrInvalidInput
	}
	challenge, err := s.challenges.GetActive(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return VerifyTOTPResult{}, ErrChallengeExpired
		}
		return VerifyTOTPResult{}, fmt.Errorf("get challenge: %w", err)
	}
	if challenge.Status != repo.ChallengeTelegramVerified {
		return VerifyTOTPResult{}, ErrChallengeStep
	}

	user, err := s.users.FindByID(ctx, challenge.AdminUserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return VerifyTOTPResult{}, ErrUnauthorized
		}
		return VerifyTOTPResult{}, fmt.Errorf("find admin user: %w", err)
	}
	if isLocked(user.LockedUntil) {
		return VerifyTOTPResult{}, ErrAccountLocked
	}
	if !user.TOTPEnabled || strings.TrimSpace(user.TOTPSecret) == "" {
		return VerifyTOTPResult{}, ErrTOTPNotConfigured
	}
	totpSecret, err := s.decryptTOTPSecret(user.TOTPSecret)
	if err != nil {
		return VerifyTOTPResult{}, fmt.Errorf("decrypt totp secret: %w", err)
	}

	if !security.ValidateTOTP(totpSecret, code, time.Now().UTC()) {
		if err := s.applyFailure(ctx, user.ID); err != nil {
			return VerifyTOTPResult{}, err
		}
		return VerifyTOTPResult{}, ErrUnauthorized
	}

	if err := s.challenges.AdvanceStatus(ctx, challenge.ID, repo.ChallengeTelegramVerified, repo.ChallengeTOTPVerified); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return VerifyTOTPResult{}, ErrChallengeExpired
		}
		return VerifyTOTPResult{}, fmt.Errorf("advance challenge step: %w", err)
	}

	return VerifyTOTPResult{
		ChallengeID: challenge.ID.String(),
		NextStep:    "password",
	}, nil
}

func (s *Service) VerifyPassword(ctx context.Context, challengeID, password string) (VerifyPasswordResult, error) {
	id, err := uuid.Parse(strings.TrimSpace(challengeID))
	if err != nil {
		return VerifyPasswordResult{}, ErrInvalidInput
	}
	password = strings.TrimSpace(password)
	if password == "" {
		return VerifyPasswordResult{}, ErrInvalidInput
	}

	challenge, err := s.challenges.GetActive(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return VerifyPasswordResult{}, ErrChallengeExpired
		}
		return VerifyPasswordResult{}, fmt.Errorf("get challenge: %w", err)
	}
	if challenge.Status != repo.ChallengeTOTPVerified {
		return VerifyPasswordResult{}, ErrChallengeStep
	}

	user, err := s.users.FindByID(ctx, challenge.AdminUserID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return VerifyPasswordResult{}, ErrUnauthorized
		}
		return VerifyPasswordResult{}, fmt.Errorf("find admin user: %w", err)
	}
	if isLocked(user.LockedUntil) {
		return VerifyPasswordResult{}, ErrAccountLocked
	}

	if err := security.CheckPassword(user.PasswordHash, password); err != nil {
		if err := s.applyFailure(ctx, user.ID); err != nil {
			return VerifyPasswordResult{}, err
		}
		return VerifyPasswordResult{}, ErrUnauthorized
	}

	if err := s.users.ResetFailures(ctx, user.ID); err != nil {
		return VerifyPasswordResult{}, fmt.Errorf("reset failures: %w", err)
	}

	now := time.Now().UTC()
	sessionID := uuid.New()
	sessionExpiresAt := now.Add(s.sessionMaxTTL)
	if err := s.sessions.Create(ctx, sessionID, user.ID, sessionExpiresAt, s.sessionIdleTTL, "", ""); err != nil {
		return VerifyPasswordResult{}, fmt.Errorf("create session: %w", err)
	}

	token, expiresAt, err := s.tokens.Issue(user.ID, user.TelegramID, user.Role, user.Username, sessionID.String())
	if err != nil {
		return VerifyPasswordResult{}, fmt.Errorf("issue jwt: %w", err)
	}

	if err := s.challenges.AdvanceStatus(ctx, challenge.ID, repo.ChallengeTOTPVerified, repo.ChallengeCompleted); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return VerifyPasswordResult{}, ErrChallengeExpired
		}
		return VerifyPasswordResult{}, fmt.Errorf("advance challenge step: %w", err)
	}

	_ = s.challenges.Expire(ctx, challenge.ID)

	return VerifyPasswordResult{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		Admin: AdminInfo{
			ID:          user.ID,
			TelegramID:  user.TelegramID,
			Username:    user.Username,
			DisplayName: user.DisplayName,
			Role:        user.Role,
		},
	}, nil
}

func (s *Service) StartTOTPSetup(ctx context.Context, telegramID int64, accountName string) (TOTPSetupStartResult, error) {
	if telegramID <= 0 {
		return TOTPSetupStartResult{}, ErrInvalidInput
	}

	user, err := s.users.FindByTelegramID(ctx, telegramID)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return TOTPSetupStartResult{}, ErrUnauthorized
		}
		return TOTPSetupStartResult{}, fmt.Errorf("find admin user by telegram id: %w", err)
	}
	if !user.IsActive {
		return TOTPSetupStartResult{}, ErrForbidden
	}

	accountName = strings.TrimSpace(accountName)
	if accountName == "" {
		if user.Username != "" {
			accountName = user.Username
		} else {
			accountName = fmt.Sprintf("telegram_%d", user.TelegramID)
		}
	}

	secret, otpURL, err := security.GenerateTOTPSecret(s.issuer, accountName)
	if err != nil {
		return TOTPSetupStartResult{}, fmt.Errorf("generate totp secret: %w", err)
	}

	setupToken, err := s.setupTokens.Create(ctx, user.ID, secret, s.totpSetupTTL)
	if err != nil {
		return TOTPSetupStartResult{}, fmt.Errorf("create totp setup token: %w", err)
	}

	qrDataURL, err := security.MakeQRCodeDataURL(otpURL, 256)
	if err != nil {
		return TOTPSetupStartResult{}, fmt.Errorf("generate qr code: %w", err)
	}

	return TOTPSetupStartResult{
		SetupID:       setupToken.ID.String(),
		TelegramID:    user.TelegramID,
		OTPAuthURL:    otpURL,
		Secret:        secret,
		QRCodeDataURL: qrDataURL,
		ExpiresAt:     setupToken.ExpiresAt,
	}, nil
}

func (s *Service) ConfirmTOTPSetup(ctx context.Context, setupID, code string) error {
	id, err := uuid.Parse(strings.TrimSpace(setupID))
	if err != nil {
		return ErrInvalidInput
	}

	token, err := s.setupTokens.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrChallengeExpired
		}
		return fmt.Errorf("get totp setup token: %w", err)
	}

	if !security.ValidateTOTP(token.Secret, code, time.Now().UTC()) {
		return ErrUnauthorized
	}

	encryptedSecret, err := s.secretCipher.Encrypt(token.Secret)
	if err != nil {
		return fmt.Errorf("encrypt totp secret: %w", err)
	}

	if err := s.users.EnableTOTP(ctx, token.AdminUserID, encryptedSecret); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrUnauthorized
		}
		return fmt.Errorf("enable totp: %w", err)
	}

	if err := s.setupTokens.Delete(ctx, token.ID); err != nil {
		return fmt.Errorf("delete setup token: %w", err)
	}
	return nil
}

func (s *Service) ValidateAccessToken(ctx context.Context, token string) (security.AdminClaims, error) {
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return security.AdminClaims{}, ErrUnauthorized
	}

	sessionID, err := uuid.Parse(strings.TrimSpace(claims.SID))
	if err != nil {
		return security.AdminClaims{}, ErrUnauthorized
	}

	if err := s.sessions.Touch(ctx, sessionID, claims.UserID, s.sessionIdleTTL); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return security.AdminClaims{}, ErrSessionExpired
		}
		return security.AdminClaims{}, fmt.Errorf("touch session: %w", err)
	}

	return claims, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	claims, err := s.tokens.Parse(token)
	if err != nil {
		return ErrUnauthorized
	}
	sessionID, err := uuid.Parse(strings.TrimSpace(claims.SID))
	if err != nil {
		return ErrUnauthorized
	}

	if err := s.sessions.Revoke(ctx, sessionID, claims.UserID); err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return ErrSessionExpired
		}
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func (s *Service) applyFailure(ctx context.Context, userID int64) error {
	locked, err := s.users.MarkFailure(ctx, userID, s.maxAttempts, time.Now().UTC().Add(s.lockDuration))
	if err != nil {
		return fmt.Errorf("mark login failure: %w", err)
	}
	if locked {
		return ErrAccountLocked
	}
	return nil
}

func isLocked(lockedUntil *time.Time) bool {
	return lockedUntil != nil && lockedUntil.After(time.Now().UTC())
}

func (s *Service) decryptTOTPSecret(stored string) (string, error) {
	if s.secretCipher == nil {
		return "", fmt.Errorf("secret cipher is not configured")
	}
	return s.secretCipher.Decrypt(stored)
}
