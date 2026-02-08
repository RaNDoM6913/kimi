package apiapp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/minio/minio-go/v7"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/ivankudzin/tgapp/backend/internal/config"
	s3infra "github.com/ivankudzin/tgapp/backend/internal/infra/s3"
	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	adssvc "github.com/ivankudzin/tgapp/backend/internal/services/ads"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	authsvc "github.com/ivankudzin/tgapp/backend/internal/services/auth"
	entsvc "github.com/ivankudzin/tgapp/backend/internal/services/entitlements"
	feedsvc "github.com/ivankudzin/tgapp/backend/internal/services/feed"
	geosvc "github.com/ivankudzin/tgapp/backend/internal/services/geo"
	likessvc "github.com/ivankudzin/tgapp/backend/internal/services/likes"
	matchessvc "github.com/ivankudzin/tgapp/backend/internal/services/matches"
	mediasvc "github.com/ivankudzin/tgapp/backend/internal/services/media"
	modsvc "github.com/ivankudzin/tgapp/backend/internal/services/moderation"
	paymentsvc "github.com/ivankudzin/tgapp/backend/internal/services/payments"
	profilesvc "github.com/ivankudzin/tgapp/backend/internal/services/profiles"
	ratesvc "github.com/ivankudzin/tgapp/backend/internal/services/rate"
	swipesvc "github.com/ivankudzin/tgapp/backend/internal/services/swipes"
)

type App struct {
	cfg        config.Config
	logger     *zap.Logger
	server     *http.Server
	postgres   *pgxpool.Pool
	redis      *goredis.Client
	s3         *minio.Client
	httpRouter http.Handler
}

