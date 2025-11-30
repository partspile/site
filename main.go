package main

import (
	"time"

	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	h "github.com/parts-pile/site/handlers"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/metrics"
	"github.com/parts-pile/site/minio"
	"github.com/parts-pile/site/sms"
	"github.com/parts-pile/site/vector"
)

func main() {
	// Initialize logging first
	if err := logger.Init(config.LogLevel, config.LogFormat,
		config.LogFile); err != nil {
		logger.Fatal("Failed to initialize logger", "error", err)
	}

	// Validate security configuration
	config.SecurityCheck()

	// Start metrics collector early to track startup stats
	metrics.StartCollector()

	// Initialize SMS service
	if err := sms.Init(); err != nil {
		logger.Fatal("Failed to initialize SMS service", "error", err)
	}

	// Initialize database
	if err := db.Init(config.DatabaseURL); err != nil {
		logger.Fatal("Failed to initialize database",
			"error", err, "databaseURL", config.DatabaseURL)
	}

	// Initialize ad category names cache
	if err := ad.InitAdCategories(); err != nil {
		logger.Fatal("Failed to initialize ad categories", "error", err)
	}

	// Initialize MinIO cache
	if err := minio.Init(); err != nil {
		logger.Fatal("Failed to initialize MinIO cache", "error", err)
	}

	// Initialize embedding caches
	if err := vector.InitEmbeddingCaches(); err != nil {
		logger.Fatal("Failed to initialize embedding caches", "error", err)
	}

	// Initialize Gemini client
	if err := vector.InitGeminiClient(); err != nil {
		logger.Fatal("Failed to initialize Gemini client", "error", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: h.CustomErrorHandler,
		BodyLimit:    config.ServerBodyLimit, // Total request body size (for multiple files)
		ReadTimeout:  30 * time.Second,       // Prevent long-running requests
		WriteTimeout: 30 * time.Second,       // Prevent long-running responses
	})

	// Add security headers using Helmet (must be early in middleware chain)
	app.Use(h.ConfigureHelmet())

	// Add global rate limiter
	app.Use(h.GlobalRateLimiter)

	// Add JWT middleware
	app.Use(h.JWTMiddleware)

	// Add CSRF protection using Fiber's built-in middleware (cookie-based)
	app.Use(h.CSRFMiddleware)

	// Add logger middleware - configured for HTTP request logging
	// Use our custom logger for Fiber's HTTP request logs
	// Custom format: status | latency | ip | method | path
	app.Use(fiberlogger.New(fiberlogger.Config{
		Output: logger.Writer(),
		Format: "${status} | ${latency} | ${ip} | ${method} | ${path}\n",
	}))

	// Static files and utility
	app.Static("/", "./static")
	app.Get("/.well-known/appspecific/com.chrome.devtools.json", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	// Image proxy route - serves images from MinIO
	app.Get("/image/:adID/:idx/:size", h.HandleImageProxy)

	// Main search page
	app.Get("/", h.HandleHome)                      // x
	app.Get("/search", h.HandleSearch)              // x
	app.Get("/search-page", h.HandleSearchPage)     // x
	app.Get("/search-widget", h.HandleSearchWidget) // x
	app.Get("/switch-ad-category/:adCategory", h.HandleSwitchAdCategory)

	// Tree view routes - split by browse vs search mode
	app.Get("/tree-browse-expand/*", h.HandleTreeExpandBrowse)     // x
	app.Get("/tree-browse-collapse/*", h.HandleTreeCollapseBrowse) // x
	app.Get("/tree-search-expand/*", h.HandleTreeExpandSearch)     // x
	app.Get("/tree-search-collapse/*", h.HandleTreeCollapseSearch) // x

	// Ad in-place expand/collapse partials for htmx
	app.Get("/ad/collapse/:id", h.HandleAdCollapse)           // x
	app.Get("/ad/detail/:id", h.HandleAdDetail)               // x
	app.Get("/ad/image/:adID/:idx", h.HandleAdImage)          // x
	app.Get("/ad/grid-image/:adID/:idx", h.HandleAdGridImage) // x

	// Modal routes for HTMX-based modals
	app.Get("/modal/ad/price/:id", h.AuthRequired, h.HandlePriceModal)
	app.Get("/modal/ad/location/:id", h.AuthRequired, h.HandleLocationModal)
	app.Get("/modal/ad/description/:id", h.AuthRequired, h.HandleDescriptionModal)
	app.Get("/modal/ad/message/:id", h.AuthRequired, h.HandleMessageModal)
	app.Get("/modal/ad/share/:id", h.HandleShareModal)
	app.Get("/modal/category-select", h.HandleAdCategoryModal)

	// Ad management
	app.Get("/ad/:id", h.HandleAdPage)                // x
	app.Get("/new-ad", h.AuthRequired, h.HandleNewAd) // x
	app.Delete("/delete-ad/:id", h.AuthRequired, h.HandleDeleteAd)
	app.Post("/restore-ad/:id", h.AuthRequired, h.HandleRestoreAd)

	// API group
	api := app.Group("/api")

	// Ad management (API)
	api.Post("/new-ad", h.AuthRequired, h.HandleNewAdSubmission) // x
	api.Post("/update-ad-price/:id", h.AuthRequired, h.HandleUpdateAdPrice)
	api.Post("/update-ad-location/:id", h.AuthRequired, h.HandleUpdateAdLocation)
	api.Post("/update-ad-description/:id", h.AuthRequired, h.HandleUpdateAdDescription)
	api.Post("/bookmark-ad/:id", h.AuthRequired, h.HandleBookmarkAd)
	api.Delete("/bookmark-ad/:id", h.AuthRequired, h.HandleUnbookmarkAd)
	api.Get("/field/:fieldName", h.HandleFieldValues)

	// Rock system (API)
	api.Get("/ad-rocks/:id", h.HandleAdRocks)
	api.Post("/throw-rock/:id", h.AuthRequired, h.HandleThrowRock)
	api.Get("/ad-rocks/:id/conversations", h.HandleViewRockConversations)
	api.Post("/resolve-rock/:id", h.AuthRequired, h.HandleResolveRock)

	// Admin dashboard and management
	admin := app.Group("/admin", h.AdminRequired)
	admin.Get("/", h.HandleAdminDashboard)
	admin.Get("/embedding-cache", h.HandleAdminEmbeddingCache)
	admin.Get("/metrics", h.HandleMonitor)
	admin.Get("/clicks", h.HandleAdminClicks)

	// Admin API group
	adminAPI := api.Group("/admin", h.AdminRequired)
	adminAPI.Get("/embedding-cache/refresh", h.HandleRefreshEmbeddingCache)
	adminAPI.Post("/embedding-cache/query/clear", h.HandleClearQueryEmbeddingCache)
	adminAPI.Post("/embedding-cache/user/clear", h.HandleClearUserEmbeddingCache)
	adminAPI.Post("/embedding-cache/site/clear", h.HandleClearSiteEmbeddingCache)
	adminAPI.Get("/metrics/refresh", h.HandleMonitorRefresh)
	adminAPI.Get("/clicks/refresh", h.HandleAdminClicksRefresh)

	// User registration/authentication
	app.Get("/register", h.HandleRegistration)
	api.Post("/register/step1", h.RegistrationRateLimiter, h.HandleRegistrationStep1)
	api.Post("/register/step2", h.HandleRegistrationStep2)
	api.Post("/sms/webhook", h.HandleSMSWebhook)
	app.Get("/rocks", h.HandleRocksPage)
	app.Get("/login", h.HandleLogin)
	api.Post("/login", h.HandleLoginSubmission)
	app.Post("/logout", h.HandleLogout)

	// Legal pages
	app.Get("/terms", h.HandleTermsOfService)
	app.Get("/privacy", h.HandlePrivacyPolicy)
	app.Get("/about", h.HandleAbout)

	// Sitemap
	app.Get("/sitemap.xml", h.HandleSitemap)

	// Health check
	app.Get("/health", h.HandleHealth)

	// User settings
	app.Get("/settings", h.AuthRequired, h.HandleSettings)                // x
	app.Get("/ads", h.AuthRequired, h.HandleAdsPage)                      // x
	app.Get("/ads/bookmarked", h.AuthRequired, h.HandleBookmarkedAdsPage) // x
	app.Get("/ads/active", h.AuthRequired, h.HandleActiveAdsPage)         // x
	app.Get("/ads/deleted", h.AuthRequired, h.HandleDeletedAdsPage)       // x
	api.Post("/change-password", h.AuthRequired, h.HandleChangePassword)
	api.Post("/update-notification-method", h.AuthRequired, h.HandleUpdateNotificationMethod)
	api.Post("/notification-method-changed", h.AuthRequired, h.HandleNotificationMethodChanged)
	api.Post("/delete-account", h.AuthRequired, h.HandleDeleteAccount)
	app.Get("/user-menu", h.AuthRequired, h.HandleUserMenu) // x

	// Messaging system
	app.Get("/messages", h.AuthRequired, h.HandleMessagesPage)
	app.Get("/messages/:id/expand", h.AuthRequired, h.HandleExpandConversation)
	app.Get("/messages/:id/collapse", h.AuthRequired, h.HandleCollapseConversation)

	app.Get("/messages/sse", h.AuthRequired, h.HandleSSE)
	app.Get("/messages/:id/sse-update", h.AuthRequired, h.HandleSSEConversationUpdate)
	app.Post("/messages/:id/send", h.AuthRequired, h.HandleSendMessage)
	app.Post("/messages/:id/ping", h.AuthRequired, h.HandlePing)
	app.Get("/messages/start/:adID", h.AuthRequired, h.HandleStartConversation)
	api.Get("/messages/:action", h.AuthRequired, h.HandleMessagesAPI)

	// Views for HTMX view switching
	app.Post("/view/list", h.HandleListView) // x
	app.Post("/view/tree", h.HandleTreeView) // x
	app.Post("/view/grid", h.HandleGridView) // x

	// Start background vector processor for ads
	vector.StartBackgroundProcessor()

	// Initially process existing ads without vectors
	vector.ProcessAdsWithoutVectors()

	logger.Info("Starting server", "port", config.ServerPort)
	if err := app.Listen(":" + config.ServerPort); err != nil {
		logger.Fatal("Server failed to start", "error", err, "port",
			config.ServerPort)
	}
}
