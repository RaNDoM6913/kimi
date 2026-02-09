package botapp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"github.com/ivankudzin/tgapp/backend/internal/config"
	s3infra "github.com/ivankudzin/tgapp/backend/internal/infra/s3"
	tginfra "github.com/ivankudzin/tgapp/backend/internal/infra/telegram"
	"github.com/ivankudzin/tgapp/backend/internal/jobs/cleanup"
	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	mediasvc "github.com/ivankudzin/tgapp/backend/internal/services/media"
	modsvc "github.com/ivankudzin/tgapp/backend/internal/services/moderation"
)

const (
	missingSetupInstruction = "Сначала открой Mini App, заверши onboarding и укажи username в Telegram, затем отправь кружок снова."
	uploadedInstruction     = "Кружок получен и отправлен на модерацию."
	queueEmptyInstruction   = "Очередь модерации пуста."
)

type rejectState struct {
	ItemID      int64
	ModeratorID int64
	ReasonText  string
	AwaitFix    bool
}

type App struct {
	cfg               config.Config
	logger            *zap.Logger
	postgres          *pgxpool.Pool
	s3                *minio.Client
	bot               *tginfra.Bot
	storage           *mediasvc.S3Storage
	userRepo          *pgrepo.UserRepo
	mediaRepo         *pgrepo.MediaRepo
	moderationRepo    *pgrepo.ModerationRepo
	profileRepo       *pgrepo.ProfileRepo
	moderationService *modsvc.Service
	cleanupJob        *cleanup.Job

	rejectMu     sync.Mutex
	rejectByChat map[int64]rejectState
}

func New(ctx context.Context, cfg config.Config, logger *zap.Logger) (*App, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is nil")
	}

	pool, err := pgrepo.NewPool(ctx, cfg.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("init postgres for bot app: %w", err)
	}

	s3Client, err := s3infra.NewClient(s3infra.Config{
		Endpoint:  cfg.S3.Endpoint,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		UseSSL:    cfg.S3.UseSSL,
	})
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("init s3 for bot app: %w", err)
	}

	storage := mediasvc.NewS3Storage(s3Client, cfg.S3.Bucket)
	userRepo := pgrepo.NewUserRepo(pool)
	mediaRepo := pgrepo.NewMediaRepo(pool)
	moderationRepo := pgrepo.NewModerationRepo(pool)
	profileRepo := pgrepo.NewProfileRepo(pool)
	moderationService := modsvc.NewService(moderationRepo, profileRepo, mediaRepo, storage)
	cleanupJob := cleanup.NewCircleCleanupJob(mediaRepo, moderationRepo, storage, cfg.Bot.CircleRetention, logger)

	var bot *tginfra.Bot
	if strings.TrimSpace(cfg.Bot.Token) != "" {
		bot, err = tginfra.NewBot(cfg.Bot.Token)
		if err != nil {
			pool.Close()
			return nil, fmt.Errorf("init telegram bot: %w", err)
		}
	} else {
		logger.Warn("BOT_TOKEN is empty, video_note listener disabled")
	}

	return &App{
		cfg:               cfg,
		logger:            logger,
		postgres:          pool,
		s3:                s3Client,
		bot:               bot,
		storage:           storage,
		userRepo:          userRepo,
		mediaRepo:         mediaRepo,
		moderationRepo:    moderationRepo,
		profileRepo:       profileRepo,
		moderationService: moderationService,
		cleanupJob:        cleanupJob,
		rejectByChat:      make(map[int64]rejectState),
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("bot app started")

	errCh := make(chan error, 2)
	go func() {
		errCh <- a.runCleanupLoop(ctx)
	}()

	if a.bot != nil {
		go func() {
			errCh <- a.bot.Listen(ctx, tginfra.Handlers{
				OnVideoNote: a.handleVideoNote,
				OnCommand:   a.handleCommand,
				OnText:      a.handleText,
				OnCallback:  a.handleCallback,
			})
		}()
	}

	for {
		select {
		case <-ctx.Done():
			a.logger.Info("bot app stopped")
			return nil
		case err := <-errCh:
			if err == nil || errors.Is(err, context.Canceled) {
				continue
			}
			return err
		}
	}
}

