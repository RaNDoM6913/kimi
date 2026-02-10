package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"bot_moderator/internal/domain/enums"
	"bot_moderator/internal/domain/model"
	"bot_moderator/internal/infra/telegram"
	"bot_moderator/internal/repo/adminhttp"
	"bot_moderator/internal/services/access"
	lookupsvc "bot_moderator/internal/services/lookup"
	moderationsvc "bot_moderator/internal/services/moderation"
	"bot_moderator/internal/ui"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const (
	callbackPrefixAccess     = "acc"
	callbackPrefixModeration = "mod"
	callbackPrefixLookup     = "find"
	callbackPrefixSystem     = "sys"
	callbackPrefixWorkStats  = "wst"
)

const (
	lookupActionLookup      = "LOOKUP"
	lookupActionBan         = "BAN"
	lookupActionUnban       = "UNBAN"
	lookupActionForceReview = "FORCE_REVIEW"
)

var rejectReasonCodes = []string{
	"PHOTO_NO_FACE",
	"PHOTO_FAKE_NOT_YOU",
	"PHOTO_PROHIBITED",
	"CIRCLE_MISMATCH",
	"CIRCLE_FAILED",
	"PROFILE_INCOMPLETE",
	"SPAM_ADS_LINKS",
	"BOT_SUSPECT",
	"OTHER",
}

type rejectReasonOption struct {
	Code  string
	Label string
}

var rejectReasonOptions = []rejectReasonOption{
	{Code: "PHOTO_NO_FACE", Label: "PHOTO_NO_FACE"},
	{Code: "PHOTO_FAKE_NOT_YOU", Label: "PHOTO_FAKE_NOT_YOU"},
	{Code: "PHOTO_PROHIBITED", Label: "PHOTO_PROHIBITED"},
	{Code: "CIRCLE_MISMATCH", Label: "CIRCLE_MISMATCH"},
	{Code: "CIRCLE_FAILED", Label: "CIRCLE_FAILED"},
	{Code: "PROFILE_INCOMPLETE", Label: "PROFILE_INCOMPLETE"},
	{Code: "SPAM_ADS_LINKS", Label: "SPAM_ADS_LINKS"},
	{Code: "BOT_SUSPECT", Label: "BOT_SUSPECT"},
	{Code: "OTHER", Label: "Другая причина"},
}

type rejectCommentState struct {
	ActorTGID int64
	ActorRole enums.Role
	ItemID    int64
}

type lookupInputState struct {
	ActorTGID int64
	ActorRole enums.Role
}

type lookupSessionState struct {
	ActorTGID   int64
	ActorRole   enums.Role
	Query       string
	FoundUserID int64
}

func (a *App) routeUpdate(ctx context.Context, update tgbotapi.Update) {
	if update.Message != nil {
		a.routeMessage(ctx, update.Message)
	}

	if update.CallbackQuery != nil {
		a.handleCallback(ctx, update.CallbackQuery)
	}
}

func (a *App) routeMessage(ctx context.Context, message *tgbotapi.Message) {
	if message == nil {
		return
	}
	if message.From != nil {
		ctx = adminhttp.WithActorTGID(ctx, message.From.ID)
	}

	if message.IsCommand() {
		switch message.Command() {
		case "start":
			a.handleStart(ctx, message)
		default:
			a.sendText(message.Chat.ID, "Неизвестная команда. Используйте /start")
		}
		return
	}

	if a.handleRejectCommentIfNeeded(ctx, message) {
		return
	}

	if a.handleLookupInputIfNeeded(ctx, message) {
		return
	}

	a.handleMenuMessage(ctx, message)
}

func (a *App) handleMenuMessage(ctx context.Context, message *tgbotapi.Message) {
	if message == nil {
		return
	}

	switch strings.TrimSpace(message.Text) {
	case "Access":
		a.handleAccessEntry(ctx, message)
	case "Find user":
		a.handleFindUserEntry(ctx, message)
	case "Work Stats":
		a.handleWorkStatsEntry(ctx, message)
	case "History":
		a.handleHistoryEntry(ctx, message)
	case "System":
		a.handleSystemEntry(ctx, message)
	case "Приступить к модерации":
		a.handleAcquireModerationItem(ctx, message)
	}
}

func (a *App) handleAcquireModerationItem(ctx context.Context, message *tgbotapi.Message) {
	if message == nil || message.From == nil {
		return
	}

	actorTGID, role, err := a.resolveActorRole(ctx, message.From)
	if err != nil {
		a.logger.Warn("resolve actor role for moderation", "error", err, "tg_id", message.From.ID)
		a.sendText(message.Chat.ID, "Не удалось определить роль")
		return
	}
	if role != enums.RoleOwner && role != enums.RoleAdmin && role != enums.RoleModerator {
		a.sendText(message.Chat.ID, "Dating App: нет доступа")
		return
	}

	a.acquireAndSendNextModerationItem(ctx, message.Chat.ID, actorTGID)
}

func (a *App) sendModerationMedia(chatID int64, item model.ModerationQueueItem) {
	hasMedia := false
	for idx, mediaURL := range item.PhotoURLs {
		url := strings.TrimSpace(mediaURL)
		if url == "" {
			continue
		}
		hasMedia = true
		caption := fmt.Sprintf("Фото %d/3", idx+1)
		if err := a.sendPhotoByURL(chatID, url, caption); err != nil {
			a.logger.Warn("send photo", "error", err, "chat_id", chatID, "item_id", item.ModerationItemID, "index", idx)
		}
	}

	circleURL := strings.TrimSpace(item.CircleURL)
	if circleURL != "" {
		hasMedia = true
		if err := a.sendVideoByURL(chatID, circleURL, "Кружок"); err != nil {
			a.logger.Warn("send circle as video", "error", err, "chat_id", chatID, "item_id", item.ModerationItemID)
		}
	}

	if !hasMedia {
		a.sendText(chatID, "Медиа не найдены")
	}
}

func (a *App) sendModerationDecisionPrompt(chatID int64, moderationItemID int64) {
	rows := [][]telegram.InlineButton{{
		{Text: "✅ Approve", Data: fmt.Sprintf("%s:approve:%d", callbackPrefixModeration, moderationItemID)},
		{Text: "❌ Reject", Data: fmt.Sprintf("%s:reject:%d", callbackPrefixModeration, moderationItemID)},
	}}
	a.sendInline(chatID, "Решение по анкете", rows)
}

