package main

import (
	"os"
	"time"

	"github.com/partspile/site/internal/ad"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/handler"
	"github.com/partspile/site/internal/imagestore"
	"github.com/partspile/site/internal/kv"
	"github.com/partspile/site/internal/logger"
	"github.com/partspile/site/internal/rockopinion"
	"github.com/partspile/site/internal/service/sms"
	"github.com/partspile/site/internal/service/turnstile"
	"github.com/partspile/site/internal/service/verify"
	"github.com/partspile/site/internal/vector"
	"github.com/sasha-s/go-deadlock"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	flogger "github.com/gofiber/fiber/v2/middleware/logger"
)

// setupApp configures the Fiber app with middleware and routes
func setupApp() *fiber.App {
	app := fiber.New(fiber.Config{
		ErrorHandler: handler.ErrorHandler,
		BodyLimit:    config.ServerBodyLimit, // Total request body size (for multiple files)
		ReadTimeout:  30 * time.Second,       // Prevent long-running requests
		//WriteTimeout: 30 * time.Second,       // Prevent long-running responses
	})

	// Must be early in middleware chain
	app.Use(handler.ConfigureHelmet())

	app.Use(limiter.New(limiter.Config{
		Max:        config.ServerRateLimitMax,
		Expiration: config.ServerRateLimitExp,
		Storage:    kv.Storage(),
		KeyGenerator: func(c *fiber.Ctx) string {
			return "srv:" + handler.VisitorIP(c)
		},
	}))

	app.Use(handler.JWTMiddleware)
	app.Use(handler.CSRFMiddleware)

	app.Use(flogger.New(flogger.Config{
		Output: logger.Writer(),
		Format: "${status} | ${latency} | ${ip} | ${method} | ${path}\n",
		CustomTags: map[string]flogger.LogFunc{
			"ip": func(output flogger.Buffer, c *fiber.Ctx,
				data *flogger.Data, extraParam string) (int, error) {
				return output.WriteString(handler.VisitorIP(c))
			},
		},
	}))

	// Public routes

	app.Static("/", "./static")

	app.Get("/", handler.HomeHandler)
	app.Get("/robots.txt", handler.RobotsHandler)
	app.Get("/sitemap.xml", handler.SitemapHandler)
	app.Get("/login", handler.LoginHandler)
	app.Get("/logout", handler.LogoutHandler)
	app.Get("/register", handler.RegisterHandler)
	app.Get("/recover", handler.RecoveryRateLimiter, handler.RecoverHandler)
	app.Get("/health", handler.HandleHealth)
	app.Get("/error", handler.ErrorPageHandler)
	app.Get("/about", handler.AboutHandler)
	app.Get("/donate", handler.DonateHandler)
	app.Get("/funstats", handler.FunStatsHandler)
	app.Get("/privacy", handler.PrivacyHandler)
	app.Get("/terms", handler.TermsHandler)
	app.Get("/faq", handler.FAQHandler)
	app.Get("/faq/:section", handler.FAQHandler)
	app.Get("/ad/:id", handler.AdHandler)
	app.Get("/ad/:id/rock/:ordinal", handler.PublicAdRockOpinionHandler)
	app.Get("/c/:category", handler.ShortCategoryHandler)
	app.Get("/u/:token", handler.SharedUserProfileHandler)
	app.Get("/u/:token/view/:view", handler.SharedUserProfileViewHandler)
	app.Get("/media/ad/:id/:index/:size", handler.MediaAdImageHandler)

	// Auth routes

	auth := app.Group("/auth", handler.AuthRequired)

	auth.Get("/sse", handler.SSEHandler)

	auth.Get("/ad/new", handler.NewAdHandler)
	auth.Get("/ad/new/price-field", handler.NewAdPriceFieldHandler)
	auth.Get("/ad/new/suggestions", handler.SuggestionsHandler)
	auth.Post("/ad/new", handler.CreateAdHandler)
	auth.Get("/ad/:id/copy", handler.CopyAdHandler)
	auth.Get("/ad/:id/flyer", handler.AdFlyerHandler)
	auth.Get("/ad/:id/edit", handler.EditAdHandler)
	auth.Post("/ad/:id/edit", handler.UpdateAdHandler)
	auth.Post("/ad/:id/presign-images", handler.PresignAdImagesHandler)
	auth.Post("/ad/:id/confirm-images", handler.ConfirmAdImagesHandler)
	auth.Get("/ad/:id/edit/suggestions", handler.EditSuggestionsHandler)
	auth.Get("/ad/:id/remove-modal", handler.AdRemoveModalHandler)
	auth.Post("/ad/:id/pause", handler.PauseAdHandler)
	auth.Delete("/ad/:id/delete", handler.DeleteAdHandler)
	auth.Post("/ad/:id/restore", handler.RestoreAdHandler)
	auth.Post("/ad/:id/renew", handler.RenewAdHandler)
	auth.Get("/ad/:id/new-conversation", handler.MessageModalHandler)
	auth.Post("/ad/:id/send", handler.SendMessageHandler)
	auth.Post("/ad/:id/rock/throw", handler.ThrowRockOnAdHandler)
	auth.Get("/ad/:id/rock/confirm", handler.ConfirmRockOnAdHandler)
	auth.Post("/ad/:id/rock/preview", handler.PreviewRockOnAdHandler)
	auth.Get("/ad/:id/rock/:ordinal", handler.AdRockConversationHandler)

	auth.Get("/user/menu", handler.UserMenuHandler)
	auth.Get("/user/share", handler.UserShareHandler)
	auth.Get("/user/myads", handler.UserMyAdsHandler)
	auth.Get("/user/myads/tab/:tab", handler.UserMyAdsTabHandler)
	auth.Get("/user/myads/tab/:tab/view/:view", handler.UserMyAdsViewHandler)
	auth.Get("/user/messages", handler.UserMessagesHandler)
	auth.Get("/user/settings", handler.UserSettingsHandler)
	auth.Post("/user/settings/notifications", handler.NotificationsToggleHandler)
	auth.Post("/user/settings/password", handler.ChangePasswordHandler)
	auth.Post("/user/settings/phone", handler.ChangePhoneRequestHandler)
	auth.Post("/user/settings/phone/verify", handler.ChangePhoneVerifyHandler)
	auth.Get("/user/settings/delete-confirm", handler.DeleteAccountConfirmModalHandler)
	auth.Post("/user/settings/delete", handler.DeleteAccountHandler)
	auth.Post("/user/settings/account-picture/presign",
		handler.PresignAccountPictureHandler)
	auth.Post("/user/settings/account-picture/confirm",
		handler.ConfirmAccountPictureHandler)
	auth.Post("/user/settings/account-picture",
		handler.SaveAccountPictureURLHandler)
	auth.Post("/user/settings/account-picture/remove",
		handler.RemoveAccountPictureHandler)
	auth.Get("/user/:id", handler.UserProfileHandler)
	auth.Get("/user/:id/view/:view", handler.UserProfileViewHandler)
	auth.Get("/user/:id/summary", handler.UserSummaryHandler)
	auth.Get("/user/:id/rock/:ordinal", handler.UserRockConversationHandler)
	auth.Get("/welcome", handler.WelcomeHandler)

	auth.Get("/conversation/:id", handler.ConversationModalHandler)
	auth.Get("/conversation/:id/rock-opinion", handler.RockOpinionHandler)
	auth.Get("/conversation/:id/rock/confirm", handler.ConfirmRockHandler)
	auth.Post("/conversation/:id/rock/preview", handler.PreviewRockHandler)
	auth.Post("/conversation/:id/send", handler.SendConversationMessageHandler)
	auth.Post("/conversation/:id/rock/throw", handler.ThrowRockHandler)
	auth.Delete("/conversation/:id/rock/unthrow", handler.UnthrowRockHandler)

	auth.Post("/bookmark/:id", handler.BookmarkHandler)
	auth.Delete("/bookmark/:id", handler.BookmarkHandler)

	// Admin routes

	admin := app.Group("/admin", handler.AdminRequired)
	admin.Get("/dashboard", handler.AdminDashboardHandler)
	admin.Get("/tab/:tab", handler.AdminTabHandler)
	admin.Get("/embeddings/user-activities", handler.AdminEmbeddingsUserActivitiesHandler)
	admin.Get("/embeddings/site-activities", handler.AdminEmbeddingsSiteActivitiesHandler)
	admin.Post("/embeddings/cache/query/clear", handler.AdminEmbeddingsClearQueryCacheHandler)
	admin.Post("/embeddings/cache/user/clear", handler.AdminEmbeddingsClearUserCacheHandler)
	admin.Post("/embeddings/cache/site/clear", handler.AdminEmbeddingsClearSiteCacheHandler)
	admin.Post("/embeddings/cache/clear-all", handler.AdminEmbeddingsClearAllCachesHandler)

	// API routes

	api := app.Group("/api")

	api.Post("/login", handler.LoginRateLimiter, handler.LoginSubmitHandler)
	api.Post("/register/step1", handler.RegistrationRateLimiter, handler.RegisterStep1Handler)
	api.Post("/register/step2", handler.RegisterStep2Handler)
	api.Post("/register/step3", handler.RegisterStep3Handler)
	api.Get("/recover/status", handler.RecoverStatusHandler)
	api.Post("/recover/password", handler.RecoverPasswordHandler)
	api.Post("/sms/webhook", handler.SMSWebhookHandler)
	api.Get("/view/:view", handler.ViewHandler)
	api.Get("/image-nav/:id", handler.ImageNavigationHandler)
	api.Get("/image-full/:id", handler.ImageFullScreenHandler)
	api.Get("/show-filters", handler.ShowFiltersHandler)
	api.Get("/hide-filters", handler.HideFiltersHandler)
	api.Get("/toggle-search", handler.ToggleSearchHandler)
	api.Get("/search-location-modal", handler.SearchLocationModalHandler)
	api.Get("/search-location/save", handler.SearchLocationSaveHandler)
	api.Post("/password-field", handler.PasswordFieldHandler)
	api.Get("/modal-remove/:name", handler.ModalRemoveHandler)
	api.Get("/intro-banner/dismiss", handler.IntroBannerDismissHandler)
	api.Get("/search/", handler.SearchPageHandler)
	api.Get("/ad/:id/share", handler.AdShareHandler)
	api.Get("/ad/share/copy", handler.AdShareCopyHandler)
	api.Get("/ad/:id/new-conversation", handler.MessageModalHandler)

	api.Get("/category/:category/switch", handler.SwitchCategoryHandler)

	return app
}

