package telegram

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type UpdateHandler func(context.Context, tgbotapi.Update)

type Client struct {
	api         *tgbotapi.BotAPI
	logger      *slog.Logger
	handler     UpdateHandler
	pollTimeout int
	dryRun      bool
}

func NewClient(token string, pollTimeout int, logger *slog.Logger, handler UpdateHandler) (*Client, error) {
	if handler == nil {
		return nil, errors.New("telegram update handler is required")
	}
	if logger == nil {
		logger = slog.Default()
	}

	if strings.TrimSpace(token) == "" {
		return &Client{
			logger:      logger,
			handler:     handler,
			pollTimeout: pollTimeout,
			dryRun:      true,
		}, nil
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}

	return &Client{
		api:         api,
		logger:      logger,
		handler:     handler,
		pollTimeout: pollTimeout,
	}, nil
}

func (c *Client) Start(ctx context.Context) error {
	if c.dryRun {
		c.logger.Warn("BOT_TOKEN is empty, running in dry mode")
		<-ctx.Done()
		return nil
	}

	timeout := c.pollTimeout
	if timeout <= 0 {
		timeout = 30
	}

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = timeout
	updates := c.api.GetUpdatesChan(updateConfig)

	for {
		select {
		case <-ctx.Done():
			c.api.StopReceivingUpdates()
			return nil
		case update, ok := <-updates:
			if !ok {
				return nil
			}
			c.handler(ctx, update)
		}
	}
}

func (c *Client) Send(msg tgbotapi.Chattable) error {
	if c.dryRun {
		return nil
	}
	_, err := c.api.Send(msg)
	return err
}
