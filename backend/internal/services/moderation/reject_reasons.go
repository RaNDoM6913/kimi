package moderation

import (
	"sort"
	"strings"
)

type RejectReasonItem struct {
	ReasonCode      string
	Label           string
	ReasonText      string
	RequiredFixStep string
}

type rejectReasonTemplate struct {
	Label           string
	ReasonText      string
	RequiredFixStep string
}

// Templates are copied 1:1 from tgbots/bot_moderator rejectTemplates (MVP sync).
var rejectReasonTemplates = map[string]rejectReasonTemplate{
	"PHOTO_NO_FACE": {
		Label:           "Фото: не видно лица",
		ReasonText:      "На фото не видно лица.",
		RequiredFixStep: "Загрузите фото, где лицо хорошо различимо.",
	},
	"PHOTO_FAKE_NOT_YOU": {
		Label:           "Фото: не вы",
		ReasonText:      "Фото не соответствует владельцу анкеты.",
		RequiredFixStep: "Загрузите ваши реальные фотографии без чужих изображений.",
	},
	"PHOTO_PROHIBITED": {
		Label:           "Фото: запрещенный контент",
		ReasonText:      "Обнаружен запрещенный фото-контент.",
		RequiredFixStep: "Удалите запрещённый контент и загрузите новые фото.",
	},
	"CIRCLE_MISMATCH": {
		Label:           "Кружок: не совпадает",
		ReasonText:      "Кружок не совпадает с фотографиями анкеты.",
		RequiredFixStep: "Перезапишите кружок, чтобы внешность совпадала с фото.",
	},
	"CIRCLE_FAILED": {
		Label:           "Кружок: не прошел проверку",
		ReasonText:      "Кружок не прошел проверку качества.",
		RequiredFixStep: "Перезапишите кружок при хорошем освещении и стабильной связи.",
	},
	"PROFILE_INCOMPLETE": {
		Label:           "Профиль: не заполнен",
		ReasonText:      "Профиль заполнен не полностью.",
		RequiredFixStep: "Заполните обязательные поля профиля и отправьте на модерацию повторно.",
	},
	"SPAM_ADS_LINKS": {
		Label:           "Спам/реклама/ссылки",
		ReasonText:      "Обнаружены признаки спама, рекламы или внешних ссылок.",
		RequiredFixStep: "Удалите спам/рекламу/ссылки из профиля и отправьте на модерацию повторно.",
	},
	"BOT_SUSPECT": {
		Label:           "Подозрение на бота",
		ReasonText:      "Профиль помечен как подозрительный на автоматизацию.",
		RequiredFixStep: "Обновите анкету и пройдите повторную модерацию вручную.",
	},
	"OTHER": {
		Label:           "Другое",
		ReasonText:      "Требуется корректировка анкеты.",
		RequiredFixStep: "Обновите анкету по замечанию модератора и отправьте на модерацию повторно.",
	},
}

func (s *Service) ListRejectReasons() []RejectReasonItem {
	codes := make([]string, 0, len(allowedRejectReasonCodes))
	for code := range allowedRejectReasonCodes {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	items := make([]RejectReasonItem, 0, len(codes))
	for _, code := range codes {
		template, ok := rejectReasonTemplates[code]
		if !ok {
			items = append(items, RejectReasonItem{
				ReasonCode:      code,
				Label:           defaultRejectReasonLabel(code),
				ReasonText:      "",
				RequiredFixStep: "",
			})
			continue
		}
		items = append(items, RejectReasonItem{
			ReasonCode:      code,
			Label:           strings.TrimSpace(template.Label),
			ReasonText:      strings.TrimSpace(template.ReasonText),
			RequiredFixStep: strings.TrimSpace(template.RequiredFixStep),
		})
	}

	return items
}

func defaultRejectReasonLabel(reasonCode string) string {
	normalized := strings.TrimSpace(reasonCode)
	if normalized == "" {
		return "Other"
	}
	normalized = strings.ReplaceAll(normalized, "_", " ")
	return strings.Title(strings.ToLower(normalized))
}
