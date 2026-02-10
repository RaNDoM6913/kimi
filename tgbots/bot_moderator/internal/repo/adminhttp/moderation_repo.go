package adminhttp

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"bot_moderator/internal/domain/model"
	"bot_moderator/internal/repo/postgres"
)

type moderationCacheEntry struct {
	Item     model.ModerationItem
	Profile  model.ModerationProfile
	PhotoRef []string
	Circle   string
}

type ModerationRepo struct {
	client *Client
	db     *postgres.ModerationRepo
	dual   bool

	cacheMu        sync.RWMutex
	cacheByItemID  map[int64]moderationCacheEntry
	lastItemByUser map[int64]int64
}

func NewModerationRepo(client *Client, db *postgres.ModerationRepo, dual bool) *ModerationRepo {
	return &ModerationRepo{
		client:         client,
		db:             db,
		dual:           dual,
		cacheByItemID:  make(map[int64]moderationCacheEntry),
		lastItemByUser: make(map[int64]int64),
	}
}

func (r *ModerationRepo) AcquireNextPending(ctx context.Context, actorTGID int64, lockDuration time.Duration) (model.ModerationItem, error) {
	if lockDuration <= 0 {
		lockDuration = 10 * time.Minute
	}

	request := map[string]interface{}{
		"actor_tg_id":       actorTGID,
		"lock_duration_sec": int64(lockDuration / time.Second),
	}

	response := moderationAcquireResponseDTO{}
	err := r.client.DoJSON(ctx, http.MethodPost, "/admin/bot/mod/queue/acquire", request, &response)
	if shouldFallbackModeration(r.dual, err) && r.db != nil {
		return r.db.AcquireNextPending(ctx, actorTGID, lockDuration)
	}
	if err != nil {
		return model.ModerationItem{}, err
	}

	entry := response.toCacheEntry()
	if entry.Item.ID == 0 {
		return model.ModerationItem{}, &RequestError{
			Op:           "decode moderation acquire response",
			Fallbackable: false,
			Err:          errors.New("missing moderation_item.id"),
		}
	}
	if entry.Item.UserID == 0 {
		return model.ModerationItem{}, &RequestError{
			Op:           "decode moderation acquire response",
			Fallbackable: false,
			Err:          errors.New("missing moderation_item.user_id"),
		}
	}
	if entry.Item.LockedAt == nil {
		now := time.Now().UTC()
		entry.Item.LockedAt = &now
	}
	if entry.Item.LockedByTGID == nil && actorTGID != 0 {
		lockedBy := actorTGID
		entry.Item.LockedByTGID = &lockedBy
	}
	if entry.Profile.UserID == 0 {
		entry.Profile.UserID = entry.Item.UserID
	}

	r.putCache(entry)
	return entry.Item, nil
}

func (r *ModerationRepo) GetProfile(ctx context.Context, userID int64) (model.ModerationProfile, error) {
	if cached, ok := r.getCacheByUserID(userID); ok {
		profile := cached.Profile
		if profile.UserID == 0 {
			profile.UserID = userID
		}
		return profile, nil
	}

	if r.dual && r.db != nil {
		return r.db.GetProfile(ctx, userID)
	}
	return model.ModerationProfile{UserID: userID}, nil
}

func (r *ModerationRepo) ListPhotoKeys(ctx context.Context, userID int64, limit int) ([]string, error) {
	if cached, ok := r.getCacheByUserID(userID); ok {
		refs := cloneStrings(cached.PhotoRef)
		if limit > 0 && len(refs) > limit {
			refs = refs[:limit]
		}
		return refs, nil
	}

	if r.dual && r.db != nil {
		return r.db.ListPhotoKeys(ctx, userID, limit)
	}
	return []string{}, nil
}

func (r *ModerationRepo) GetLatestCircleKey(ctx context.Context, userID int64) (string, error) {
	if cached, ok := r.getCacheByUserID(userID); ok {
		return strings.TrimSpace(cached.Circle), nil
	}

	if r.dual && r.db != nil {
		return r.db.GetLatestCircleKey(ctx, userID)
	}
	return "", nil
}

