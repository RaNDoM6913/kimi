package moderation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"bot_moderator/internal/domain/enums"
	"bot_moderator/internal/domain/model"
	pgrepo "bot_moderator/internal/repo/postgres"
)

var ErrQueueEmpty = errors.New("moderation queue is empty")

const signedURLTTL = 5 * time.Minute

type Repo interface {
	AcquireNextPending(context.Context, int64, time.Duration) (model.ModerationItem, error)
	GetProfile(context.Context, int64) (model.ModerationProfile, error)
	ListPhotoKeys(context.Context, int64, int) ([]string, error)
	GetLatestCircleKey(context.Context, int64) (string, error)
	GetByID(context.Context, int64) (model.ModerationItem, error)
	MarkApproved(context.Context, int64) error
	MarkRejected(context.Context, int64, string, string, string) error
	InsertModerationAction(context.Context, model.BotModerationAction) error
}

type URLSigner interface {
	PresignGet(context.Context, string, time.Duration) (string, error)
}

type Service struct {
	repo   Repo
	signer URLSigner
}

func NewService(repo Repo, signer URLSigner) *Service {
	return &Service{repo: repo, signer: signer}
}

func (s *Service) AcquireNextPending(ctx context.Context, actorTGID int64) (model.ModerationQueueItem, error) {
	if s.repo == nil {
		return model.ModerationQueueItem{}, ErrQueueEmpty
	}

	item, err := s.repo.AcquireNextPending(ctx, actorTGID, 10*time.Minute)
	if err != nil {
		if errors.Is(err, pgrepo.ErrModerationQueueEmpty) {
			return model.ModerationQueueItem{}, ErrQueueEmpty
		}
		return model.ModerationQueueItem{}, err
	}

	profile, err := s.repo.GetProfile(ctx, item.UserID)
	if err != nil {
		return model.ModerationQueueItem{}, err
	}

	photoKeys, err := s.repo.ListPhotoKeys(ctx, item.UserID, 3)
	if err != nil {
		return model.ModerationQueueItem{}, err
	}

	circleKey, err := s.repo.GetLatestCircleKey(ctx, item.UserID)
	if err != nil {
		return model.ModerationQueueItem{}, err
	}

	lockedAt := time.Now().UTC()
	if item.LockedAt != nil {
		lockedAt = item.LockedAt.UTC()
	}

	photoURLs := make([]string, 0, len(photoKeys))
	for _, key := range photoKeys {
		url, signErr := s.signKey(ctx, key)
		if signErr != nil {
			return model.ModerationQueueItem{}, signErr
		}
		if strings.TrimSpace(url) != "" {
			photoURLs = append(photoURLs, url)
		}
	}

	circleURL, err := s.signKey(ctx, circleKey)
	if err != nil {
		return model.ModerationQueueItem{}, err
	}

	return model.ModerationQueueItem{
		ModerationItemID: item.ID,
		TargetUserID:     item.UserID,
		ETABucket:        item.ETABucket,
		CreatedAt:        item.CreatedAt,
		LockedAt:         lockedAt,
		Profile:          profile,
		PhotoURLs:        photoURLs,
		CircleURL:        circleURL,
	}, nil
}

func (s *Service) signKey(ctx context.Context, key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return "", nil
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		return trimmed, nil
	}
	if s.signer == nil {
		return "", fmt.Errorf("moderation url signer is not configured")
	}
	return s.signer.PresignGet(ctx, trimmed, signedURLTTL)
}

type ApproveInput struct {
	ActorTGID        int64
	ActorRole        enums.Role
	ModerationItemID int64
}

type ApproveResult struct {
	TargetUserID     int64
	ModerationItemID int64
	DurationSec      *int
}

func (s *Service) Approve(ctx context.Context, input ApproveInput) (ApproveResult, error) {
	if s.repo == nil {
		return ApproveResult{}, fmt.Errorf("moderation repo is not configured")
	}
	if input.ActorTGID == 0 {
		return ApproveResult{}, fmt.Errorf("invalid actor tg id")
	}
	if input.ModerationItemID <= 0 {
		return ApproveResult{}, fmt.Errorf("invalid moderation item id")
	}

	item, err := s.repo.GetByID(ctx, input.ModerationItemID)
	if err != nil {
		return ApproveResult{}, err
	}

	if err := s.repo.MarkApproved(ctx, input.ModerationItemID); err != nil {
		return ApproveResult{}, err
	}

	durationSec := calculateDurationSec(item.LockedAt)
	action := model.BotModerationAction{
		ActorTGID:        input.ActorTGID,
		ActorRole:        string(input.ActorRole),
		TargetUserID:     item.UserID,
		ModerationItemID: input.ModerationItemID,
		Decision:         "APPROVE",
		DurationSec:      durationSec,
		CreatedAt:        time.Now().UTC(),
	}
	if err := s.repo.InsertModerationAction(ctx, action); err != nil {
		return ApproveResult{}, err
	}

	return ApproveResult{
		TargetUserID:     item.UserID,
		ModerationItemID: input.ModerationItemID,
		DurationSec:      durationSec,
	}, nil
}