func (a *App) handleRejectCommentIfNeeded(ctx context.Context, message *tgbotapi.Message) bool {
	if message == nil || message.From == nil {
		return false
	}

	a.rejectMu.Lock()
	state, ok := a.rejectByChat[message.Chat.ID]
	a.rejectMu.Unlock()
	if !ok {
		return false
	}

	if state.ActorTGID != message.From.ID {
		return false
	}

	comment := strings.TrimSpace(message.Text)
	if comment == "-" {
		comment = ""
	}

	a.rejectMu.Lock()
	delete(a.rejectByChat, message.Chat.ID)
	a.rejectMu.Unlock()

	if _, err := a.rejectAndContinue(ctx, message.Chat.ID, state.ActorTGID, state.ActorRole, state.ItemID, "OTHER", comment); err != nil {
		a.logger.Warn("reject with comment", "error", err, "item_id", state.ItemID, "tg_id", state.ActorTGID)
		a.sendText(message.Chat.ID, "Не удалось отклонить анкету")
	}

	return true
}

func (a *App) handleLookupInputIfNeeded(ctx context.Context, message *tgbotapi.Message) bool {
	if message == nil || message.From == nil {
		return false
	}

	a.lookupInputMu.Lock()
	state, ok := a.lookupInputByChat[message.Chat.ID]
	a.lookupInputMu.Unlock()
	if !ok {
		return false
	}

	if state.ActorTGID != message.From.ID {
		return false
	}

	query := strings.TrimSpace(message.Text)
	if query == "" {
		a.sendText(message.Chat.ID, "Введите @username или tg_id")
		return true
	}

	deleteLookupInputState(a, message.Chat.ID)

	found, err := a.lookupService.FindUser(ctx, query)
	if errors.Is(err, lookupsvc.ErrUserNotFound) {
		a.sendText(message.Chat.ID, "Пользователь не найден. Введите @username или tg_id")
		setLookupInputState(a, message.Chat.ID, state)
		return true
	}
	if err != nil {
		a.logger.Warn("find user", "error", err, "query", query, "tg_id", state.ActorTGID)
		a.sendText(message.Chat.ID, "Не удалось выполнить поиск")
		setLookupInputState(a, message.Chat.ID, state)
		return true
	}

	session := lookupSessionState{
		ActorTGID:   state.ActorTGID,
		ActorRole:   state.ActorRole,
		Query:       query,
		FoundUserID: found.UserID,
	}
	setLookupSessionState(a, message.Chat.ID, session)

	foundUserID := found.UserID
	if err := a.lookupService.LogAction(ctx, state.ActorTGID, state.ActorRole, query, &foundUserID, lookupActionLookup, map[string]interface{}{
		"target_user_id": found.UserID,
		"target_tg_id":   found.TGID,
		"query":          query,
	}); err != nil {
		a.logger.Warn("log lookup action", "error", err, "query", query, "tg_id", state.ActorTGID)
	}
	if err := a.auditService.LogLookup(ctx, state.ActorTGID, query, found.UserID); err != nil {
		a.logger.Warn("log lookup audit", "error", err, "query", query, "tg_id", state.ActorTGID)
	}

	a.sendLookupUserCard(message.Chat.ID, found)
	return true
}

func (a *App) sendPhotoByURL(chatID int64, mediaURL string, caption string) error {
	photo := tgbotapi.NewPhoto(chatID, tgbotapi.FileURL(mediaURL))
	photo.Caption = caption
	return a.tg.Send(photo)
}

func (a *App) sendVideoByURL(chatID int64, mediaURL string, caption string) error {
	video := tgbotapi.NewVideo(chatID, tgbotapi.FileURL(mediaURL))
	video.Caption = caption
	return a.tg.Send(video)
}

func (a *App) handleStart(ctx context.Context, message *tgbotapi.Message) {
	role := enums.RoleNone
	actorTGID := int64(0)
	if message.From != nil {
		actorTGID = message.From.ID
		if err := a.accessService.TouchUser(ctx, model.BotUser{
			TgID:       actorTGID,
			Username:   message.From.UserName,
			FirstName:  message.From.FirstName,
			LastName:   message.From.LastName,
			LastSeenAt: time.Now().UTC(),
		}); err != nil {
			a.logger.Warn("touch bot user", "error", err, "tg_id", actorTGID)
		}

		resolvedRole, err := a.accessService.ResolveRole(ctx, actorTGID)
		if err != nil {
			a.logger.Warn("resolve role", "error", err, "tg_id", actorTGID)
		} else {
			role = resolvedRole
		}
	}

	if role == enums.RoleNone {
		response := tgbotapi.NewMessage(message.Chat.ID, "У вас нет доступа к этому боту")
		response.ReplyMarkup = tgbotapi.NewRemoveKeyboard(true)
		if err := a.tg.Send(response); err != nil {
			a.logger.Error("send /start no-access response", "error", err)
		}

		if err := a.auditService.LogStart(ctx, actorTGID, role); err != nil {
			a.logger.Warn("write audit log", "error", err)
		}
		return
	}

	text, menu := ui.RenderStart(role)
	response := tgbotapi.NewMessage(message.Chat.ID, text)
	response.ReplyMarkup = telegram.BuildReplyKeyboard(menu)

	if err := a.tg.Send(response); err != nil {
		a.logger.Error("send /start response", "error", err)
	}

	if err := a.auditService.LogStart(ctx, actorTGID, role); err != nil {
		a.logger.Warn("write audit log", "error", err)
	}
}

func (a *App) handleAccessEntry(ctx context.Context, message *tgbotapi.Message) {
	if message == nil || message.From == nil {
		return
	}

	_, role, err := a.resolveActorRole(ctx, message.From)
	if err != nil {
		a.logger.Warn("resolve actor role for access", "error", err, "tg_id", message.From.ID)
		a.sendText(message.Chat.ID, "Не удалось открыть Access")
		return
	}

	if !a.accessService.CanOpenAccess(role) {
		a.sendText(message.Chat.ID, "Dating App: нет доступа")
		return
	}

	a.sendAccessScreen(ctx, message.Chat.ID)
}

func (a *App) handleFindUserEntry(ctx context.Context, message *tgbotapi.Message) {
	if message == nil || message.From == nil {
		return
	}

	actorTGID, role, err := a.resolveActorRole(ctx, message.From)
	if err != nil {
		a.logger.Warn("resolve actor role for find user", "error", err, "tg_id", message.From.ID)
		a.sendText(message.Chat.ID, "Не удалось открыть Find user")
		return
	}

	if role != enums.RoleOwner && role != enums.RoleAdmin {
		a.sendText(message.Chat.ID, "Dating App: нет доступа")
		return
	}

	setLookupInputState(a, message.Chat.ID, lookupInputState{
		ActorTGID: actorTGID,
		ActorRole: role,
	})
	a.sendText(message.Chat.ID, "Введите @username или tg_id")
}

