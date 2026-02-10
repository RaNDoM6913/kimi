package app

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"bot_moderator/internal/config"
	s3infra "bot_moderator/internal/infra/s3"
	"bot_moderator/internal/infra/telegram"
	"bot_moderator/internal/repo/adminhttp"
	"bot_moderator/internal/repo/dualrepo"
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

	adminMode := normalizeAdminMode(cfg.AdminMode)

	db, err := postgres.Open(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Warn("postgres unavailable, continuing without database", "error", err)
		db = nil
	}

	botUsersRepo := postgres.NewBotUsersRepo(db)
	botRolesRepo := postgres.NewBotRolesRepo(db)
	moderationRepo := postgres.NewModerationRepo(db)
	usersLookupRepo := postgres.NewUsersLookupRepo(db)
	bansRepo := postgres.NewBansRepo(db)
	auditRepo := postgres.NewAuditRepo(db)
	exportsRepo := postgres.NewExportsQueueRepo(db)
	workStatsRepo := postgres.NewWorkStatsRepo(db)
	systemRepo := postgres.NewSystemRepo(db)

	useHTTPRepos := adminMode == "http" || adminMode == "dual"
	dualFallback := adminMode == "dual"

	var adminHTTPClient *adminhttp.Client
	if useHTTPRepos {
		if !cfg.IsHTTPEnabled() {
			if adminMode == "http" {
				return nil, fmt.Errorf("admin http mode requires ADMIN_API_URL and ADMIN_BOT_TOKEN")
			}
			useHTTPRepos = false
			dualFallback = false
			logger.Info("admin http disabled by config, using db repositories")
		} else {
			adminHTTPClient, err = adminhttp.NewClient(
				cfg.AdminAPIURL,
				cfg.AdminBotToken,
				time.Duration(cfg.AdminHTTPTimeout)*time.Second,
			)
			if err != nil {
				if adminMode == "http" {
					return nil, fmt.Errorf("create admin http client: %w", err)
				}
				useHTTPRepos = false
				dualFallback = false
				logger.Warn("admin http unavailable in dual mode, using db repositories", "error", err)
			}
		}
	}

	var accessUsersRepo access.UsersRepo = botUsersRepo
	var accessRolesRepo access.RolesRepo = botRolesRepo
	var moderationServiceRepo moderation.Repo = dualrepo.NewModerationRepo(nil, moderationRepo, adminMode)
	var lookupRepo lookup.Repo = usersLookupRepo
	var bansServiceRepo bans.Repo = bansRepo
	var auditServiceRepo audit.Repo = auditRepo
	var exportServiceRepo exportsvc.Repo = exportsRepo
	var statsServiceRepo statssvc.Repo = workStatsRepo
	var systemServiceRepo systemsvc.Repo = systemRepo

	if useHTTPRepos {
		accessUsersRepo = adminhttp.NewAccessUsersRepo(adminHTTPClient, botUsersRepo, dualFallback)
		accessRolesRepo = adminhttp.NewAccessRolesRepo(adminHTTPClient, botRolesRepo, dualFallback)
		httpModerationRepo := adminhttp.NewModerationRepo(adminHTTPClient, moderationRepo, dualFallback)
		moderationServiceRepo = dualrepo.NewModerationRepo(httpModerationRepo, moderationRepo, adminMode)
		lookupRepo = adminhttp.NewUsersLookupRepo(adminHTTPClient, usersLookupRepo, dualFallback)
		bansServiceRepo = adminhttp.NewBansRepo(adminHTTPClient, bansRepo, dualFallback)
		auditServiceRepo = adminhttp.NewAuditRepo(adminHTTPClient, auditRepo, dualFallback)
		exportServiceRepo = adminhttp.NewExportsQueueRepo(adminHTTPClient, exportsRepo, dualFallback)
		statsServiceRepo = adminhttp.NewWorkStatsRepo(adminHTTPClient, workStatsRepo, dualFallback)
		systemServiceRepo = adminhttp.NewSystemRepo(adminHTTPClient, systemRepo, dualFallback)
	}

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
		accessService:       access.NewService(cfg.OwnerTGID, accessUsersRepo, accessRolesRepo),
		moderationService:   moderation.NewService(moderationServiceRepo, signer),
		lookupService:       lookup.NewService(lookupRepo, signer),
		bansService:         bans.NewService(bansServiceRepo),
		auditService:        audit.NewService(auditServiceRepo),
		exportService:       exportsvc.NewService(exportServiceRepo),
		statsService:        statssvc.NewService(statsServiceRepo),
		systemService:       systemsvc.NewService(systemServiceRepo),
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

func normalizeAdminMode(raw string) string {
	mode := strings.ToLower(strings.TrimSpace(raw))
	switch mode {
	case "db", "http", "dual":
		return mode
	default:
		return "dual"
	}
}
