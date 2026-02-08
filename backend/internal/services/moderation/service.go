package moderation

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
)

const signedURLTTL = 5 * time.Minute

var ErrQueueEmpty = errors.New("moderation queue is empty")

type URLSigner interface {
	PresignGet(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type Service struct {
	moderationRepo *pgrepo.ModerationRepo
	profileRepo    *pgrepo.ProfileRepo
	mediaRepo      *pgrepo.MediaRepo
	signer         URLSigner
}

type UserStatus struct {
	Status          string
	ReasonText      *string
	RequiredFixStep *string
	ETABucket       string
}

type QueueItem struct {
	ItemID    int64
	UserID    int64
	QueueSize int
	ETABucket string
	Profile   pgrepo.ProfileQueueSummary
	PhotoURLs []string
	CircleURL *string
	CreatedAt time.Time
}

func NewService(moderationRepo *pgrepo.ModerationRepo, profileRepo *pgrepo.ProfileRepo, mediaRepo *pgrepo.MediaRepo, signer URLSigner) *Service {
	return &Service{
		moderationRepo: moderationRepo,
		profileRepo:    profileRepo,
		mediaRepo:      mediaRepo,
		signer:         signer,
	}
}

func (s *Service) GetUserStatus(ctx context.Context, userID int64) (UserStatus, error) {
	if userID <= 0 {
		return UserStatus{}, fmt.Errorf("invalid user id")
	}
	if s.moderationRepo == nil || s.profileRepo == nil {
		return UserStatus{}, fmt.Errorf("moderation service dependencies are not configured")
	}

	item, err := s.moderationRepo.GetLatestByUser(ctx, userID)
	if err != nil && !errors.Is(err, pgrepo.ErrModerationItemNotFound) {
		return UserStatus{}, err
	}

	if errors.Is(err, pgrepo.ErrModerationItemNotFound) {
		snapshot, profileErr := s.profileRepo.GetModerationSnapshot(ctx, userID)
		if profileErr != nil && !errors.Is(profileErr, pgrepo.ErrProfileNotFound) {
			return UserStatus{}, profileErr
		}

		status := "PENDING"
		if !errors.Is(profileErr, pgrepo.ErrProfileNotFound) && strings.TrimSpace(snapshot.Status) != "" {
			status = strings.ToUpper(snapshot.Status)
		}

		pendingCount, countErr := s.moderationRepo.CountPending(ctx)
		if countErr != nil {
			return UserStatus{}, countErr
		}
		eta := ETABucketFromQueueSize(pendingCount)

		return UserStatus{Status: status, ETABucket: eta}, nil
	}

	status := strings.ToUpper(strings.TrimSpace(item.Status))
	if status == "" {
		status = "PENDING"
	}

	eta := strings.TrimSpace(item.ETABucket)
	if status == "PENDING" {
		pendingCount, countErr := s.moderationRepo.CountPending(ctx)
		if countErr != nil {
			return UserStatus{}, countErr
		}
		eta = ETABucketFromQueueSize(pendingCount)
		_ = s.moderationRepo.UpdateETABucket(ctx, item.ID, eta)
	}

	if eta == "" {
		eta = "up_to_10"
	}

	return UserStatus{
		Status:          status,
		ReasonText:      item.ReasonText,
		RequiredFixStep: item.RequiredFixStep,
		ETABucket:       eta,
	}, nil
}

func (s *Service) GetNextQueueItem(ctx context.Context) (QueueItem, error) {
	if s.moderationRepo == nil || s.profileRepo == nil || s.mediaRepo == nil {
		return QueueItem{}, fmt.Errorf("moderation service dependencies are not configured")
	}

	item, err := s.moderationRepo.GetNextPending(ctx)
	if err != nil {
		if errors.Is(err, pgrepo.ErrModerationItemNotFound) {
			return QueueItem{}, ErrQueueEmpty
		}
		return QueueItem{}, err
	}

	queueSize, err := s.moderationRepo.CountPending(ctx)
	if err != nil {
		return QueueItem{}, err
	}

	etaBucket := ETABucketFromQueueSize(queueSize)
	_ = s.moderationRepo.UpdateETABucket(ctx, item.ID, etaBucket)

	profile, err := s.profileRepo.GetQueueSummary(ctx, item.UserID)
	if err != nil {
		return QueueItem{}, err
	}

	photos, err := s.mediaRepo.ListUserPhotos(ctx, item.UserID, 3)
	if err != nil {
		return QueueItem{}, err
	}

	photoURLs := make([]string, 0, len(photos))
	for _, photo := range photos {
		url, signErr := s.signKey(ctx, photo.ObjectKey)
		if signErr != nil {
			return QueueItem{}, signErr
		}
		photoURLs = append(photoURLs, url)
	}

	var circleURL *string
	circle, err := s.mediaRepo.GetLatestCircle(ctx, item.UserID)
	if err != nil {
		return QueueItem{}, err
	}
	if circle != nil {
		url, signErr := s.signKey(ctx, circle.ObjectKey)
		if signErr != nil {
			return QueueItem{}, signErr
		}
		circleURL = &url
	}

	return QueueItem{
		ItemID:    item.ID,
		UserID:    item.UserID,
		QueueSize: queueSize,
		ETABucket: etaBucket,
		Profile:   profile,
		PhotoURLs: photoURLs,
		CircleURL: circleURL,
		CreatedAt: item.CreatedAt,
	}, nil
}

func (s *Service) Approve(ctx context.Context, itemID int64, moderatorTGID int64) error {
	if itemID <= 0 {
		return fmt.Errorf("invalid moderation item id")
	}
	if s.moderationRepo == nil || s.profileRepo == nil {
		return fmt.Errorf("moderation service dependencies are not configured")
	}

	item, err := s.moderationRepo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}

	queueSize, err := s.moderationRepo.CountPending(ctx)
	if err != nil {
		return err
	}
	etaBucket := ETABucketFromQueueSize(queueSize)

	if err := s.moderationRepo.MarkApproved(ctx, itemID, moderatorTGID, etaBucket); err != nil {
		return err
	}

	if err := s.profileRepo.ApplyModerationDecision(ctx, item.UserID, "APPROVED", true); err != nil {
		return err
	}

	return nil
}

func (s *Service) Reject(ctx context.Context, itemID int64, moderatorTGID int64, reasonText, requiredFixStep string) error {
	if itemID <= 0 {
		return fmt.Errorf("invalid moderation item id")
	}
	if strings.TrimSpace(reasonText) == "" || strings.TrimSpace(requiredFixStep) == "" {
		return fmt.Errorf("reason_text and required_fix_step are required")
	}
	if s.moderationRepo == nil || s.profileRepo == nil {
		return fmt.Errorf("moderation service dependencies are not configured")
	}

	item, err := s.moderationRepo.GetByID(ctx, itemID)
	if err != nil {
		return err
	}

	queueSize, err := s.moderationRepo.CountPending(ctx)
	if err != nil {
		return err
	}
	etaBucket := ETABucketFromQueueSize(queueSize)

	if err := s.moderationRepo.MarkRejected(ctx, itemID, moderatorTGID, reasonText, requiredFixStep, etaBucket); err != nil {
		return err
	}

	if err := s.profileRepo.ApplyModerationDecision(ctx, item.UserID, "REJECTED", false); err != nil {
		return err
	}

	return nil
}

func ETABucketFromQueueSize(queueSize int) string {
	if queueSize >= 50 {
		return "more_than_hour"
	}
	if queueSize <= 10 {
		return "up_to_10"
	}
	if queueSize <= 20 {
		return "up_to_20"
	}
	if queueSize <= 30 {
		return "up_to_30"
	}
	if queueSize <= 40 {
		return "up_to_40"
	}
	return "up_to_50"
}

func (s *Service) signKey(ctx context.Context, key string) (string, error) {
	if strings.TrimSpace(key) == "" {
		return "", nil
	}
	if s.signer == nil {
		return "", fmt.Errorf("moderation url signer is not configured")
	}
	url, err := s.signer.PresignGet(ctx, key, signedURLTTL)
	if err != nil {
		return "", fmt.Errorf("sign media key: %w", err)
	}
	return url, nil
}