func (r *ModerationRepo) GetByID(ctx context.Context, moderationItemID int64) (model.ModerationItem, error) {
	if cached, ok := r.getCacheByItemID(moderationItemID); ok {
		return cached.Item, nil
	}

	response := moderationAcquireResponseDTO{}
	err := r.client.DoJSON(ctx, http.MethodGet, "/admin/bot/mod/items/"+int64ToString(moderationItemID), nil, &response)
	if shouldFallbackModeration(r.dual, err) && r.db != nil {
		return r.db.GetByID(ctx, moderationItemID)
	}
	if err != nil {
		var reqErr *RequestError
		if errors.As(err, &reqErr) && reqErr.StatusCode == http.StatusNotFound {
			return model.ModerationItem{}, postgres.ErrModerationItemNotFound
		}
		return model.ModerationItem{}, err
	}

	entry := response.toCacheEntry()
	if entry.Item.ID == 0 {
		entry.Item.ID = moderationItemID
	}
	if entry.Item.ID == 0 {
		return model.ModerationItem{}, postgres.ErrModerationItemNotFound
	}
	if entry.Item.UserID != 0 {
		r.putCache(entry)
	}
	return entry.Item, nil
}

func (r *ModerationRepo) MarkApproved(ctx context.Context, moderationItemID int64) error {
	request := map[string]interface{}{
		"moderation_item_id": moderationItemID,
	}
	if actorTGID := ActorTGIDFromContext(ctx); actorTGID != 0 {
		request["actor_tg_id"] = actorTGID
	}

	err := r.client.DoJSON(
		ctx,
		http.MethodPost,
		"/admin/bot/mod/items/"+int64ToString(moderationItemID)+"/approve",
		request,
		nil,
	)
	if shouldFallbackModeration(r.dual, err) && r.db != nil {
		return r.db.MarkApproved(ctx, moderationItemID)
	}
	if err != nil {
		return err
	}

	r.dropCacheByItemID(moderationItemID)
	return nil
}

func (r *ModerationRepo) MarkRejected(ctx context.Context, moderationItemID int64, reasonCode string, reasonText string, requiredFixStep string) error {
	request := map[string]interface{}{
		"moderation_item_id": moderationItemID,
		"reason_code":        strings.TrimSpace(reasonCode),
		"reason_text":        strings.TrimSpace(reasonText),
		"required_fix_step":  strings.TrimSpace(requiredFixStep),
	}
	if actorTGID := ActorTGIDFromContext(ctx); actorTGID != 0 {
		request["actor_tg_id"] = actorTGID
	}

	err := r.client.DoJSON(
		ctx,
		http.MethodPost,
		"/admin/bot/mod/items/"+int64ToString(moderationItemID)+"/reject",
		request,
		nil,
	)
	if shouldFallbackModeration(r.dual, err) && r.db != nil {
		return r.db.MarkRejected(ctx, moderationItemID, reasonCode, reasonText, requiredFixStep)
	}
	if err != nil {
		return err
	}

	r.dropCacheByItemID(moderationItemID)
	return nil
}

func (r *ModerationRepo) InsertModerationAction(ctx context.Context, action model.BotModerationAction) error {
	if r.dual && r.db != nil {
		return r.db.InsertModerationAction(ctx, action)
	}
	return nil
}

func shouldFallbackModeration(dual bool, err error) bool {
	if !dual || err == nil {
		return false
	}
	if IsFallbackable(err) {
		return true
	}

	var reqErr *RequestError
	if errors.As(err, &reqErr) && reqErr.StatusCode == http.StatusNotFound {
		return true
	}
	return false
}

func (r *ModerationRepo) putCache(entry moderationCacheEntry) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	r.cacheByItemID[entry.Item.ID] = moderationCacheEntry{
		Item:     entry.Item,
		Profile:  entry.Profile,
		PhotoRef: cloneStrings(entry.PhotoRef),
		Circle:   entry.Circle,
	}
	r.lastItemByUser[entry.Item.UserID] = entry.Item.ID
}

func (r *ModerationRepo) getCacheByItemID(itemID int64) (moderationCacheEntry, bool) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()
	entry, ok := r.cacheByItemID[itemID]
	if !ok {
		return moderationCacheEntry{}, false
	}
	entry.PhotoRef = cloneStrings(entry.PhotoRef)
	return entry, true
}

func (r *ModerationRepo) getCacheByUserID(userID int64) (moderationCacheEntry, bool) {
	r.cacheMu.RLock()
	defer r.cacheMu.RUnlock()

	itemID, ok := r.lastItemByUser[userID]
	if !ok {
		return moderationCacheEntry{}, false
	}
	entry, ok := r.cacheByItemID[itemID]
	if !ok {
		return moderationCacheEntry{}, false
	}
	entry.PhotoRef = cloneStrings(entry.PhotoRef)
	return entry, true
}