func (a *App) runCleanupLoop(ctx context.Context) error {
	if a.cleanupJob == nil {
		return nil
	}

	interval := a.cfg.Bot.CleanupInterval
	if interval <= 0 {
		interval = 6 * time.Hour
	}

	if err := a.cleanupJob.Run(ctx); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := a.cleanupJob.Run(ctx); err != nil {
				return err
			}
		}
	}
}

func (a *App) handleVideoNote(ctx context.Context, update tginfra.VideoNoteUpdate) error {
	if a.bot == nil {
		return nil
	}

	user, err := a.userRepo.FindByTelegramID(ctx, update.UserID)
	if err != nil {
		if errors.Is(err, pgrepo.ErrUserNotFound) {
			return a.bot.SendText(ctx, update.ChatID, missingSetupInstruction)
		}
		return err
	}

	if strings.TrimSpace(user.Username) == "" && strings.TrimSpace(update.Username) == "" {
		return a.bot.SendText(ctx, update.ChatID, missingSetupInstruction)
	}

	if strings.TrimSpace(user.Username) == "" && strings.TrimSpace(update.Username) != "" {
		if err := a.userRepo.UpdateUsername(ctx, user.ID, update.Username); err != nil {
			a.logger.Warn("failed to persist telegram username", zap.Error(err), zap.Int64("user_id", user.ID))
		}
	}

	body, size, filename, contentType, err := a.bot.DownloadVideoNote(ctx, update.FileID)
	if err != nil {
		return err
	}
	defer body.Close()

	objectKey, err := buildCircleObjectKey(user.ID, filename)
	if err != nil {
		return err
	}

	if err := a.storage.EnsureBucket(ctx); err != nil {
		return err
	}
	if err := a.storage.PutPhoto(ctx, objectKey, body, size, contentType); err != nil {
		return err
	}

	circle, err := a.mediaRepo.CreateCircle(ctx, user.ID, objectKey)
	if err != nil {
		_ = a.storage.Delete(ctx, objectKey)
		return err
	}

	if err := a.moderationRepo.CreatePendingForMedia(ctx, user.ID, circle.ID); err != nil {
		_ = a.storage.Delete(ctx, objectKey)
		_ = a.mediaRepo.DeleteCircle(ctx, circle.ID)
		return err
	}

	if err := a.profileRepo.SetModerationStatus(ctx, user.ID, "PENDING"); err != nil {
		return err
	}

	return a.bot.SendText(ctx, update.ChatID, uploadedInstruction)
}

func (a *App) handleCommand(ctx context.Context, update tginfra.CommandUpdate) error {
	if a.bot == nil {
		return nil
	}

	switch strings.ToLower(strings.TrimSpace(update.Command)) {
	case "queue":
		return a.sendNextQueueItem(ctx, update.ChatID, update.UserID)
	default:
		return nil
	}
}

func (a *App) handleCallback(ctx context.Context, update tginfra.CallbackUpdate) error {
	if a.bot == nil {
		return nil
	}

	parts := strings.Split(strings.TrimSpace(update.Data), ":")
	if len(parts) != 3 || parts[0] != "mod" {
		return a.bot.AnswerCallback(ctx, update.CallbackID, "Unknown action")
	}

	itemID, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil || itemID <= 0 {
		return a.bot.AnswerCallback(ctx, update.CallbackID, "Invalid item id")
	}

	action := parts[1]
	switch action {
	case "approve":
		if err := a.moderationService.Approve(ctx, itemID, update.UserID); err != nil {
			return a.bot.AnswerCallback(ctx, update.CallbackID, "Approve failed")
		}
		if err := a.bot.AnswerCallback(ctx, update.CallbackID, "Approved"); err != nil {
			return err
		}
		return a.bot.SendText(ctx, update.ChatID, "Анкета одобрена.")
	case "reject":
		a.rejectMu.Lock()
		a.rejectByChat[update.ChatID] = rejectState{
			ItemID:      itemID,
			ModeratorID: update.UserID,
			AwaitFix:    false,
		}
		a.rejectMu.Unlock()
		if err := a.bot.AnswerCallback(ctx, update.CallbackID, "Send reason text"); err != nil {
			return err
		}
		return a.bot.SendText(ctx, update.ChatID, "Отправьте reason_text для отклонения.")
	default:
		return a.bot.AnswerCallback(ctx, update.CallbackID, "Unknown action")
	}
}