func main() {
	deadlock.Opts.Disable = false                    // default = false
	deadlock.Opts.DeadlockTimeout = 10 * time.Second // detect very long waits
	deadlock.Opts.PrintAllCurrentGoroutines = true   // very helpful
	deadlock.Opts.LogBuf = os.Stderr

	if err := logger.Init(config.LogLevel, config.LogFormat,
		config.LogFile); err != nil {
		logger.Fatal("Failed to initialize logger", "error", err)
	}

	config.SecurityCheck()
	db.SetHashPepper(config.DBHashPepper)

	if err := kv.Init(); err != nil {
		logger.Fatal("Failed to initialize Redis", "error", err)
	}
	defer kv.Close()
	handler.InitRateLimiters(kv.Storage())

	if err := db.Init(config.DatabaseURL); err != nil {
		logger.Fatal("Failed to open database", "error", err)
	}
	defer db.Close()
	host, database := db.ConnectionTarget(config.DatabaseURL)
	logger.Info("Database connected", "host", host, "database", database)

	if err := db.CheckSchema(); err != nil {
		logger.Fatal("Database not ready", "error", err)
	}

	if err := ad.LoadCategories(); err != nil {
		logger.Fatal("Failed to initialize ads", "error", err)
	}

	if err := vector.InitEmbeddingCaches(); err != nil {
		logger.Fatal("Failed to initialize embedding caches", "error", err)
	}
	if err := vector.InitEmbedder(); err != nil {
		logger.Fatal("Failed to initialize embedder", "error", err)
	}
	vector.StartBackgroundProcessor()

	if err := sms.Init(); err != nil {
		logger.Fatal("Failed to initialize SMS service", "error", err)
	}
	if err := verify.Init(); err != nil {
		logger.Fatal("Failed to initialize Twilio Verify", "error", err)
	}
	if err := turnstile.Init(); err != nil {
		logger.Fatal("Failed to initialize Turnstile", "error", err)
	}
	sms.StartSMSWorker()
	handler.StartAdExpireWorker()

	imageStore, err := imagestore.NewDefault()
	if err != nil {
		logger.Fatal("Failed to initialize image store", "error", err)
	}
	handler.SetAdImageStore(imageStore)
	rockopinion.SetImageStore(imageStore)
	logger.Info("Image store configured",
		"bucket", config.MinIOBucketName, "url", config.MinIOAPIURL)

	app := setupApp()

	port := ":" + config.ServerPort
	logger.Info("Server starting", "port", port)

	if err := app.Listen(port); err != nil {
		logger.Fatal("Server failed to start", "error", err)
	}
}
