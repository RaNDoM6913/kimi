package apiapp

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"github.com/ivankudzin/tgapp/backend/internal/config"
	pgrepo "github.com/ivankudzin/tgapp/backend/internal/repo/postgres"
	redrepo "github.com/ivankudzin/tgapp/backend/internal/repo/redis"
	adssvc "github.com/ivankudzin/tgapp/backend/internal/services/ads"
	analyticsvc "github.com/ivankudzin/tgapp/backend/internal/services/analytics"
	antiabusesvc "github.com/ivankudzin/tgapp/backend/internal/services/antiabuse"
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
	swipesvc "github.com/ivankudzin/tgapp/backend/internal/services/swipes"
	userssvc "github.com/ivankudzin/tgapp/backend/internal/services/users"
	httperrors "github.com/ivankudzin/tgapp/backend/internal/transport/http/errors"
	"github.com/ivankudzin/tgapp/backend/internal/transport/http/handlers"
)

type Dependencies struct {
	AdsService         *adssvc.Service
	AntiAbuseService   *antiabusesvc.Service
	AntiAbuseDashboard *redrepo.AntiAbuseDashboardRepo
	AnalyticsService   *analyticsvc.Service
	EntitlementService *entsvc.Service
	AuthService        *authsvc.Service
	DailyMetricsRepo   *pgrepo.DailyMetricsRepo
	FeedService        *feedsvc.Service
	GeoService         *geosvc.Service
	LikeService        *likessvc.Service
	MatchService       *matchessvc.Service
	MediaService       *mediasvc.Service
	ModerationService  *modsvc.Service
	PaymentService     *paymentsvc.Service
	ProfileService     *profilesvc.Service
	SwipeService       *swipesvc.Service
	UserService        *userssvc.Service
	Logger             *zap.Logger
	Config             config.Config
}