func (a *App) handleText(ctx context.Context, update tginfra.TextUpdate) error {
	if a.bot == nil {
		return nil
	}

	a.rejectMu.Lock()
	state, ok := a.rejectByChat[update.ChatID]
	a.rejectMu.Unlock()
	if !ok {
		return nil
	}
	if state.ModeratorID != update.UserID {
		return nil
	}

	text := strings.TrimSpace(update.Text)
	if text == "" {
		return a.bot.SendText(ctx, update.ChatID, "Текст не может быть пустым.")
	}

	if !state.AwaitFix {
		state.ReasonText = text
		state.AwaitFix = true
		a.rejectMu.Lock()
		a.rejectByChat[update.ChatID] = state
		a.rejectMu.Unlock()
		return a.bot.SendText(ctx, update.ChatID, "Теперь отправьте required_fix_step.")
	}

	if err := a.moderationService.Reject(ctx, state.ItemID, state.ModeratorID, state.ReasonText, text); err != nil {
		return a.bot.SendText(ctx, update.ChatID, "Не удалось отклонить анкету.")
	}

	a.rejectMu.Lock()
	delete(a.rejectByChat, update.ChatID)
	a.rejectMu.Unlock()

	return a.bot.SendText(ctx, update.ChatID, "Анкета отклонена.")
}

func (a *App) sendNextQueueItem(ctx context.Context, chatID int64, actorTGID int64) error {
	if a.bot == nil {
		return nil
	}

	item, err := a.moderationService.GetNextQueueItem(ctx, actorTGID)
	if err != nil {
		if errors.Is(err, modsvc.ErrQueueEmpty) {
			return a.bot.SendText(ctx, chatID, queueEmptyInstruction)
		}
		return err
	}

	text := formatQueueMessage(item)
	return a.bot.SendModerationQueue(ctx, chatID, text, item.ItemID)
}

func formatQueueMessage(item modsvc.QueueItem) string {
	lines := []string{
		fmt.Sprintf("Queue item #%d", item.ItemID),
		fmt.Sprintf("User ID: %d", item.UserID),
		fmt.Sprintf("Queue size: %d", item.QueueSize),
		fmt.Sprintf("ETA bucket: %s", item.ETABucket),
		"",
		"Profile:",
		fmt.Sprintf("- display_name: %s", defaultString(item.Profile.DisplayName, "-")),
		fmt.Sprintf("- city_id: %s", defaultString(item.Profile.CityID, "-")),
		fmt.Sprintf("- gender: %s", defaultString(item.Profile.Gender, "-")),
		fmt.Sprintf("- looking_for: %s", defaultString(item.Profile.LookingFor, "-")),
		fmt.Sprintf("- occupation: %s", defaultString(item.Profile.Occupation, "-")),
		fmt.Sprintf("- education: %s", defaultString(item.Profile.Education, "-")),
		fmt.Sprintf("- goals: %s", strings.Join(item.Profile.Goals, ", ")),
		"",
		"Photos:",
	}

	if len(item.PhotoURLs) == 0 {
		lines = append(lines, "- none")
	} else {
		for i, u := range item.PhotoURLs {
			lines = append(lines, fmt.Sprintf("- photo_%d: %s", i+1, u))
		}
	}

	lines = append(lines, "", "Circle:")
	if item.CircleURL == nil || strings.TrimSpace(*item.CircleURL) == "" {
		lines = append(lines, "- none")
	} else {
		lines = append(lines, "- "+*item.CircleURL)
	}

	return strings.Join(lines, "\n")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (a *App) Close() {
	if a.postgres != nil {
		a.postgres.Close()
	}
	_ = a.s3
}