func (a *App) handleHistoryEntry(ctx context.Context, message *tgbotapi.Message) {
	if message == nil || message.From == nil {
		return
	}

	_, role, err := a.resolveActorRole(ctx, message.From)
	if err != nil {
		a.logger.Warn("resolve actor role for history", "error", err, "tg_id", message.From.ID)
		a.sendText(message.Chat.ID, "Не удалось открыть History")
		return
	}
	if role != enums.RoleOwner {
		a.sendText(message.Chat.ID, "Dating App: нет доступа")
		return
	}

	entries, err := a.auditService.ListRecent(ctx, 50)
	if err != nil {
		a.logger.Warn("list recent audit", "error", err)
		a.sendText(message.Chat.ID, "Не удалось загрузить History")
		return
	}

	a.sendAuditHistory(message.Chat.ID, entries)
}

func (a *App) handleWorkStatsEntry(ctx context.Context, message *tgbotapi.Message) {
	if message == nil || message.From == nil {
		return
	}

	actorTGID, role, err := a.resolveActorRole(ctx, message.From)
	if err != nil {
		a.logger.Warn("resolve actor role for work stats", "error", err, "tg_id", message.From.ID)
		a.sendText(message.Chat.ID, "Не удалось открыть Work Stats")
		return
	}
	if role != enums.RoleOwner {
		a.sendText(message.Chat.ID, "Dating App: нет доступа")
		return
	}

	report, err := a.statsService.BuildReport(ctx)
	if err != nil {
		a.logger.Warn("build work stats report", "error", err, "tg_id", actorTGID)
		a.sendText(message.Chat.ID, "Не удалось загрузить Work Stats")
		return
	}

	if err := a.auditService.LogSystemViewWorkStats(ctx, actorTGID); err != nil {
		a.logger.Warn("write work stats audit", "error", err, "tg_id", actorTGID)
	}

	a.sendWorkStatsScreen(message.Chat.ID, report)
}

func (a *App) handleSystemEntry(ctx context.Context, message *tgbotapi.Message) {
	if message == nil || message.From == nil {
		return
	}

	_, role, err := a.resolveActorRole(ctx, message.From)
	if err != nil {
		a.logger.Warn("resolve actor role for system", "error", err, "tg_id", message.From.ID)
		a.sendText(message.Chat.ID, "Не удалось открыть System")
		return
	}
	if role != enums.RoleOwner {
		a.sendText(message.Chat.ID, "Dating App: нет доступа")
		return
	}

	a.sendSystemScreen(ctx, message.Chat.ID)
}

func (a *App) handleCallback(ctx context.Context, query *tgbotapi.CallbackQuery) {
	if query == nil || query.From == nil {
		return
	}
	ctx = adminhttp.WithActorTGID(ctx, query.From.ID)

	chatID, ok := callbackChatID(query)
	if !ok {
		a.answerCallback(query.ID, "", false)
		return
	}

	ackText := ""
	ackAlert := false
	defer a.answerCallback(query.ID, ackText, ackAlert)

	parts := strings.Split(query.Data, ":")
	if len(parts) < 2 {
		return
	}

	switch parts[0] {
	case callbackPrefixAccess:
		ackText, ackAlert = a.handleAccessCallback(ctx, chatID, query, parts)
	case callbackPrefixModeration:
		ackText, ackAlert = a.handleModerationCallback(ctx, chatID, query, parts)
	case callbackPrefixLookup:
		ackText, ackAlert = a.handleLookupCallback(ctx, chatID, query, parts)
	case callbackPrefixSystem:
		ackText, ackAlert = a.handleSystemCallback(ctx, chatID, query, parts)
	case callbackPrefixWorkStats:
		ackText, ackAlert = a.handleWorkStatsCallback(ctx, chatID, query, parts)
	}
}

func (a *App) handleAccessCallback(ctx context.Context, chatID int64, query *tgbotapi.CallbackQuery, parts []string) (string, bool) {
	actorTGID, actorRole, err := a.resolveActorRole(ctx, query.From)
	if err != nil {
		return "Не удалось определить роль", true
	}
	if !a.accessService.CanOpenAccess(actorRole) {
		return "Нет доступа", true
	}

	switch parts[1] {
	case "root":
		a.sendAccessScreen(ctx, chatID)
	case "back":
		a.sendMainMenu(chatID, actorRole)
	case "add":
		if len(parts) < 3 {
			return "", false
		}
		targetRole, valid := parseAssignableRole(parts[2])
		if !valid {
			return "", false
		}
		if !a.accessService.CanGrantRole(actorRole, targetRole) {
			return "Недостаточно прав", true
		}
		a.sendRecentUsersScreen(ctx, chatID, targetRole)
	case "addsel":
		if len(parts) < 4 {
			return "", false
		}
		targetRole, valid := parseAssignableRole(parts[2])
		if !valid {
			return "", false
		}
		targetTGID, err := parseTGID(parts[3])
		if err != nil {
			return "Некорректный пользователь", true
		}
		if !a.accessService.CanGrantRole(actorRole, targetRole) {
			return "Недостаточно прав", true
		}

		targetUser := a.safeGetUser(ctx, targetTGID)
		a.sendGrantConfirmScreen(chatID, targetUser, targetRole)
	case "addok":
		if len(parts) < 4 {
			return "", false
		}
		targetRole, valid := parseAssignableRole(parts[2])
		if !valid {
			return "", false
		}
		targetTGID, err := parseTGID(parts[3])
		if err != nil {
			return "Некорректный пользователь", true
		}

		err = a.accessService.GrantRole(ctx, actorTGID, actorRole, targetTGID, targetRole)
		if errors.Is(err, access.ErrAccessDenied) {
			return "Недостаточно прав", true
		}
		if err != nil {
			a.logger.Warn("grant role", "error", err, "actor_tg_id", actorTGID, "target_tg_id", targetTGID)
			return "Не удалось назначить роль", true
		}

		targetUser := a.safeGetUser(ctx, targetTGID)
		if err := a.auditService.LogRoleGranted(ctx, actorTGID, targetTGID, targetUser.Username, targetRole); err != nil {
			a.logger.Warn("write role granted audit", "error", err)
		}

		a.sendText(chatID, fmt.Sprintf("Роль %s назначена: %s", targetRole, renderUserLabel(targetUser.TgID, targetUser.Username)))
		a.sendAccessScreen(ctx, chatID)
	case "edit":
		a.sendEditListScreen(ctx, chatID)
	case "editsel":
		if len(parts) < 3 {
			return "", false
		}
		targetTGID, err := parseTGID(parts[2])
		if err != nil {
			return "Некорректный пользователь", true
		}

		targetRole, err := a.accessService.GetActiveManagedRole(ctx, targetTGID)
		if err != nil {
			a.logger.Warn("get active managed role", "error", err, "target_tg_id", targetTGID)
			return "Не удалось загрузить роль", true
		}
		if targetRole == enums.RoleNone {
			a.sendText(chatID, "У пользователя нет активной роли ADMIN/MODERATOR")
			return "", false
		}
		if !a.accessService.CanRevokeRole(actorRole, targetRole) {
			return "Недостаточно прав", true
		}

		targetUser := a.safeGetUser(ctx, targetTGID)
		a.sendEditConfirmScreen(chatID, targetUser, targetRole)
	case "revoke":
		if len(parts) < 3 {
			return "", false
		}
		targetTGID, err := parseTGID(parts[2])
		if err != nil {
			return "Некорректный пользователь", true
		}

		targetUser := a.safeGetUser(ctx, targetTGID)
		targetRole, revoked, err := a.accessService.RevokeRole(ctx, actorRole, targetTGID)
		if errors.Is(err, access.ErrAccessDenied) {
			return "Недостаточно прав", true
		}
		if err != nil {
			a.logger.Warn("revoke role", "error", err, "actor_tg_id", actorTGID, "target_tg_id", targetTGID)
			return "Не удалось снять роль", true
		}
		if !revoked {
			a.sendText(chatID, "У пользователя нет активной роли для снятия")
			return "", false
		}

		if err := a.auditService.LogRoleRevoked(ctx, actorTGID, targetTGID, targetUser.Username, targetRole); err != nil {
			a.logger.Warn("write role revoked audit", "error", err)
		}

		a.sendText(chatID, fmt.Sprintf("Роль %s снята: %s", targetRole, renderUserLabel(targetUser.TgID, targetUser.Username)))
		a.sendAccessScreen(ctx, chatID)
	}

	return "", false
}

