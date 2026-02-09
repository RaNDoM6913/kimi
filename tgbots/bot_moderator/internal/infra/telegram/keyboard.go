package telegram

import tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"

type InlineButton struct {
	Text string
	Data string
}

func BuildReplyKeyboard(rows [][]string) tgbotapi.ReplyKeyboardMarkup {
	keyboardRows := make([][]tgbotapi.KeyboardButton, 0, len(rows))
	for _, row := range rows {
		buttons := make([]tgbotapi.KeyboardButton, 0, len(row))
		for _, title := range row {
			buttons = append(buttons, tgbotapi.NewKeyboardButton(title))
		}
		keyboardRows = append(keyboardRows, buttons)
	}

	keyboard := tgbotapi.NewReplyKeyboard(keyboardRows...)
	keyboard.ResizeKeyboard = true
	keyboard.OneTimeKeyboard = false
	keyboard.Selective = true
	return keyboard
}

func RemoveKeyboard() tgbotapi.ReplyKeyboardRemove {
	return tgbotapi.NewRemoveKeyboard(true)
}

func BuildInlineKeyboard(rows [][]InlineButton) tgbotapi.InlineKeyboardMarkup {
	keyboardRows := make([][]tgbotapi.InlineKeyboardButton, 0, len(rows))
	for _, row := range rows {
		buttons := make([]tgbotapi.InlineKeyboardButton, 0, len(row))
		for _, button := range row {
			buttons = append(buttons, tgbotapi.NewInlineKeyboardButtonData(button.Text, button.Data))
		}
		keyboardRows = append(keyboardRows, buttons)
	}
	return tgbotapi.NewInlineKeyboardMarkup(keyboardRows...)
}