func (r *ModerationRepo) dropCacheByItemID(itemID int64) {
	r.cacheMu.Lock()
	defer r.cacheMu.Unlock()

	entry, ok := r.cacheByItemID[itemID]
	if ok {
		delete(r.lastItemByUser, entry.Item.UserID)
	}
	delete(r.cacheByItemID, itemID)
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	cloned := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		cloned = append(cloned, trimmed)
	}
	return cloned
}

type moderationAcquireResponseDTO struct {
	ModerationItem moderationItemDTO    `json:"moderation_item"`
	Item           moderationItemDTO    `json:"item"`
	Profile        moderationProfileDTO `json:"profile"`
	Media          moderationMediaDTO   `json:"media"`

	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	Status        string     `json:"status"`
	ETABucket     string     `json:"eta_bucket"`
	CreatedAt     time.Time  `json:"created_at"`
	LockedAt      *time.Time `json:"locked_at"`
	LockedUntil   *time.Time `json:"locked_until"`
	UpdatedAt     time.Time  `json:"updated_at"`
	TargetType    string     `json:"target_type"`
	TargetID      *int64     `json:"target_id"`
	ModeratorTGID *int64     `json:"moderator_tg_id"`
	LockedByTGID  *int64     `json:"locked_by_tg_id"`

	PhotoKeys          []string `json:"photo_keys"`
	PhotoURLs          []string `json:"photo_urls"`
	PresignedPhotoURLs []string `json:"presigned_photo_urls"`
	CircleKey          string   `json:"circle_key"`
	CircleURL          string   `json:"circle_url"`
	PresignedCircleURL string   `json:"presigned_circle_url"`
}

func (dto moderationAcquireResponseDTO) toCacheEntry() moderationCacheEntry {
	itemDTO := dto.pickItemDTO()
	item := itemDTO.toModel()

	profile := dto.Profile.toModel()
	if profile.UserID == 0 {
		profile.UserID = item.UserID
	}

	media := dto.Media
	media.PhotoKeys = append(media.PhotoKeys, dto.PhotoKeys...)
	media.PhotoURLs = append(media.PhotoURLs, dto.PhotoURLs...)
	media.PresignedPhotoURLs = append(media.PresignedPhotoURLs, dto.PresignedPhotoURLs...)
	if strings.TrimSpace(media.CircleKey) == "" {
		media.CircleKey = dto.CircleKey
	}
	if strings.TrimSpace(media.CircleURL) == "" {
		media.CircleURL = dto.CircleURL
	}
	if strings.TrimSpace(media.PresignedCircleURL) == "" {
		media.PresignedCircleURL = dto.PresignedCircleURL
	}

	return moderationCacheEntry{
		Item:     item,
		Profile:  profile,
		PhotoRef: media.pickPhotoRefs(),
		Circle:   media.pickCircleRef(),
	}
}

func (dto moderationAcquireResponseDTO) pickItemDTO() moderationItemDTO {
	if dto.ModerationItem.ID != 0 || dto.ModerationItem.UserID != 0 {
		return dto.ModerationItem
	}
	if dto.Item.ID != 0 || dto.Item.UserID != 0 {
		return dto.Item
	}
	return moderationItemDTO{
		ID:            dto.ID,
		UserID:        dto.UserID,
		Status:        dto.Status,
		ETABucket:     dto.ETABucket,
		CreatedAt:     dto.CreatedAt,
		LockedAt:      dto.LockedAt,
		LockedUntil:   dto.LockedUntil,
		UpdatedAt:     dto.UpdatedAt,
		TargetType:    dto.TargetType,
		TargetID:      dto.TargetID,
		ModeratorTGID: dto.ModeratorTGID,
		LockedByTGID:  dto.LockedByTGID,
	}
}

type moderationItemDTO struct {
	ID            int64      `json:"id"`
	UserID        int64      `json:"user_id"`
	Status        string     `json:"status"`
	ETABucket     string     `json:"eta_bucket"`
	CreatedAt     time.Time  `json:"created_at"`
	LockedAt      *time.Time `json:"locked_at"`
	LockedUntil   *time.Time `json:"locked_until"`
	UpdatedAt     time.Time  `json:"updated_at"`
	TargetType    string     `json:"target_type"`
	TargetID      *int64     `json:"target_id"`
	ModeratorTGID *int64     `json:"moderator_tg_id"`
	LockedByTGID  *int64     `json:"locked_by_tg_id"`
}