func (a *App) handleModerationCallback(ctx context.Context, chatID int64, query *tgbotapi.CallbackQuery, parts []string) (string, bool) {
	actorTGID, actorRole, err := a.resolveActorRole(ctx, query.From)
	if err != nil {
		return "Не удалось определить роль", true
	}
	if actorRole != enums.RoleOwner && actorRole != enums.RoleAdmin && actorRole != enums.RoleModerator {
		return "Нет доступа", true
	}

	switch parts[1] {
	case "decision":
		if len(parts) < 3 {
			return "", false
		}
		itemID, err := parseTGID(parts[2])
		if err != nil {
			return "Некорректный item id", true
		}
		a.sendModerationDecisionPrompt(chatID, itemID)
		return "", false
	case "approve":
		if len(parts) < 3 {
			return "", false
		}
		itemID, err := parseTGID(parts[2])
		if err != nil {
			return "Некорректный item id", true
		}
		if _, err := a.approveAndContinue(ctx, chatID, actorTGID, actorRole, itemID); err != nil {
			a.logger.Warn("approve moderation item", "error", err, "item_id", itemID, "tg_id", actorTGID)
			return "Не удалось одобрить анкету", true
		}
		return "Одобрено", false
	case "reject":
		if len(parts) < 3 {
			return "", false
		}
		itemID, err := parseTGID(parts[2])
		if err != nil {
			return "Некорректный item id", true
		}
		a.sendRejectReasonPrompt(chatID, itemID)
		return "Выберите причину", false
	case "reason":
		if len(parts) < 4 {
			return "", false
		}
		itemID, err := parseTGID(parts[2])
		if err != nil {
			return "Некорректный item id", true
		}
		reasonCode := normalizeReasonCode(parts[3])
		if !isValidRejectReason(reasonCode) {
			return "Некорректный reason code", true
		}

		if reasonCode == "OTHER" {
			a.rejectMu.Lock()
			a.rejectByChat[chatID] = rejectCommentState{
				ActorTGID: actorTGID,
				ActorRole: actorRole,
				ItemID:    itemID,
			}
			a.rejectMu.Unlock()
			a.sendText(chatID, "Введите комментарий для причины OTHER (или '-' без комментария)")
			return "Ожидаю комментарий", false
		}

		if _, err := a.rejectAndContinue(ctx, chatID, actorTGID, actorRole, itemID, reasonCode, ""); err != nil {
			a.logger.Warn("reject moderation item", "error", err, "item_id", itemID, "reason_code", reasonCode)
			return "Не удалось отклонить анкету", true
		}
		return "Отклонено", false
	default:
		return "", false
	}
}

func (a *App) handleLookupCallback(ctx context.Context, chatID int64, query *tgbotapi.CallbackQuery, parts []string) (string, bool) {
	actorTGID, actorRole, err := a.resolveActorRole(ctx, query.From)
	if err != nil {
		return "Не удалось определить роль", true
	}
	if actorRole != enums.RoleOwner && actorRole != enums.RoleAdmin {
		return "Нет доступа", true
	}

	switch parts[1] {
	case "back":
		deleteLookupInputState(a, chatID)
		deleteLookupSessionState(a, chatID)
		a.sendMainMenu(chatID, actorRole)
		return "", false
	case "act":
		if len(parts) < 4 {
			return "", false
		}
		action := normalizeLookupAction(parts[2])
		if !isValidLookupAction(action) {
			return "Некорректное действие", true
		}

		userID, err := parseTGID(parts[3])
		if err != nil || userID <= 0 {
			return "Некорректный user id", true
		}

		a.sendLookupActionConfirm(chatID, action, userID)
		return "Подтвердите действие", false
	case "yes":
		if len(parts) < 4 {
			return "", false
		}
		action := normalizeLookupAction(parts[2])
		if !isValidLookupAction(action) {
			return "Некорректное действие", true
		}

		userID, err := parseTGID(parts[3])
		if err != nil || userID <= 0 {
			return "Некорректный user id", true
		}

		if err := a.executeLookupAction(ctx, chatID, actorTGID, actorRole, action, userID); err != nil {
			a.logger.Warn("execute lookup action", "error", err, "action", action, "user_id", userID, "tg_id", actorTGID)
			return "Не удалось выполнить действие", true
		}
		return "Готово", false
	case "no":
		if len(parts) < 4 {
			return "", false
		}
		userID, err := parseTGID(parts[3])
		if err != nil || userID <= 0 {
			return "Некорректный user id", true
		}

		user, err := a.lookupService.GetByUserID(ctx, userID)
		if err != nil {
			a.logger.Warn("reload lookup user after cancel", "error", err, "user_id", userID)
			return "Не удалось обновить карточку", true
		}
		a.sendLookupUserCard(chatID, user)
		return "Отменено", false
	default:
		return "", false
	}
}

