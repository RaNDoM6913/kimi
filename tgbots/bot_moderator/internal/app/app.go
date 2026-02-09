package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"

	"bot_moderator/internal/config"
	s3infra "bot_moderator/internal/infra/s3"
	"bot_moderator/internal/infra/telegram"
	"bot_moderator/internal/repo/postgres"
	"bot_moderator/internal/services/access"
	"bot_moderator/internal/services/audit"
	"bot_moderator/internal/services/bans"
	exportsvc "bot_moderator/internal/services/export"
	"bot_moderator/internal/services/lookup"
	"bot_moderator/internal/services/moderation"
	statssvc "bot_moderator/internal/services/stats"
	systemsvc "bot_moderator/internal/services/system"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
	db     *sql.DB
	tg     *telegram.Client

	accessService     *access.Service
	moderationService *moderation.Service
	lookupService     *lookup.Service
	bansService       *bans.Service
	auditService      *audit.Service
	exportService     *exportsvc.Service
	statsService      *statssvc.Service
	systemService     *systemsvc.Service

	rejectMu     sync.Mutex
	rejectByChat map[int64]rejectCommentState

	lookupInputMu     sync.Mutex
	lookupInputByChat map[int64]lookupInputState

	lookupSessionMu     sync.Mutex
	lookupSessionByChat map[int64]lookupSessionState
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	if logger == nil {
		logger = slog.Default()
	}

	db, err := postgres.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Warn("postgres unavailable, continuing without database", "error", err)
		db = nil
	}

	botUsersRepo := postgres.NewBotUsersRepo(db)
	botRolesRepo := postgres.NewBotRolesRepo(db)
	moderationRepo := postgres.NewModerationRepo(db)

	var signer *s3infra.Signer
	if strings.TrimSpace(cfg.S3Endpoint) != "" && strings.TrimSpace(cfg.S3Bucket) != "" {
		signer, err = s3infra.NewSigner(cfg.S3Endpoint, cfg.S3AccessKey, cfg.S3SecretKey, cfg.S3Bucket, cfg.S3UseSSL)
		if err != nil {
			logger.Warn("s3 signer unavailable, media urls will be disabled", "error", err)
		}
	} else {
		logger.Warn("s3 signer is disabled: missing S3_ENDPOINT or S3_BUCKET")
	}

	app := &App{
		cfg:                 cfg,
		logger:              logger,
		db:                  db,
		accessService:       access.NewService(cfg.OwnerTGID, botUsersRepo, botRolesRepo),
		moderationService:   moderation.NewService(moderationRepo, signer),
		lookupService:       lookup.NewService(postgres.NewUsersLookupRepo(db), signer),
		bansService:         bans.NewService(postgres.NewBansRepo(db)),
		auditService:        audit.NewService(postgres.NewAuditRepo(db)),
		exportService:       exportsvc.NewService(postgres.NewExportsQueueRepo(db)),
		statsService:        statssvc.NewService(postgres.NewWorkStatsRepo(db)),
		systemService:       systemsvc.NewService(postgres.NewSystemRepo(db)),
		rejectByChat:        make(map[int64]rejectCommentState),
		lookupInputByChat:   make(map[int64]lookupInputState),
		lookupSessionByChat: make(map[int64]lookupSessionState),
	}

	app.tg, err = telegram.NewClient(cfg.BotToken, cfg.PollTimeoutSeconds, logger, app.routeUpdate)
	if err != nil {
		if db != nil {
			_ = db.Close()
		}
		return nil, fmt.Errorf("create telegram client: %w", err)
	}

	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	defer a.close()
	return a.tg.Start(ctx)
}

func (a *App) close() {
	if a.db != nil {
		if err := a.db.Close(); err != nil {
			a.logger.Error("close postgres", "error", err)
		}
	}
}
