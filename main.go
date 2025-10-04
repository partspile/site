package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/handlers"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
)

func main() {
	// Initialize database
	if err := db.Init(config.DatabaseURL); err != nil {
		log.Fatalf("error initializing database: %v", err)
	}

	// Initialize B2 cache
	if err := b2util.Init(); err != nil {
		log.Fatalf("Failed to initialize B2 cache: %v", err)
	}

	// Initialize embedding caches
	if err := vector.InitEmbeddingCaches(); err != nil {
		log.Fatalf("Failed to initialize embedding caches: %v", err)
	}

	// Initialize Gemini client
	if err := vector.InitGeminiClient(); err != nil {
		log.Fatalf("Failed to initialize Gemini client: %v", err)
	}

	// Initialize Qdrant client
	if err := vector.InitQdrantClient(); err != nil {
		log.Fatalf("Failed to initialize Qdrant client: %v", err)
	}

	// Ensure collection exists and setup indexes
	if err := vector.EnsureCollectionExists(); err != nil {
		log.Fatalf("Failed to ensure collection exists: %v", err)
	}

	// Initialize vehicle cache
	if err := vehicle.InitVehicleCache(); err != nil {
		log.Fatalf("Failed to initialize vehicle cache: %v", err)
	}

	// Initialize parts static data
	if err := part.InitPartsData(); err != nil {
		log.Fatalf("Failed to initialize parts data: %v", err)
	}

	if err := vector.SetupPayloadIndexes(); err != nil {
		log.Fatalf("Failed to setup payload indexes: %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
		BodyLimit:    config.ServerUploadLimit,
		ReadTimeout:  30 * time.Second, // Prevent long-running requests
		WriteTimeout: 30 * time.Second, // Prevent long-running responses
	})

	// Add session middleware
	store := session.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_store", store)
		return c.Next()
	})

	// Add rate limiter
	app.Use(limiter.New(limiter.Config{
		Max:        config.ServerRateLimitMax,
		Expiration: config.ServerRateLimitExp,
	}))

	// Add logger middleware
	app.Use(logger.New())

	// Add global user stashing middleware - populates c.Locals("user") for all requests
	app.Use(handlers.StashUser)

	// Static files and utility
	app.Static("/", "./static")
	app.Get("/.well-known/appspecific/com.chrome.devtools.json", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	// Main pages
	app.Get("/", handlers.HandleHome)                  // x
	app.Get("/search", handlers.HandleSearch)          // x
	app.Get("/search-page", handlers.HandleSearchPage) // x

	// Tree view routes - split by browse vs search mode
	app.Get("/tree-browse-expand/*", handlers.HandleTreeExpandBrowse)     // x
	app.Get("/tree-browse-collapse/*", handlers.HandleTreeCollapseBrowse) // x
	app.Get("/tree-search-expand/*", handlers.HandleTreeExpandSearch)     // x
	app.Get("/tree-search-collapse/*", handlers.HandleTreeCollapseSearch) // x

	// Ad in-place expand/collapse partials for htmx
	app.Get("/ad/card/:id", handlers.HandleAdCard)     // x
	app.Get("/ad/detail/:id", handlers.HandleAdDetail) // x
	app.Get("/ad/edit-partial/:id", handlers.AuthRequired, handlers.HandleEditAdPartial)
	app.Get("/ad/image/:adID/:idx", handlers.HandleAdImage) // x

	// Ad management
	app.Get("/ad/:id", handlers.HandleAdPage)                       // x
	app.Get("/new-ad", handlers.AuthRequired, handlers.HandleNewAd) // x
	app.Get("/edit-ad/:id", handlers.AuthRequired, handlers.HandleEditAd)
	app.Delete("/delete-ad/:id", handlers.AuthRequired, handlers.HandleDeleteAd)

	// API group
	api := app.Group("/api")

	// Search API
	api.Get("/search", handlers.HandleSearchAPI)

	// Ad management (API)
	api.Post("/new-ad", handlers.AuthRequired, handlers.HandleNewAdSubmission)
	api.Post("/update-ad/:id", handlers.AuthRequired, handlers.HandleUpdateAdSubmission)
	api.Post("/bookmark-ad/:id", handlers.AuthRequired, handlers.HandleBookmarkAd)
	api.Delete("/bookmark-ad/:id", handlers.AuthRequired, handlers.HandleUnbookmarkAd)
	api.Get("/makes", handlers.HandleMakes)
	api.Get("/years", handlers.HandleYears)
	api.Get("/models", handlers.HandleModels)
	api.Get("/engines", handlers.HandleEngines)
	api.Get("/categories", handlers.HandleCategories)
	api.Get("/subcategories", handlers.HandleSubCategories)
	api.Get("/ad-image-url/:adID", handlers.HandleAdImageSignedURL)

	// Rock system (API)
	api.Get("/ad-rocks/:id", handlers.HandleAdRocks)
	api.Post("/throw-rock/:id", handlers.AuthRequired, handlers.HandleThrowRock)
	api.Get("/ad-rocks/:id/conversations", handlers.HandleViewRockConversations)
	api.Post("/resolve-rock/:id", handlers.AuthRequired, handlers.HandleResolveRock)

	// Admin dashboard and management
	admin := app.Group("/admin", handlers.AdminRequired)
	admin.Get("/", handlers.HandleAdminDashboard)
	admin.Get("/b2-cache", handlers.HandleAdminB2Cache)
	admin.Get("/embedding-cache", handlers.HandleAdminEmbeddingCache)
	admin.Get("/vehicle-cache", handlers.HandleAdminVehicleCache)

	// Admin API group
	adminAPI := api.Group("/admin", handlers.AdminRequired)
	adminAPI.Post("/b2-cache/clear", handlers.HandleClearB2Cache)
	adminAPI.Get("/b2-cache/refresh", handlers.HandleRefreshB2Cache)
	adminAPI.Post("/b2-cache/refresh", handlers.HandleRefreshB2Token)
	adminAPI.Get("/embedding-cache/refresh", handlers.HandleRefreshEmbeddingCache)
	adminAPI.Post("/embedding-cache/query/clear", handlers.HandleClearQueryEmbeddingCache)
	adminAPI.Post("/embedding-cache/user/clear", handlers.HandleClearUserEmbeddingCache)
	adminAPI.Post("/embedding-cache/site/clear", handlers.HandleClearSiteEmbeddingCache)
	adminAPI.Post("/vehicle-cache/clear", handlers.HandleClearVehicleCache)
	adminAPI.Get("/vehicle-cache/refresh", handlers.HandleRefreshVehicleCache)

	// User registration/authentication
	app.Get("/register", handlers.HandleRegistrationStep1)
	api.Post("/register/step1", handlers.HandleRegistrationStep1Submission)
	app.Get("/register/verify", handlers.HandleRegistrationVerification)
	api.Post("/register/verify", handlers.HandleRegistrationStep2Submission)
	api.Post("/sms/webhook", handlers.HandleSMSWebhook)
	app.Get("/rocks", handlers.HandleRocksPage)
	app.Get("/login", handlers.HandleLogin)
	api.Post("/login", handlers.HandleLoginSubmission)
	app.Post("/logout", handlers.HandleLogout)

	// Legal pages
	app.Get("/terms", handlers.HandleTermsOfService)
	app.Get("/privacy", handlers.HandlePrivacyPolicy)

	// Sitemap
	app.Get("/sitemap.xml", handlers.HandleSitemap)

	// User settings
	app.Get("/settings", handlers.AuthRequired, handlers.HandleSettings)       // x
	app.Get("/bookmarks", handlers.AuthRequired, handlers.HandleBookmarksPage) // x
	api.Post("/change-password", handlers.AuthRequired, handlers.HandleChangePassword)
	api.Post("/update-notification-method", handlers.AuthRequired, handlers.HandleUpdateNotificationMethod)
	api.Post("/notification-method-changed", handlers.AuthRequired, handlers.HandleNotificationMethodChanged)
	api.Post("/delete-account", handlers.AuthRequired, handlers.HandleDeleteAccount)
	app.Get("/user-menu", handlers.AuthRequired, handlers.HandleUserMenu) // x

	// Messaging system
	app.Get("/messages", handlers.AuthRequired, handlers.HandleMessagesPage)
	app.Get("/messages/:id/expand", handlers.AuthRequired, handlers.HandleExpandConversation)
	app.Get("/messages/:id/collapse", handlers.AuthRequired, handlers.HandleCollapseConversation)

	app.Get("/messages/sse", handlers.AuthRequired, handlers.HandleSSE)
	app.Get("/messages/:id/sse-update", handlers.AuthRequired, handlers.HandleSSEConversationUpdate)
	app.Post("/messages/:id/send", handlers.AuthRequired, handlers.HandleSendMessage)
	app.Get("/messages/start/:adID", handlers.AuthRequired, handlers.HandleStartConversation)
	api.Get("/messages/:action", handlers.AuthRequired, handlers.HandleMessagesAPI)

	// Views for HTMX view switching
	app.Post("/view/list", handlers.HandleListView) // x
	app.Post("/view/tree", handlers.HandleTreeView) // x
	app.Post("/view/grid", handlers.HandleGridView) // x
	app.Post("/view/map", handlers.HandleMapView)   // x

	// Start background user embedding processor
	vector.StartUserBackgroundProcessor()

	// Start background vector processor for ads
	vector.StartBackgroundProcessor()

	// Initially process existing ads without vectors
	vector.ProcessAdsWithoutVectors()

	fmt.Printf("Starting server on port %s...\n", config.ServerPort)
	log.Fatal(app.Listen(":" + config.ServerPort))
}

func customErrorHandler(ctx *fiber.Ctx, err error) error {
	// Status code defaults to 500
	code := fiber.StatusInternalServerError

	// Retrieve the custom status code if it's a *fiber.Error
	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
	}

	// Send custom error page
	ctx.Set(fiber.HeaderContentType, fiber.MIMETextHTML)
	return ui.ErrorPage(code, err.Error()).Render(ctx)
}