func (a *App) handleSystemCallback(ctx context.Context, chatID int64, query *tgbotapi.CallbackQuery, parts []string) (string, bool) {
	actorTGID, actorRole, err := a.resolveActorRole(ctx, query.From)
	if err != nil {
		return "Не удалось определить роль", true
	}
	if actorRole != enums.RoleOwner {
		return "Нет доступа", true
	}

	switch parts[1] {
	case "root":
		a.sendSystemScreen(ctx, chatID)
		return "", false
	case "toggle":
		newValue, err := a.systemService.ToggleRegistration(ctx, actorTGID)
		if err != nil {
			a.logger.Warn("toggle registration", "error", err, "tg_id", actorTGID)
			return "Не удалось переключить регистрацию", true
		}
		if err := a.auditService.LogSystemToggleRegistration(ctx, actorTGID, newValue); err != nil {
			a.logger.Warn("write system toggle audit", "error", err, "tg_id", actorTGID)
		}
		a.sendSystemScreen(ctx, chatID)
		return "Обновлено", false
	case "users":
		count, err := a.systemService.GetUsersCount(ctx)
		if err != nil {
			a.logger.Warn("get users count", "error", err, "tg_id", actorTGID)
			return "Не удалось получить статистику", true
		}
		if err := a.auditService.LogSystemViewUsersCount(ctx, actorTGID, count.Total, count.Approved); err != nil {
			a.logger.Warn("write users count audit", "error", err, "tg_id", actorTGID)
		}
		a.sendUsersCountScreen(chatID, count.Total, count.Approved)
		return "", false
	case "back":
		a.sendMainMenu(chatID, actorRole)
		return "", false
	default:
		return "", false
	}
}

func (a *App) handleWorkStatsCallback(ctx context.Context, chatID int64, query *tgbotapi.CallbackQuery, parts []string) (string, bool) {
	_, actorRole, err := a.resolveActorRole(ctx, query.From)
	if err != nil {
		return "Не удалось определить роль", true
	}
	if actorRole != enums.RoleOwner {
		return "Нет доступа", true
	}

	if len(parts) < 2 {
		return "", false
	}
	switch parts[1] {
	case "back":
		a.sendMainMenu(chatID, actorRole)
		return "", false
	default:
		return "", false
	}
}

func (a *App) sendRejectReasonPrompt(chatID int64, moderationItemID int64) {
	rows := make([][]telegram.InlineButton, 0, len(rejectReasonOptions)+1)
	for _, option := range rejectReasonOptions {
		rows = append(rows, []telegram.InlineButton{{
			Text: option.Label,
			Data: fmt.Sprintf("%s:reason:%d:%s", callbackPrefixModeration, moderationItemID, option.Code),
		}})
	}
	rows = append(rows, []telegram.InlineButton{{
		Text: "⬅️ Back",
		Data: fmt.Sprintf("%s:decision:%d", callbackPrefixModeration, moderationItemID),
	}})
	a.sendInline(chatID, "Выберите причину отклонения", rows)
}

func (a *App) rejectAndContinue(
	ctx context.Context,
	chatID int64,
	actorTGID int64,
	actorRole enums.Role,
	moderationItemID int64,
	reasonCode string,
	comment string,
) (moderationsvc.RejectResult, error) {
	result, err := a.moderationService.Reject(ctx, moderationsvc.RejectInput{
		ActorTGID:        actorTGID,
		ActorRole:        actorRole,
		ModerationItemID: moderationItemID,
		ReasonCode:       reasonCode,
		Comment:          comment,
	})
	if err != nil {
		return moderationsvc.RejectResult{}, err
	}

	if err := a.auditService.LogModerationReject(ctx, actorTGID, result.TargetUserID, result.ModerationItemID, result.ReasonCode); err != nil {
		a.logger.Warn("write moderation reject audit", "error", err)
	}

	a.sendText(chatID, fmt.Sprintf("Анкета #%d отклонена (%s)", result.ModerationItemID, result.ReasonCode))
	a.acquireAndSendNextModerationItem(ctx, chatID, actorTGID)
	return result, nil
}

func (a *App) approveAndContinue(
	ctx context.Context,
	chatID int64,
	actorTGID int64,
	actorRole enums.Role,
	moderationItemID int64,
) (moderationsvc.ApproveResult, error) {
	result, err := a.moderationService.Approve(ctx, moderationsvc.ApproveInput{
		ActorTGID:        actorTGID,
		ActorRole:        actorRole,
		ModerationItemID: moderationItemID,
	})
	if err != nil {
		return moderationsvc.ApproveResult{}, err
	}

	if err := a.auditService.LogModerationApprove(ctx, actorTGID, result.TargetUserID, result.ModerationItemID); err != nil {
		a.logger.Warn("write moderation approve audit", "error", err)
	}

	a.sendText(chatID, fmt.Sprintf("Анкета #%d одобрена", result.ModerationItemID))
	a.acquireAndSendNextModerationItem(ctx, chatID, actorTGID)
	return result, nil
}

func (a *App) acquireAndSendNextModerationItem(ctx context.Context, chatID int64, actorTGID int64) {
	item, err := a.moderationService.AcquireNextPending(ctx, actorTGID)
	if errors.Is(err, moderationsvc.ErrQueueEmpty) {
		a.sendText(chatID, "Очередь пуста")
		return
	}
	if err != nil {
		a.logger.Warn("acquire next moderation item after decision", "error", err, "tg_id", actorTGID)
		a.sendText(chatID, "Не удалось получить следующую анкету из очереди")
		return
	}

	a.sendText(chatID, renderModerationQueueItem(item))
	a.sendModerationMedia(chatID, item)
	a.sendModerationDecisionPrompt(chatID, item.ModerationItemID)
}

func (a *App) sendLookupActionConfirm(chatID int64, action string, userID int64) {
	var text string
	switch action {
	case lookupActionBan:
		text = fmt.Sprintf("Подтвердить Ban для user_id=%d?", userID)
	case lookupActionUnban:
		text = fmt.Sprintf("Подтвердить Unban для user_id=%d?", userID)
	case lookupActionForceReview:
		text = fmt.Sprintf("Подтвердить Force Review для user_id=%d?", userID)
	default:
		text = "Подтвердить действие?"
	}

	rows := [][]telegram.InlineButton{
		{
			{Text: "Yes", Data: fmt.Sprintf("%s:yes:%s:%d", callbackPrefixLookup, action, userID)},
			{Text: "No", Data: fmt.Sprintf("%s:no:%s:%d", callbackPrefixLookup, action, userID)},
		},
	}
	a.sendInline(chatID, text, rows)
}