func New(ctx context.Context, cfg config.Config, log *zap.Logger) (*App, error) {
	if log == nil {
		return nil, fmt.Errorf("logger is nil")
	}

	r := chi.NewRouter()
	ApplyMiddlewares(r, log)

	var pool *pgxpool.Pool
	if p, err := pgrepo.NewPool(ctx, cfg.Postgres.DSN); err != nil {
		log.Warn("postgres init failed, continuing in degraded mode", zap.Error(err))
	} else {
		pool = p
	}

	redisClient := redrepo.NewClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	sessionRepo := redrepo.NewSessionRepo(redisClient)
	rateRepo := redrepo.NewRateRepo(redisClient)
	adRepo := pgrepo.NewAdRepo(pool)
	feedRepo := pgrepo.NewFeedRepo(pool)
	swipeRepo := pgrepo.NewSwipeRepo(pool)
	likeRepo := pgrepo.NewLikeRepo(pool)
	matchRepo := pgrepo.NewMatchRepo(pool)
	blockRepo := pgrepo.NewBlockRepo(pool)
	reportRepo := pgrepo.NewReportRepo(pool)
	profileRepo := pgrepo.NewProfileRepo(pool)
	mediaRepo := pgrepo.NewMediaRepo(pool)
	moderationRepo := pgrepo.NewModerationRepo(pool)
	quotaRepo := pgrepo.NewQuotaRepo(pool)
	entitlementRepo := pgrepo.NewEntitlementRepo(pool)
	eventRepo := pgrepo.NewEventRepo(pool)
	purchaseRepo := pgrepo.NewPurchaseRepo(pool)
	jwtManager := authsvc.NewJWTManager(cfg.Auth.JWTSecret, cfg.Auth.JWTAccessTTL)
	authService := authsvc.NewService(jwtManager, sessionRepo, cfg.Auth.RefreshTTL)
	geoService := geosvc.NewService(cfg.Remote.Cities, profileRepo)
	feedService := feedsvc.NewService(feedRepo, feedsvc.Config{
		DefaultAgeMin:   cfg.Remote.Filters.AgeMin,
		DefaultAgeMax:   cfg.Remote.Filters.AgeMax,
		DefaultRadiusKM: cfg.Remote.Filters.RadiusDefaultKM,
		MaxRadiusKM:     cfg.Remote.Filters.RadiusMaxKM,
	})
	feedService.AttachAds(adRepo, entitlementRepo, feedsvc.AdsConfig{
		FreeEvery:     cfg.Remote.AdsInject.FreeEvery,
		PlusEvery:     cfg.Remote.AdsInject.PlusEvery,
		DefaultIsPlus: cfg.Remote.MeDefaults.IsPlus,
	})
	adsService := adssvc.NewService(adRepo)
	entitlementService := entsvc.NewService(entitlementRepo, entsvc.Config{
		DefaultIsPlus: cfg.Remote.MeDefaults.IsPlus,
	})
	paymentService := paymentsvc.NewService(paymentsvc.Dependencies{
		Purchases:    purchaseRepo,
		Entitlements: entitlementRepo,
	})
	analyticsService := analyticsvc.NewService(eventRepo, analyticsvc.Config{
		MaxBatchSize: 100,
	})
	profileService := profilesvc.NewService(profileRepo)
	rateLimiter := ratesvc.NewLimiter(
		rateRepo,
		cfg.Remote.Limits.PlusRatePerMinute,
		cfg.Remote.Limits.PlusRatePer10Seconds,
	)
	likeService := likessvc.NewService(quotaRepo, entitlementRepo, rateLimiter, likessvc.Config{
		FreeLikesPerDay: cfg.Remote.Limits.FreeLikesPerDay,
		DefaultTimezone: cfg.Remote.MeDefaults.Timezone,
		DefaultIsPlus:   cfg.Remote.MeDefaults.IsPlus,
		PlusUnlimitedUI: cfg.Remote.Limits.PlusUnlimitedUI,
	})
	likeService.AttachIncoming(pool, likeRepo, entitlementRepo)
	matchesService := matchessvc.NewService(matchessvc.Dependencies{
		Pool:        pool,
		MatchStore:  matchRepo,
		BlockStore:  blockRepo,
		ReportStore: reportRepo,
	})
	swipeService := swipesvc.NewService(swipesvc.Dependencies{
		Pool:         pool,
		SwipeStore:   swipeRepo,
		LikeStore:    likeRepo,
		MatchStore:   matchRepo,
		QuotaStore:   quotaRepo,
		Entitlements: entitlementRepo,
		RateLimiter:  rateLimiter,
		QuotaView:    likeService,
	}, swipesvc.Config{
		FreeLikesPerDay:   cfg.Remote.Limits.FreeLikesPerDay,
		FreeRewindsPerDay: 1,
		PlusRewindsPerDay: cfg.Remote.Limits.PlusRewindsPerDay,
		DefaultTimezone:   cfg.Remote.MeDefaults.Timezone,
		DefaultIsPlus:     cfg.Remote.MeDefaults.IsPlus,
	})

	server := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      r,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}

	var s3Client *minio.Client
	if c, err := s3infra.NewClient(s3infra.Config{
		Endpoint:  cfg.S3.Endpoint,
		AccessKey: cfg.S3.AccessKey,
		SecretKey: cfg.S3.SecretKey,
		UseSSL:    cfg.S3.UseSSL,
	}); err != nil {
		log.Warn("s3 init failed, continuing in degraded mode", zap.Error(err))
	} else {
		s3Client = c
	}

	mediaStorage := mediasvc.NewS3Storage(s3Client, cfg.S3.Bucket)
	mediaService := mediasvc.NewService(mediaRepo, mediaStorage)
	moderationService := modsvc.NewService(moderationRepo, profileRepo, mediaRepo, mediaStorage)

	RegisterRoutes(r, Dependencies{
		AdsService:         adsService,
		AnalyticsService:   analyticsService,
		EntitlementService: entitlementService,
		AuthService:        authService,
		FeedService:        feedService,
		GeoService:         geoService,
		LikeService:        likeService,
		MatchService:       matchesService,
		MediaService:       mediaService,
		ModerationService:  moderationService,
		PaymentService:     paymentService,
		ProfileService:     profileService,
		SwipeService:       swipeService,
		Logger:             log,
		Config:             cfg,
	})

	return &App{
		cfg:        cfg,
		logger:     log,
		server:     server,
		postgres:   pool,
		redis:      redisClient,
		s3:         s3Client,
		httpRouter: r,
	}, nil
}

func (a *App) Run() error {
	a.logger.Info("api server started", zap.String("addr", a.cfg.HTTP.Addr))
	err := a.server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (a *App) Shutdown(ctx context.Context) error {
	var shutdownErr error

	if err := a.server.Shutdown(ctx); err != nil {
		shutdownErr = err
	}
	if a.postgres != nil {
		a.postgres.Close()
	}
	if a.redis != nil {
		if err := a.redis.Close(); err != nil && shutdownErr == nil {
			shutdownErr = err
		}
	}

	return shutdownErr
}

func (a *App) Handler() http.Handler {
	return a.httpRouter
}
