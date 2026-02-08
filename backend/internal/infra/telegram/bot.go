package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Bot struct {
	api        *tgbotapi.BotAPI
	httpClient *http.Client
}

type VideoNoteUpdate struct {
	ChatID   int64
	UserID   int64
	Username string
	FileID   string
}

type CommandUpdate struct {
	ChatID   int64
	UserID   int64
	Username string
	Command  string
	Args     string
}

type TextUpdate struct {
	ChatID   int64
	UserID   int64
	Username string
	Text     string
}

type CallbackUpdate struct {
	CallbackID string
	ChatID     int64
	UserID     int64
	Username   string
	Data       string
}

type Handlers struct {
	OnVideoNote func(context.Context, VideoNoteUpdate) error
	OnCommand   func(context.Context, CommandUpdate) error
	OnText      func(context.Context, TextUpdate) error
	OnCallback  func(context.Context, CallbackUpdate) error
}

func NewBot(token string) (*Bot, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("telegram bot token is empty")
	}

	api, err := tgbotapi.NewBotAPI(strings.TrimSpace(token))
	if err != nil {
		return nil, fmt.Errorf("create telegram bot api: %w", err)
	}

	return &Bot{
		api: api,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}, nil
}

func (b *Bot) Listen(ctx context.Context, handlers Handlers) error {
	if b == nil || b.api == nil {
		return fmt.Errorf("telegram bot is not initialized")
	}

	updateCfg := tgbotapi.NewUpdate(0)
	updateCfg.Timeout = 30
	updates := b.api.GetUpdatesChan(updateCfg)
	defer b.api.StopReceivingUpdates()

	for {
		select {
		case <-ctx.Done():
			return nil
		case update := <-updates:
			if update.Message != nil && update.Message.From != nil {
				if update.Message.VideoNote != nil && handlers.OnVideoNote != nil {
					err := handlers.OnVideoNote(ctx, VideoNoteUpdate{
						ChatID:   update.Message.Chat.ID,
						UserID:   update.Message.From.ID,
						Username: update.Message.From.UserName,
						FileID:   update.Message.VideoNote.FileID,
					})
					if err != nil {
						return err
					}
					continue
				}

				if update.Message.IsCommand() && handlers.OnCommand != nil {
					err := handlers.OnCommand(ctx, CommandUpdate{
						ChatID:   update.Message.Chat.ID,
						UserID:   update.Message.From.ID,
						Username: update.Message.From.UserName,
						Command:  update.Message.Command(),
						Args:     update.Message.CommandArguments(),
					})
					if err != nil {
						return err
					}
					continue
				}

				text := strings.TrimSpace(update.Message.Text)
				if text != "" && handlers.OnText != nil {
					err := handlers.OnText(ctx, TextUpdate{
						ChatID:   update.Message.Chat.ID,
						UserID:   update.Message.From.ID,
						Username: update.Message.From.UserName,
						Text:     text,
					})
					if err != nil {
						return err
					}
				}
			}

			if update.CallbackQuery != nil && update.CallbackQuery.From != nil && handlers.OnCallback != nil {
				chatID := int64(0)
				if update.CallbackQuery.Message != nil {
					chatID = update.CallbackQuery.Message.Chat.ID
				}
				err := handlers.OnCallback(ctx, CallbackUpdate{
					CallbackID: update.CallbackQuery.ID,
					ChatID:     chatID,
					UserID:     update.CallbackQuery.From.ID,
					Username:   update.CallbackQuery.From.UserName,
					Data:       update.CallbackQuery.Data,
				})
				if err != nil {
					return err
				}
			}
		}
	}
}

func (b *Bot) ListenVideoNotes(ctx context.Context, handler func(context.Context, VideoNoteUpdate) error) error {
	if handler == nil {
		return fmt.Errorf("video note handler is nil")
	}
	return b.Listen(ctx, Handlers{OnVideoNote: handler})
}

func (b *Bot) SendText(ctx context.Context, chatID int64, text string) error {
	if b == nil || b.api == nil {
		return fmt.Errorf("telegram bot is not initialized")
	}
	if chatID == 0 {
		return fmt.Errorf("chat id is required")
	}

	msg := tgbotapi.NewMessage(chatID, text)
	_, err := b.api.Send(msg)
	if err != nil {
		return fmt.Errorf("send telegram message: %w", err)
	}

	_ = ctx
	return nil
}

func (b *Bot) SendModerationQueue(ctx context.Context, chatID int64, text string, itemID int64) error {
	if b == nil || b.api == nil {
		return fmt.Errorf("telegram bot is not initialized")
	}

	approveData := "mod:approve:" + strconv.FormatInt(itemID, 10)
	rejectData := "mod:reject:" + strconv.FormatInt(itemID, 10)

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Approve", approveData),
			tgbotapi.NewInlineKeyboardButtonData("Reject", rejectData),
		),
	)

	if _, err := b.api.Send(msg); err != nil {
		return fmt.Errorf("send moderation queue message: %w", err)
	}

	_ = ctx
	return nil
}

func (b *Bot) AnswerCallback(ctx context.Context, callbackID, text string) error {
	if b == nil || b.api == nil {
		return fmt.Errorf("telegram bot is not initialized")
	}
	if strings.TrimSpace(callbackID) == "" {
		return nil
	}

	cfg := tgbotapi.NewCallback(callbackID, text)
	if _, err := b.api.Request(cfg); err != nil {
		return fmt.Errorf("answer callback query: %w", err)
	}

	_ = ctx
	return nil
}

func (b *Bot) DownloadVideoNote(ctx context.Context, fileID string) (io.ReadCloser, int64, string, string, error) {
	if b == nil || b.api == nil {
		return nil, 0, "", "", fmt.Errorf("telegram bot is not initialized")
	}
	if strings.TrimSpace(fileID) == "" {
		return nil, 0, "", "", fmt.Errorf("file id is required")
	}

	tgFile, err := b.api.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		return nil, 0, "", "", fmt.Errorf("get telegram file: %w", err)
	}

	fileURL := tgFile.Link(b.api.Token)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fileURL, nil)
	if err != nil {
		return nil, 0, "", "", fmt.Errorf("create file request: %w", err)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, 0, "", "", fmt.Errorf("download telegram file: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, 0, "", "", fmt.Errorf("unexpected telegram file status: %d", resp.StatusCode)
	}

	name := path.Base(strings.TrimSpace(tgFile.FilePath))
	if name == "." || name == "/" || name == "" {
		name = "video_note.mp4"
	}

	contentType := resp.Header.Get("Content-Type")
	if strings.TrimSpace(contentType) == "" {
		contentType = "video/mp4"
	}

	return resp.Body, resp.ContentLength, name, contentType, nil
}