func (a *App) executeLookupAction(ctx context.Context, chatID int64, actorTGID int64, actorRole enums.Role, action string, userID int64) error {
	target, err := a.lookupService.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}

	session, hasSession := getLookupSessionState(a, chatID)
	queryText := strconv.FormatInt(userID, 10)
	if hasSession && session.ActorTGID == actorTGID {
		if strings.TrimSpace(session.Query) != "" {
			queryText = strings.TrimSpace(session.Query)
		}
	}

	switch action {
	case lookupActionBan:
		reason := "BANNED_BY_ADMIN"
		if err := a.bansService.Ban(ctx, userID, reason, actorTGID); err != nil {
			return err
		}
		if err := a.lookupService.LogAction(ctx, actorTGID, actorRole, queryText, &userID, lookupActionBan, map[string]interface{}{
			"target_user_id": target.UserID,
			"target_tg_id":   target.TGID,
			"reason":         reason,
		}); err != nil {
			a.logger.Warn("log ban action", "error", err, "user_id", userID)
		}
		if err := a.auditService.LogBan(ctx, actorTGID, target.UserID, reason); err != nil {
			a.logger.Warn("log ban audit", "error", err, "user_id", userID)
		}
		a.sendText(chatID, fmt.Sprintf("Пользователь user_id=%d заблокирован", userID))
	case lookupActionUnban:
		if err := a.bansService.Unban(ctx, userID, actorTGID); err != nil {
			return err
		}
		if err := a.lookupService.LogAction(ctx, actorTGID, actorRole, queryText, &userID, lookupActionUnban, map[string]interface{}{
			"target_user_id": target.UserID,
			"target_tg_id":   target.TGID,
		}); err != nil {
			a.logger.Warn("log unban action", "error", err, "user_id", userID)
		}
		if err := a.auditService.LogUnban(ctx, actorTGID, target.UserID); err != nil {
			a.logger.Warn("log unban audit", "error", err, "user_id", userID)
		}
		a.sendText(chatID, fmt.Sprintf("Пользователь user_id=%d разблокирован", userID))
	case lookupActionForceReview:
		if err := a.lookupService.ForceReview(ctx, userID); err != nil {
			return err
		}
		if err := a.lookupService.LogAction(ctx, actorTGID, actorRole, queryText, &userID, lookupActionForceReview, map[string]interface{}{
			"target_user_id": target.UserID,
			"target_tg_id":   target.TGID,
		}); err != nil {
			a.logger.Warn("log force review action", "error", err, "user_id", userID)
		}
		if err := a.auditService.LogForceReview(ctx, actorTGID, target.UserID); err != nil {
			a.logger.Warn("log force review audit", "error", err, "user_id", userID)
		}
		a.sendText(chatID, fmt.Sprintf("Пользователь user_id=%d отправлен на повторную модерацию", userID))
	default:
		return fmt.Errorf("unknown lookup action: %s", action)
	}

	updated, err := a.lookupService.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	a.sendLookupUserCard(chatID, updated)
	return nil
}

func (a *App) sendLookupUserCard(chatID int64, user model.LookupUser) {
	a.sendText(chatID, renderLookupUser(user))

	hasMedia := false
	for idx, photoURL := range user.PhotoURLs {
		url := strings.TrimSpace(photoURL)
		if url == "" {
			continue
		}
		hasMedia = true
		caption := fmt.Sprintf("Фото %d/3", idx+1)
		if err := a.sendPhotoByURL(chatID, url, caption); err != nil {
			a.logger.Warn("send lookup photo", "error", err, "chat_id", chatID, "user_id", user.UserID, "index", idx)
		}
	}
	if strings.TrimSpace(user.CircleURL) != "" {
		hasMedia = true
		if err := a.sendVideoByURL(chatID, user.CircleURL, "Кружок"); err != nil {
			a.logger.Warn("send lookup circle", "error", err, "chat_id", chatID, "user_id", user.UserID)
		}
	}
	if !hasMedia {
		a.sendText(chatID, "Медиа не найдены")
	}

	rows := [][]telegram.InlineButton{
		{
			{Text: "Ban", Data: fmt.Sprintf("%s:act:%s:%d", callbackPrefixLookup, lookupActionBan, user.UserID)},
			{Text: "Unban", Data: fmt.Sprintf("%s:act:%s:%d", callbackPrefixLookup, lookupActionUnban, user.UserID)},
		},
		{
			{Text: "Force Review", Data: fmt.Sprintf("%s:act:%s:%d", callbackPrefixLookup, lookupActionForceReview, user.UserID)},
		},
		{
			{Text: "Back", Data: fmt.Sprintf("%s:back", callbackPrefixLookup)},
		},
	}
	a.sendInline(chatID, "Действия", rows)
}

func (a *App) sendAuditHistory(chatID int64, entries []model.Audit) {
	if len(entries) == 0 {
		a.sendText(chatID, "History пуста")
		return
	}

	lines := make([]string, 0, len(entries)+1)
	lines = append(lines, "History (последние 50):")
	for _, entry := range entries {
		payload := strings.TrimSpace(string(entry.Payload))
		if payload == "" {
			payload = "{}"
		}
		line := fmt.Sprintf(
			"%s | %s | actor=%d | %s",
			entry.CreatedAt.UTC().Format("2006-01-02 15:04:05"),
			entry.Action,
			entry.ActorTGID,
			payload,
		)
		lines = append(lines, line)
	}

	for _, chunk := range splitByLength(lines, 3600) {
		a.sendText(chatID, chunk)
	}
}

func (a *App) sendSystemScreen(ctx context.Context, chatID int64) {
	enabled, err := a.systemService.GetRegistrationEnabled(ctx)
	if err != nil {
		a.logger.Warn("get registration flag", "error", err)
		a.sendText(chatID, "Не удалось загрузить System")
		return
	}

	regText := "Registration: OFF"
	if enabled {
		regText = "Registration: ON"
	}

	rows := [][]telegram.InlineButton{
		{{Text: regText, Data: fmt.Sprintf("%s:toggle", callbackPrefixSystem)}},
		{{Text: "Users count", Data: fmt.Sprintf("%s:users", callbackPrefixSystem)}},
		{{Text: "Back", Data: fmt.Sprintf("%s:back", callbackPrefixSystem)}},
	}
	a.sendInline(chatID, "System", rows)
}

func (a *App) sendUsersCountScreen(chatID int64, total int64, approved int64) {
	text := fmt.Sprintf("Users count\nTotal: %d\nApproved: %d", total, approved)
	rows := [][]telegram.InlineButton{
		{{Text: "Back", Data: fmt.Sprintf("%s:root", callbackPrefixSystem)}},
	}
	a.sendInline(chatID, text, rows)
}

func (a *App) sendWorkStatsScreen(chatID int64, report model.WorkStatsReport) {
	text := ui.RenderWorkStats(report)
	chunks := splitByLength(strings.Split(text, "\n"), 3600)
	if len(chunks) == 0 {
		chunks = []string{"Work Stats"}
	}
	for i := 0; i < len(chunks)-1; i++ {
		a.sendText(chatID, chunks[i])
	}

	text = chunks[len(chunks)-1]
	rows := [][]telegram.InlineButton{
		{{Text: "Back", Data: fmt.Sprintf("%s:back", callbackPrefixWorkStats)}},
	}
	a.sendInline(chatID, text, rows)
}

func (a *App) sendAccessScreen(ctx context.Context, chatID int64) {
	text, err := a.buildAccessSummary(ctx)
	if err != nil {
		a.logger.Warn("build access summary", "error", err)
		a.sendText(chatID, "Не удалось загрузить Access")
		return
	}

	rows := [][]telegram.InlineButton{
		{
			{Text: "➕ Add Moderator", Data: "acc:add:MODERATOR"},
			{Text: "➕ Add Admin", Data: "acc:add:ADMIN"},
		},
		{
			{Text: "✏️ Edit", Data: "acc:edit"},
			{Text: "⬅️ Back", Data: "acc:back"},
		},
	}
	a.sendInline(chatID, text, rows)
}

