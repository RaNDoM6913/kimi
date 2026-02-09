package lookup

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"bot_moderator/internal/domain/enums"
	"bot_moderator/internal/domain/model"
	pgrepo "bot_moderator/internal/repo/postgres"
)

const signedURLTTL = 5 * time.Minute

var ErrUserNotFound = errors.New("user not found")

type Repo interface {
	FindUser(context.Context, string) (model.LookupUser, error)
	FindByUserID(context.Context, int64) (model.LookupUser, error)
	InsertAction(context.Context, model.BotLookupAction) error
	ForceReview(context.Context, int64) error
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

func (s *Service) FindUser(ctx context.Context, query string) (model.LookupUser, error) {
	if s.repo == nil {
		return model.LookupUser{}, ErrUserNotFound
	}

	user, err := s.repo.FindUser(ctx, query)
	if err != nil {
		if errors.Is(err, pgrepo.ErrLookupUserNotFound) {
			return model.LookupUser{}, ErrUserNotFound
		}
		return model.LookupUser{}, err
	}

	signedUser, err := s.attachSignedMedia(ctx, user)
	if err != nil {
		return model.LookupUser{}, err
	}
	return signedUser, nil
}

func (s *Service) GetByUserID(ctx context.Context, userID int64) (model.LookupUser, error) {
	if s.repo == nil {
		return model.LookupUser{}, ErrUserNotFound
	}

	user, err := s.repo.FindByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgrepo.ErrLookupUserNotFound) {
			return model.LookupUser{}, ErrUserNotFound
		}
		return model.LookupUser{}, err
	}

	signedUser, err := s.attachSignedMedia(ctx, user)
	if err != nil {
		return model.LookupUser{}, err
	}
	return signedUser, nil
}

func (s *Service) ForceReview(ctx context.Context, userID int64) error {
	if s.repo == nil {
		return nil
	}
	return s.repo.ForceReview(ctx, userID)
}

func (s *Service) LogAction(ctx context.Context, actorTGID int64, actorRole enums.Role, query string, foundUserID *int64, action string, payload map[string]interface{}) error {
	if s.repo == nil {
		return nil
	}

	rawPayload, err := json.Marshal(payload)
	if err != nil {
		rawPayload = json.RawMessage(`{}`)
	}

	entry := model.BotLookupAction{
		ActorTGID:   actorTGID,
		ActorRole:   string(actorRole),
		Query:       query,
		FoundUserID: foundUserID,
		Action:      strings.TrimSpace(strings.ToUpper(action)),
		Payload:     rawPayload,
		CreatedAt:   time.Now().UTC(),
	}
	return s.repo.InsertAction(ctx, entry)
}

func (s *Service) attachSignedMedia(ctx context.Context, user model.LookupUser) (model.LookupUser, error) {
	photoURLs := make([]string, 0, len(user.PhotoKeys))
	for _, key := range user.PhotoKeys {
		url, err := s.signKey(ctx, key)
		if err != nil {
			return model.LookupUser{}, err
		}
		if strings.TrimSpace(url) != "" {
			photoURLs = append(photoURLs, url)
		}
	}
	user.PhotoURLs = photoURLs

	circleURL, err := s.signKey(ctx, user.CircleKey)
	if err != nil {
		return model.LookupUser{}, err
	}
	user.CircleURL = circleURL
	return user, nil
}

func (s *Service) signKey(ctx context.Context, key string) (string, error) {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" || s.signer == nil {
		return "", nil
	}
	return s.signer.PresignGet(ctx, trimmed, signedURLTTL)
}