type RejectInput struct {
	ActorTGID        int64
	ActorRole        enums.Role
	ModerationItemID int64
	ReasonCode       string
	Comment          string
}

type RejectResult struct {
	TargetUserID     int64
	ModerationItemID int64
	ReasonCode       string
	DurationSec      *int
}

type rejectTemplate struct {
	reasonText      string
	requiredFixStep string
}

var rejectTemplates = map[string]rejectTemplate{
	"PHOTO_NO_FACE": {
		reasonText:      "На фото не видно лица.",
		requiredFixStep: "Загрузите фото, где лицо хорошо различимо.",
	},
	"PHOTO_FAKE_NOT_YOU": {
		reasonText:      "Фото не соответствует владельцу анкеты.",
		requiredFixStep: "Загрузите ваши реальные фотографии без чужих изображений.",
	},
	"PHOTO_PROHIBITED": {
		reasonText:      "Обнаружен запрещенный фото-контент.",
		requiredFixStep: "Удалите запрещённый контент и загрузите новые фото.",
	},
	"CIRCLE_MISMATCH": {
		reasonText:      "Кружок не совпадает с фотографиями анкеты.",
		requiredFixStep: "Перезапишите кружок, чтобы внешность совпадала с фото.",
	},
	"CIRCLE_FAILED": {
		reasonText:      "Кружок не прошел проверку качества.",
		requiredFixStep: "Перезапишите кружок при хорошем освещении и стабильной связи.",
	},
	"PROFILE_INCOMPLETE": {
		reasonText:      "Профиль заполнен не полностью.",
		requiredFixStep: "Заполните обязательные поля профиля и отправьте на модерацию повторно.",
	},
	"SPAM_ADS_LINKS": {
		reasonText:      "Обнаружены признаки спама, рекламы или внешних ссылок.",
		requiredFixStep: "Удалите спам/рекламу/ссылки из профиля и отправьте на модерацию повторно.",
	},
	"BOT_SUSPECT": {
		reasonText:      "Профиль помечен как подозрительный на автоматизацию.",
		requiredFixStep: "Обновите анкету и пройдите повторную модерацию вручную.",
	},
	"OTHER": {
		reasonText:      "Требуется корректировка анкеты.",
		requiredFixStep: "Обновите анкету по замечанию модератора и отправьте на модерацию повторно.",
	},
}

func (s *Service) Reject(ctx context.Context, input RejectInput) (RejectResult, error) {
	if s.repo == nil {
		return RejectResult{}, fmt.Errorf("moderation repo is not configured")
	}
	if input.ActorTGID == 0 {
		return RejectResult{}, fmt.Errorf("invalid actor tg id")
	}
	if input.ModerationItemID <= 0 {
		return RejectResult{}, fmt.Errorf("invalid moderation item id")
	}

	reasonCode := normalizeReasonCode(input.ReasonCode)
	tpl, ok := rejectTemplates[reasonCode]
	if !ok {
		return RejectResult{}, fmt.Errorf("unsupported reason code")
	}

	item, err := s.repo.GetByID(ctx, input.ModerationItemID)
	if err != nil {
		return RejectResult{}, err
	}

	reasonText := tpl.reasonText
	requiredFixStep := tpl.requiredFixStep
	comment := strings.TrimSpace(input.Comment)
	if comment != "" {
		reasonText = fmt.Sprintf("%s Комментарий модератора: %s", reasonText, comment)
		requiredFixStep = fmt.Sprintf("%s Комментарий модератора: %s", requiredFixStep, comment)
	}

	if err := s.repo.MarkRejected(ctx, input.ModerationItemID, reasonCode, reasonText, requiredFixStep); err != nil {
		return RejectResult{}, err
	}

	durationSec := calculateDurationSec(item.LockedAt)

	action := model.BotModerationAction{
		ActorTGID:        input.ActorTGID,
		ActorRole:        string(input.ActorRole),
		TargetUserID:     item.UserID,
		ModerationItemID: input.ModerationItemID,
		Decision:         "REJECT",
		ReasonCode:       reasonCode,
		DurationSec:      durationSec,
		CreatedAt:        time.Now().UTC(),
	}
	if err := s.repo.InsertModerationAction(ctx, action); err != nil {
		return RejectResult{}, err
	}

	return RejectResult{
		TargetUserID:     item.UserID,
		ModerationItemID: input.ModerationItemID,
		ReasonCode:       reasonCode,
		DurationSec:      durationSec,
	}, nil
}

func normalizeReasonCode(raw string) string {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if code == "" {
		return "OTHER"
	}
	return code
}

func calculateDurationSec(lockedAt *time.Time) *int {
	if lockedAt == nil {
		return nil
	}

	duration := int(time.Since(lockedAt.UTC()).Seconds())
	if duration < 0 {
		duration = 0
	}
	return &duration
}