func RegisterRoutes(r chi.Router, deps Dependencies) {
	adsHandler := handlers.NewAdsHandler(deps.AdsService)
	authHandler := handlers.NewAuthHandler(deps.AuthService)
	healthHandler := handlers.NewHealthHandler()
	meHandler := handlers.NewMeHandler(deps.Config.Remote, deps.AntiAbuseService)
	configHandler := handlers.NewConfigHandler(deps.Config.Remote)
	locationHandler := handlers.NewLocationHandler(deps.GeoService)
	profileHandler := handlers.NewProfileHandler(deps.ProfileService)
	mediaHandler := handlers.NewMediaHandler(deps.MediaService)
	moderationHandler := handlers.NewModerationHandler(deps.ModerationService)
	quotaHandler := handlers.NewQuotaHandler(deps.LikeService)
	feedHandler := handlers.NewFeedHandler(deps.FeedService)
	candidateHandler := handlers.NewCandidateHandler(deps.FeedService, deps.MediaService, deps.AnalyticsService)
	swipeHandler := handlers.NewSwipeHandler(deps.SwipeService)
	rewindHandler := handlers.NewRewindHandler(deps.SwipeService)
	boostHandler := handlers.NewBoostHandler()
	likesHandler := handlers.NewLikesHandler(deps.LikeService)
	matchesHandler := handlers.NewMatchesHandler(deps.MatchService)
	dmHandler := handlers.NewDMHandler()
	partnersHandler := handlers.NewPartnersHandler()
	settingsHandler := handlers.NewSettingsHandler()
	travelHandler := handlers.NewTravelHandler()
	purchaseHandler := handlers.NewPurchaseHandler(deps.PaymentService, deps.EntitlementService)
	eventsHandler := handlers.NewEventsHandler(deps.AnalyticsService)
	adminHandler := handlers.NewAdminHandler(deps.UserService, deps.AnalyticsService)
	adminHandler.AttachDailyMetrics(deps.DailyMetricsRepo)
	adminHandler.AttachAntiAbuseDashboard(deps.AntiAbuseDashboard)
	adminBotModerationHandler := handlers.NewAdminBotModerationHandler(deps.ModerationService, deps.AnalyticsService)
	adminBotUsersHandler := handlers.NewAdminBotUsersHandler(deps.UserService, deps.AnalyticsService)
	authMW := AuthMiddleware(deps.AuthService, deps.Logger)
	adminHealthRoleMW := RequireRole("OWNER", "SUPPORT", "MODERATOR")
	adminPrivateRoleMW := RequireRole("OWNER", "SUPPORT")
	adminMetricsRoleMW := RequireRole("OWNER", "SUPPORT")
	devPayRoleMW := RequireRole("OWNER")
	adminBotAuthMW := AdminBotAuthMiddleware(deps.Config.Admin, deps.Logger)
	adminBotNotImplemented := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		httperrors.Write(w, http.StatusNotImplemented, httperrors.APIError{
			Code:    "NOT_IMPLEMENTED",
			Message: "admin bot endpoint is not implemented",
		})
	})

	r.Get("/healthz", healthHandler.Get)
	r.Route("/admin/bot", func(r chi.Router) {
		r.Use(adminBotAuthMW)
		r.Post("/mod/queue/acquire", adminBotModerationHandler.QueueAcquire)
		r.Post("/mod/items/{id}/approve", adminBotModerationHandler.Approve)
		r.Post("/mod/items/{id}/reject", adminBotModerationHandler.Reject)
		r.Get("/lookup/user", adminBotUsersHandler.LookupUser)
		r.Post("/users/{id}/ban", adminBotUsersHandler.BanUser)
		r.Post("/users/{id}/unban", adminBotUsersHandler.UnbanUser)
		r.Post("/users/{id}/force-review", adminBotUsersHandler.ForceReview)
		r.Handle("/", adminBotNotImplemented)
		r.Handle("/*", adminBotNotImplemented)
	})
	r.Route("/admin", func(r chi.Router) {
		r.With(authMW, adminHealthRoleMW).Get("/health", adminHandler.Health)
		r.With(authMW, adminPrivateRoleMW).Get("/users/{id}/private", adminHandler.UserPrivate)
		r.With(authMW, adminMetricsRoleMW).Get("/metrics/daily", adminHandler.MetricsDaily)
		r.With(authMW, adminMetricsRoleMW).Get("/antiabuse/summary", adminHandler.AntiAbuseSummary)
		r.With(authMW, adminMetricsRoleMW).Get("/antiabuse/top", adminHandler.AntiAbuseTop)
	})
	r.Get("/config", configHandler.Handle)
	r.With(authMW).Post("/profile/location", locationHandler.Handle)
	r.With(authMW).Post("/profile/core", profileHandler.Core)
	r.With(authMW).Post("/media/photo", mediaHandler.PhotoUpload)
	r.With(authMW).Get("/media/photos", mediaHandler.PhotosList)
	r.With(authMW).Get("/moderation/status", moderationHandler.Handle)
	r.With(authMW).Get("/quota", quotaHandler.Handle)
	r.With(authMW).Post("/swipe", swipeHandler.Handle)
	r.With(authMW).Get("/feed", feedHandler.Handle)
	r.With(authMW).Post("/rewind", rewindHandler.Handle)
	r.With(authMW).Get("/likes/incoming", likesHandler.Incoming)
	r.With(authMW).Post("/likes/reveal_one", likesHandler.RevealOne)
	r.With(authMW).Get("/matches", matchesHandler.Handle)
	r.With(authMW).Post("/unmatch", matchesHandler.Unmatch)
	r.With(authMW).Post("/block", matchesHandler.Block)
	r.With(authMW).Post("/report", matchesHandler.Report)
	r.With(authMW).Post("/ads/impression", adsHandler.Impression)
	r.With(authMW).Post("/ads/click", adsHandler.Click)
	r.With(authMW).Post("/purchase/create", purchaseHandler.Create)
	r.Post("/purchase/webhook", purchaseHandler.Webhook)
	r.With(authMW).Get("/entitlements", purchaseHandler.Entitlements)
	r.With(authMW, devPayRoleMW).Post("/pay/dev/begin", purchaseHandler.DevBegin)
	r.With(authMW, devPayRoleMW).Post("/pay/dev/confirm", purchaseHandler.DevConfirm)
	r.Post("/events/batch", eventsHandler.Batch)

	r.Route("/auth", func(r chi.Router) {
		r.Post("/telegram", authHandler.Telegram)
		r.Post("/refresh", authHandler.Refresh)
		r.With(authMW).Post("/logout", authHandler.Logout)
		r.With(authMW).Post("/logout_all", authHandler.LogoutAll)
	})

	r.Route("/v1/auth", func(r chi.Router) {
		r.Post("/telegram", authHandler.Telegram)
		r.Post("/refresh", authHandler.Refresh)
		r.With(authMW).Post("/logout", authHandler.Logout)
		r.With(authMW).Post("/logout_all", authHandler.LogoutAll)
	})

	r.Route("/v1", func(r chi.Router) {
		r.With(authMW).Get("/me", meHandler.Handle)
		r.Get("/config", configHandler.Handle)
		r.With(authMW).Post("/location", locationHandler.Handle)
		r.With(authMW).Post("/profile/location", locationHandler.Handle)
		r.With(authMW).Post("/profile/core", profileHandler.Core)
		r.Get("/profile", profileHandler.Handle)
		r.Put("/profile", profileHandler.Handle)
		r.With(authMW).Post("/media/upload", mediaHandler.PhotoUpload)
		r.With(authMW).Post("/media/photo", mediaHandler.PhotoUpload)
		r.With(authMW).Get("/media/photos", mediaHandler.PhotosList)
		r.With(authMW).Get("/moderation/status", moderationHandler.Handle)
		r.With(authMW).Get("/quota", quotaHandler.Handle)
		r.With(authMW).Get("/feed", feedHandler.Handle)
		r.With(authMW).Get("/candidates/{user_id}/profile", candidateHandler.Profile)
		r.With(authMW).Get("/candidates/{user_id}/media/photos", candidateHandler.Photos)
		r.With(authMW).Post("/swipes", swipeHandler.Handle)
		r.With(authMW).Post("/rewind", rewindHandler.Handle)
		r.Post("/boost", boostHandler.Handle)
		r.With(authMW).Get("/likes", likesHandler.Handle)
		r.With(authMW).Get("/likes/incoming", likesHandler.Incoming)
		r.With(authMW).Post("/likes/reveal_one", likesHandler.RevealOne)
		r.With(authMW).Get("/matches", matchesHandler.Handle)
		r.With(authMW).Post("/unmatch", matchesHandler.Unmatch)
		r.With(authMW).Post("/block", matchesHandler.Block)
		r.With(authMW).Post("/report", matchesHandler.Report)
		r.With(authMW).Post("/ads/impression", adsHandler.Impression)
		r.With(authMW).Post("/ads/click", adsHandler.Click)
		r.Post("/dm/invite", dmHandler.Handle)
		r.Get("/partners", partnersHandler.Handle)
		r.Get("/settings", settingsHandler.Handle)
		r.Post("/travel", travelHandler.Handle)
		r.With(authMW).Post("/purchase", purchaseHandler.Handle)
		r.With(authMW).Post("/purchase/create", purchaseHandler.Create)
		r.Post("/purchase/webhook", purchaseHandler.Webhook)
		r.With(authMW).Get("/entitlements", purchaseHandler.Entitlements)
		r.With(authMW, devPayRoleMW).Post("/pay/dev/begin", purchaseHandler.DevBegin)
		r.With(authMW, devPayRoleMW).Post("/pay/dev/confirm", purchaseHandler.DevConfirm)
		r.Post("/events", eventsHandler.Handle)
		r.Post("/events/batch", eventsHandler.Batch)
	})
}