func (a *App) sendRecentUsersScreen(ctx context.Context, chatID int64, targetRole enums.Role) {
	users, err := a.accessService.ListRecentUsers(ctx, 20)
	if err != nil {
		a.logger.Warn("list recent users", "error", err)
		a.sendText(chatID, "Не удалось загрузить Recent users")
		return
	}

	rows := make([][]telegram.InlineButton, 0, len(users)+1)
	for _, user := range users {
		rows = append(rows, []telegram.InlineButton{{
			Text: renderUserLabel(user.TgID, user.Username),
			Data: fmt.Sprintf("acc:addsel:%s:%d", targetRole, user.TgID),
		}})
	}
	rows = append(rows, []telegram.InlineButton{{Text: "⬅️ Back", Data: "acc:root"}})

	text := fmt.Sprintf("Recent users (последние %d, /start): выберите пользователя для роли %s", len(users), targetRole)
	if len(users) == 0 {
		text = "Recent users пуст. Пользователи появятся после /start."
	}
	a.sendInline(chatID, text, rows)
}

func (a *App) sendGrantConfirmScreen(chatID int64, user model.BotUser, targetRole enums.Role) {
	rows := [][]telegram.InlineButton{
		{{Text: "✅ Confirm", Data: fmt.Sprintf("acc:addok:%s:%d", targetRole, user.TgID)}},
		{{Text: "⬅️ Back", Data: fmt.Sprintf("acc:add:%s", targetRole)}},
	}
	text := fmt.Sprintf("Назначить роль %s пользователю %s?", targetRole, renderUserLabel(user.TgID, user.Username))
	a.sendInline(chatID, text, rows)
}

func (a *App) sendEditListScreen(ctx context.Context, chatID int64) {
	assignments, err := a.accessService.ListActiveAssignments(ctx)
	if err != nil {
		a.logger.Warn("list active assignments", "error", err)
		a.sendText(chatID, "Не удалось загрузить роли")
		return
	}

	rows := make([][]telegram.InlineButton, 0, len(assignments)+1)
	for _, assignment := range assignments {
		rows = append(rows, []telegram.InlineButton{{
			Text: fmt.Sprintf("%s [%s]", renderUserLabel(assignment.TgID, assignment.Username), assignment.Role),
			Data: fmt.Sprintf("acc:editsel:%d", assignment.TgID),
		}})
	}
	rows = append(rows, []telegram.InlineButton{{Text: "⬅️ Back", Data: "acc:root"}})

	text := "Выберите Admin/Moderator для изменения роли"
	if len(assignments) == 0 {
		text = "Нет активных ролей Admin/Moderator"
	}
	a.sendInline(chatID, text, rows)
}

func (a *App) sendEditConfirmScreen(chatID int64, user model.BotUser, role enums.Role) {
	rows := [][]telegram.InlineButton{
		{{Text: "Remove role", Data: fmt.Sprintf("acc:revoke:%d", user.TgID)}},
		{{Text: "Back", Data: "acc:edit"}},
	}
	text := fmt.Sprintf("Пользователь: %s\nТекущая роль: %s", renderUserLabel(user.TgID, user.Username), role)
	a.sendInline(chatID, text, rows)
}

func (a *App) buildAccessSummary(ctx context.Context) (string, error) {
	assignments, err := a.accessService.ListActiveAssignments(ctx)
	if err != nil {
		return "", err
	}

	admins := make([]string, 0)
	moderators := make([]string, 0)
	for _, assignment := range assignments {
		entry := "- " + renderUserLabel(assignment.TgID, assignment.Username)
		switch assignment.Role {
		case enums.RoleAdmin:
			admins = append(admins, entry)
		case enums.RoleModerator:
			moderators = append(moderators, entry)
		}
	}

	if len(admins) == 0 {
		admins = append(admins, "- —")
	}
	if len(moderators) == 0 {
		moderators = append(moderators, "- —")
	}

	return fmt.Sprintf(
		"Access\n\nAdmins:\n%s\n\nModerators:\n%s",
		strings.Join(admins, "\n"),
		strings.Join(moderators, "\n"),
	), nil
}

func (a *App) safeGetUser(ctx context.Context, tgID int64) model.BotUser {
	user, err := a.accessService.GetUser(ctx, tgID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			a.logger.Warn("get bot user", "error", err, "tg_id", tgID)
		}
		return model.BotUser{TgID: tgID}
	}
	return user
}

func (a *App) resolveActorRole(ctx context.Context, user *tgbotapi.User) (int64, enums.Role, error) {
	if user == nil {
		return 0, enums.RoleNone, nil
	}
	role, err := a.accessService.ResolveRole(ctx, user.ID)
	if err != nil {
		return user.ID, enums.RoleNone, err
	}
	return user.ID, role, nil
}

func (a *App) sendMainMenu(chatID int64, role enums.Role) {
	text, menu := ui.RenderStart(role)
	response := tgbotapi.NewMessage(chatID, text)
	response.ReplyMarkup = telegram.BuildReplyKeyboard(menu)
	if err := a.tg.Send(response); err != nil {
		a.logger.Error("send main menu", "error", err)
	}
}

func (a *App) sendInline(chatID int64, text string, rows [][]telegram.InlineButton) {
	msg := tgbotapi.NewMessage(chatID, text)
	markup := telegram.BuildInlineKeyboard(rows)
	msg.ReplyMarkup = markup
	if err := a.tg.Send(msg); err != nil {
		a.logger.Error("send inline message", "error", err, "chat_id", chatID)
	}
}

func (a *App) answerCallback(callbackID, text string, alert bool) {
	cfg := tgbotapi.NewCallback(callbackID, text)
	cfg.ShowAlert = alert
	if err := a.tg.Send(cfg); err != nil {
		a.logger.Warn("answer callback", "error", err)
	}
}

func (a *App) sendText(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)
	if err := a.tg.Send(msg); err != nil {
		a.logger.Error("send message", "error", fmt.Errorf("chat=%d: %w", chatID, err))
	}
}

func callbackChatID(query *tgbotapi.CallbackQuery) (int64, bool) {
	if query == nil || query.Message == nil {
		return 0, false
	}
	return query.Message.Chat.ID, true
}

func parseAssignableRole(raw string) (enums.Role, bool) {
	switch strings.ToUpper(strings.TrimSpace(raw)) {
	case string(enums.RoleAdmin):
		return enums.RoleAdmin, true
	case string(enums.RoleModerator):
		return enums.RoleModerator, true
	default:
		return enums.RoleNone, false
	}
}

func parseTGID(raw string) (int64, error) {
	return strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
}