func (dto moderationItemDTO) toModel() model.ModerationItem {
	return model.ModerationItem{
		ID:            dto.ID,
		UserID:        dto.UserID,
		Status:        model.ModerationStatus(strings.ToUpper(strings.TrimSpace(dto.Status))),
		ETABucket:     strings.TrimSpace(dto.ETABucket),
		CreatedAt:     dto.CreatedAt,
		LockedByTGID:  dto.LockedByTGID,
		LockedAt:      dto.LockedAt,
		LockedUntil:   dto.LockedUntil,
		UpdatedAt:     dto.UpdatedAt,
		TargetType:    strings.TrimSpace(dto.TargetType),
		TargetID:      dto.TargetID,
		ModeratorTGID: dto.ModeratorTGID,
	}
}

type moderationProfileDTO struct {
	UserID      int64      `json:"user_id"`
	TGID        int64      `json:"tg_id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"display_name"`
	CityID      string     `json:"city_id"`
	Birthdate   *time.Time `json:"birthdate"`
	Age         int        `json:"age"`
	Gender      string     `json:"gender"`
	LookingFor  string     `json:"looking_for"`
	Goals       []string   `json:"goals"`
	Languages   []string   `json:"languages"`
	Occupation  string     `json:"occupation"`
	Education   string     `json:"education"`
}

func (dto moderationProfileDTO) toModel() model.ModerationProfile {
	return model.ModerationProfile{
		UserID:      dto.UserID,
		TGID:        dto.TGID,
		Username:    strings.TrimSpace(dto.Username),
		DisplayName: strings.TrimSpace(dto.DisplayName),
		CityID:      strings.TrimSpace(dto.CityID),
		Birthdate:   dto.Birthdate,
		Age:         dto.Age,
		Gender:      strings.TrimSpace(dto.Gender),
		LookingFor:  strings.TrimSpace(dto.LookingFor),
		Goals:       cloneStrings(dto.Goals),
		Languages:   cloneStrings(dto.Languages),
		Occupation:  strings.TrimSpace(dto.Occupation),
		Education:   strings.TrimSpace(dto.Education),
	}
}

type moderationMediaDTO struct {
	Photos             []moderationMediaRefDTO `json:"photos"`
	PhotoKeys          []string                `json:"photo_keys"`
	PhotoURLs          []string                `json:"photo_urls"`
	PresignedPhotoURLs []string                `json:"presigned_photo_urls"`
	Circle             moderationMediaRefDTO   `json:"circle"`
	CircleKey          string                  `json:"circle_key"`
	CircleURL          string                  `json:"circle_url"`
	PresignedCircleURL string                  `json:"presigned_circle_url"`
}

type moderationMediaRefDTO struct {
	Key          string `json:"key"`
	S3Key        string `json:"s3_key"`
	URL          string `json:"url"`
	PresignedURL string `json:"presigned_url"`
	Presigned    string `json:"presigned"`
}

func (dto moderationMediaDTO) pickPhotoRefs() []string {
	refs := make([]string, 0, len(dto.PhotoKeys)+len(dto.PhotoURLs)+len(dto.PresignedPhotoURLs)+len(dto.Photos))
	addUniqueString(&refs, dto.PhotoURLs...)
	addUniqueString(&refs, dto.PresignedPhotoURLs...)
	addUniqueString(&refs, dto.PhotoKeys...)
	for _, media := range dto.Photos {
		addUniqueString(&refs, media.pickRef())
	}
	return refs
}

func (dto moderationMediaDTO) pickCircleRef() string {
	return firstNonEmpty(
		strings.TrimSpace(dto.CircleURL),
		strings.TrimSpace(dto.PresignedCircleURL),
		dto.Circle.pickRef(),
		strings.TrimSpace(dto.CircleKey),
	)
}

func (dto moderationMediaRefDTO) pickRef() string {
	return firstNonEmpty(
		strings.TrimSpace(dto.PresignedURL),
		strings.TrimSpace(dto.Presigned),
		strings.TrimSpace(dto.URL),
		strings.TrimSpace(dto.Key),
		strings.TrimSpace(dto.S3Key),
	)
}

func addUniqueString(target *[]string, values ...string) {
	if target == nil {
		return
	}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}

		exists := false
		for _, current := range *target {
			if current == trimmed {
				exists = true
				break
			}
		}
		if !exists {
			*target = append(*target, trimmed)
		}
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
