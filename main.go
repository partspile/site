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
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/vector"
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

	// Initialize embedding cache
	if err := vector.InitEmbeddingCache(); err != nil {
		log.Fatalf("Failed to initialize embedding cache: %v", err)
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

	if err := vector.SetupPayloadIndexes(); err != nil {
		log.Fatalf("Failed to setup payload indexes: %v", err)
	}

	// Test that payload indexes are working
	if err := vector.TestPayloadIndexes(); err != nil {
		log.Printf("Warning: Payload index test failed: %v", err)
	} else {
		log.Printf("Payload indexes verified and working correctly")
	}

	// Start background user embedding processor
	vector.GetEmbeddingQueue().StartBackgroundProcessor()

	// Start background vector processor for ads
	vector.GetVectorProcessor().StartBackgroundProcessor()

	// Initially populate the queue with existing ads without vectors
	vector.GetVectorProcessor().QueueAdsWithoutVectors()

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

	// Static files and utility
	app.Static("/", "./static")
	app.Get("/.well-known/appspecific/com.chrome.devtools.json", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	// Main pages
	app.Get("/", handlers.HandleHome)
	app.Get("/search", handlers.HandleSearch)
	app.Get("/search-page", handlers.HandleSearchPage)
	app.Get("/api/search", handlers.HandleSearchAPI)
	app.Get("/tree", handlers.TreeView)
	app.Get("/tree/*", handlers.TreeView)
	app.Get("/tree-collapsed/*", handlers.HandleTreeCollapse)

	// Ad in-place expand/collapse partials for htmx
	app.Get("/ad/card/:id", handlers.HandleAdCardPartial)
	app.Get("/ad/detail/:id", handlers.HandleAdDetailPartial)
	app.Get("/ad/edit-partial/:id", handlers.AuthRequired, handlers.HandleEditAdPartial)
	app.Get("/ad/image/:adID/:idx", handlers.HandleAdImagePartial)
	app.Get("/ad/expand-tree/:id", handlers.HandleExpandAdTree)
	app.Get("/ad/collapse-tree/:id", handlers.HandleCollapseAdTree)

	// Ad management
	app.Get("/ad/:id", handlers.OptionalAuth, handlers.HandleViewAd)
	app.Get("/new-ad", handlers.AuthRequired, handlers.HandleNewAd)
	app.Get("/edit-ad/:id", handlers.AuthRequired, handlers.HandleEditAd)
	app.Delete("/delete-ad/:id", handlers.AuthRequired, handlers.HandleDeleteAd)

	// API group
	api := app.Group("/api")

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

	// Admin dashboard and management
	admin := app.Group("/admin", handlers.AdminRequired)
	admin.Get("/", handlers.HandleAdminDashboard)
	admin.Get("/users", handlers.HandleAdminUsers)
	admin.Get("/ads", handlers.HandleAdminAds)
	admin.Get("/transactions", handlers.HandleAdminTransactions)
	admin.Get("/makes", handlers.HandleAdminMakes)
	admin.Get("/models", handlers.HandleAdminModels)
	admin.Get("/years", handlers.HandleAdminYears)
	admin.Get("/part-categories", handlers.HandleAdminPartCategories)
	admin.Get("/part-sub-categories", handlers.HandleAdminPartSubCategories)
	admin.Get("/parent-companies", handlers.HandleAdminParentCompanies)
	admin.Get("/make-parent-companies", handlers.HandleAdminMakeParentCompanies)
	admin.Get("/b2-cache", handlers.HandleAdminB2Cache)
	admin.Get("/embedding-cache", handlers.HandleAdminEmbeddingCache)

	// Admin API group
	adminAPI := api.Group("/admin", handlers.AdminRequired)
	adminAPI.Post("/users/set-admin", handlers.HandleSetAdmin)
	adminAPI.Delete("/users/archive/:id", handlers.HandleArchiveUser)
	adminAPI.Post("/users/restore/:id", handlers.HandleRestoreUser)
	adminAPI.Delete("/ads/archive/:id", handlers.HandleArchiveAd)
	adminAPI.Post("/ads/restore/:id", handlers.HandleRestoreAd)
	adminAPI.Get("/export/users", handlers.HandleAdminExportUsers)
	adminAPI.Get("/export/ads", handlers.HandleAdminExportAds)
	adminAPI.Get("/export/transactions", handlers.HandleAdminExportTransactions)
	adminAPI.Post("/b2-cache/clear", handlers.HandleClearB2Cache)
	adminAPI.Post("/b2-cache/refresh", handlers.HandleRefreshB2Token)
	adminAPI.Post("/embedding-cache/clear", handlers.HandleClearEmbeddingCache)

	// User registration/authentication
	app.Get("/register", handlers.HandleRegistrationStep1)
	api.Post("/register/step1", handlers.HandleRegistrationStep1Submission)
	app.Get("/register/verify", handlers.HandleRegistrationVerification)
	api.Post("/register/verify", handlers.HandleRegistrationStep2Submission)
	api.Post("/sms/webhook", handlers.HandleSMSWebhook)
	app.Get("/login", handlers.HandleLogin)
	api.Post("/login", handlers.HandleLoginSubmission)
	app.Post("/logout", handlers.HandleLogout)

	// Legal pages
	app.Get("/terms", handlers.HandleTermsOfService)
	app.Get("/privacy", handlers.HandlePrivacyPolicy)

	// User settings
	app.Get("/settings", handlers.AuthRequired, handlers.HandleSettings)
	api.Post("/change-password", handlers.AuthRequired, handlers.HandleChangePassword)
	api.Post("/update-notification-method", handlers.AuthRequired, handlers.HandleUpdateNotificationMethod)
	api.Post("/notification-method-changed", handlers.AuthRequired, handlers.HandleNotificationMethodChanged)
	api.Post("/delete-account", handlers.AuthRequired, handlers.HandleDeleteAccount)
	app.Get("/settings/bookmarked-ads", handlers.AuthRequired, handlers.HandleBookmarkedAds)

	// Messaging system
	app.Get("/messages", handlers.AuthRequired, handlers.HandleMessagesPage)
	app.Get("/messages/:id/expand", handlers.AuthRequired, handlers.HandleExpandConversation)
	app.Get("/messages/:id/collapse", handlers.AuthRequired, handlers.HandleCollapseConversation)
	app.Get("/messages/sse", handlers.AuthRequired, handlers.HandleSSE)
	app.Post("/messages/:id/send", handlers.AuthRequired, handlers.HandleSendMessage)
	app.Get("/messages/start/:adID", handlers.AuthRequired, handlers.HandleStartConversation)
	api.Get("/messages/:action", handlers.AuthRequired, handlers.HandleMessagesAPI)

	// Views for HTMX/direct navigation
	app.Get("/view/list", handlers.HandleListView)
	app.Get("/view/tree", handlers.HandleTreeViewContent)
	app.Get("/view/grid", handlers.HandleGridView)
	app.Get("/view/map", handlers.HandleMapView)
	app.Post("/view/list", handlers.HandleListView)
	app.Post("/view/tree", handlers.HandleTreeViewContent)
	app.Post("/view/grid", handlers.HandleGridView)
	app.Post("/view/map", handlers.HandleMapView)

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