func normalizeLookupAction(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

func isValidLookupAction(action string) bool {
	switch action {
	case lookupActionBan, lookupActionUnban, lookupActionForceReview:
		return true
	default:
		return false
	}
}

func normalizeReasonCode(raw string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(raw))
	if trimmed == "" {
		return "OTHER"
	}
	return trimmed
}

func isValidRejectReason(code string) bool {
	for _, option := range rejectReasonOptions {
		if code == option.Code {
			return true
		}
	}
	return false
}

func renderUserLabel(tgID int64, username string) string {
	name := strings.TrimSpace(username)
	if name != "" {
		return "@" + name
	}
	return strconv.FormatInt(tgID, 10)
}

func renderLookupUser(user model.LookupUser) string {
	birthdate := "-"
	if user.Birthdate != nil {
		birthdate = user.Birthdate.UTC().Format("2006-01-02")
	}
	age := "-"
	if user.Age > 0 {
		age = strconv.Itoa(user.Age)
	}
	goals := "-"
	if len(user.Goals) > 0 {
		goals = strings.Join(user.Goals, ", ")
	}
	languages := "-"
	if len(user.Languages) > 0 {
		languages = strings.Join(user.Languages, ", ")
	}
	plus := "-"
	if user.PlusExpiresAt != nil {
		plus = user.PlusExpiresAt.UTC().Format(time.RFC3339)
	}
	boost := "-"
	if user.BoostUntil != nil {
		boost = user.BoostUntil.UTC().Format(time.RFC3339)
	}
	banned := "false"
	if user.IsBanned {
		banned = "true"
	}

	return strings.Join([]string{
		"Find user:",
		fmt.Sprintf("user_id: %d", user.UserID),
		fmt.Sprintf("tg_id: %d", user.TGID),
		fmt.Sprintf("username: %s", defaultText(user.Username, "-")),
		fmt.Sprintf("city_id: %s", defaultText(user.CityID, "-")),
		fmt.Sprintf("birthdate: %s", birthdate),
		fmt.Sprintf("age: %s", age),
		fmt.Sprintf("gender: %s", defaultText(user.Gender, "-")),
		fmt.Sprintf("looking_for: %s", defaultText(user.LookingFor, "-")),
		fmt.Sprintf("goals: %s", goals),
		fmt.Sprintf("languages: %s", languages),
		fmt.Sprintf("occupation: %s", defaultText(user.Occupation, "-")),
		fmt.Sprintf("education: %s", defaultText(user.Education, "-")),
		fmt.Sprintf("moderation_status: %s", defaultText(user.ModerationStatus, "-")),
		fmt.Sprintf("approved: %t", user.Approved),
		fmt.Sprintf("banned: %s", banned),
		fmt.Sprintf("ban_reason: %s", defaultText(user.BanReason, "-")),
		fmt.Sprintf("plus_expires_at: %s", plus),
		fmt.Sprintf("boost_until: %s", boost),
		fmt.Sprintf("superlike_credits: %d", user.SuperlikeCredits),
		fmt.Sprintf("reveal_credits: %d", user.RevealCredits),
		fmt.Sprintf("like_tokens: %d", user.LikeTokens),
	}, "\n")
}

func splitByLength(lines []string, maxLen int) []string {
	if maxLen <= 0 {
		maxLen = 3500
	}
	chunks := make([]string, 0, 1)
	current := strings.Builder{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if current.Len() == 0 {
			current.WriteString(line)
			continue
		}

		if current.Len()+1+len(line) > maxLen {
			chunks = append(chunks, current.String())
			current.Reset()
			current.WriteString(line)
			continue
		}

		current.WriteString("\n")
		current.WriteString(line)
	}
	if current.Len() > 0 {
		chunks = append(chunks, current.String())
	}
	return chunks
}

func setLookupInputState(a *App, chatID int64, state lookupInputState) {
	a.lookupInputMu.Lock()
	defer a.lookupInputMu.Unlock()
	a.lookupInputByChat[chatID] = state
}

func deleteLookupInputState(a *App, chatID int64) {
	a.lookupInputMu.Lock()
	defer a.lookupInputMu.Unlock()
	delete(a.lookupInputByChat, chatID)
}

func setLookupSessionState(a *App, chatID int64, state lookupSessionState) {
	a.lookupSessionMu.Lock()
	defer a.lookupSessionMu.Unlock()
	a.lookupSessionByChat[chatID] = state
}

func getLookupSessionState(a *App, chatID int64) (lookupSessionState, bool) {
	a.lookupSessionMu.Lock()
	defer a.lookupSessionMu.Unlock()
	state, ok := a.lookupSessionByChat[chatID]
	return state, ok
}

func deleteLookupSessionState(a *App, chatID int64) {
	a.lookupSessionMu.Lock()
	defer a.lookupSessionMu.Unlock()
	delete(a.lookupSessionByChat, chatID)
}

func renderModerationQueueItem(item model.ModerationQueueItem) string {
	profile := item.Profile
	goals := "-"
	if len(profile.Goals) > 0 {
		goals = strings.Join(profile.Goals, ", ")
	}
	languages := "-"
	if len(profile.Languages) > 0 {
		languages = strings.Join(profile.Languages, ", ")
	}
	birthdate := "-"
	if profile.Birthdate != nil {
		birthdate = profile.Birthdate.UTC().Format("2006-01-02")
	}
	age := "-"
	if profile.Age > 0 {
		age = strconv.Itoa(profile.Age)
	}

	lines := []string{
		fmt.Sprintf("Анкета в модерации #%d", item.ModerationItemID),
		fmt.Sprintf("user_id: %d", profile.UserID),
		fmt.Sprintf("tg_id: %d", profile.TGID),
		fmt.Sprintf("username: %s", defaultText(profile.Username, "-")),
		fmt.Sprintf("city_id: %s", defaultText(profile.CityID, "-")),
		fmt.Sprintf("birthdate: %s", birthdate),
		fmt.Sprintf("age: %s", age),
		fmt.Sprintf("gender: %s", defaultText(profile.Gender, "-")),
		fmt.Sprintf("looking_for: %s", defaultText(profile.LookingFor, "-")),
		fmt.Sprintf("goals: %s", goals),
		fmt.Sprintf("languages: %s", languages),
		fmt.Sprintf("occupation: %s", defaultText(profile.Occupation, "-")),
		fmt.Sprintf("education: %s", defaultText(profile.Education, "-")),
		fmt.Sprintf("eta_bucket: %s", defaultText(item.ETABucket, "-")),
		fmt.Sprintf("created_at: %s", item.CreatedAt.UTC().Format(time.RFC3339)),
		fmt.Sprintf("locked_at: %s", item.LockedAt.UTC().Format(time.RFC3339)),
	}

	return strings.Join(lines, "\n")
}

func defaultText(value string, fallback string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}
