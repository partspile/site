package main

import (
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/b2util"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	h "github.com/parts-pile/site/handlers"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/vector"
	"github.com/parts-pile/site/vehicle"
)

func main() {
	// Initialize database
	if err := db.Init(config.DatabaseURL); err != nil {
		log.Fatalf("error initializing database: %v", err)
	}

	// Initialize ad category names cache
	ad.SetAdCategoryNames()

	// Initialize vehicle cache
	if err := vehicle.InitVehicleCache(); err != nil {
		log.Fatalf("Failed to initialize vehicle cache: %v", err)
	}

	// Initialize part cache
	if err := part.InitPartCache(); err != nil {
		log.Fatalf("Failed to initialize part cache: %v", err)
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
	if err := vector.InitQdrantCollection(); err != nil {
		log.Fatalf("Failed to ensure collection exists: %v", err)
	}

	if err := vector.InitQdrantIndexes(); err != nil {
		log.Fatalf("Failed to setup payload indexes: %v", err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: h.CustomErrorHandler,
		BodyLimit:    config.ServerUploadLimit,
		ReadTimeout:  30 * time.Second, // Prevent long-running requests
		WriteTimeout: 30 * time.Second, // Prevent long-running responses
	})

	// Add rate limiter
	app.Use(limiter.New(limiter.Config{
		Max:        config.ServerRateLimitMax,
		Expiration: config.ServerRateLimitExp,
	}))

	// Add JWT middleware
	app.Use(h.JWTMiddleware)

	// Add logger middleware
	app.Use(logger.New())

	// Static files and utility
	app.Static("/", "./static")
	app.Get("/.well-known/appspecific/com.chrome.devtools.json", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

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
	app.Get("/ad/:id", h.HandleAdPage)                                // x
	app.Get("/new-ad", h.AuthRequired, h.HandleNewAd)                 // x
	app.Get("/duplicate-ad/:id", h.AuthRequired, h.HandleDuplicateAd) // x
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
	api.Get("/filter-makes", h.HandleFilterMakes)
	api.Get("/years", h.HandleYears)
	api.Get("/models", h.HandleModels)
	api.Get("/engines", h.HandleEngines)
	api.Get("/subcategories", h.HandleSubCategories)
	api.Get("/ad-image-url/:adID", h.HandleAdImageSignedURL)

	// Rock system (API)
	api.Get("/ad-rocks/:id", h.HandleAdRocks)
	api.Post("/throw-rock/:id", h.AuthRequired, h.HandleThrowRock)
	api.Get("/ad-rocks/:id/conversations", h.HandleViewRockConversations)
	api.Post("/resolve-rock/:id", h.AuthRequired, h.HandleResolveRock)

	// Admin dashboard and management
	admin := app.Group("/admin", h.AdminRequired)
	admin.Get("/", h.HandleAdminDashboard)
	admin.Get("/b2-cache", h.HandleAdminB2Cache)
	admin.Get("/embedding-cache", h.HandleAdminEmbeddingCache)
	admin.Get("/vehicle-cache", h.HandleAdminVehicleCache)
	admin.Get("/part-cache", h.HandleAdminPartCache)

	// Admin API group
	adminAPI := api.Group("/admin", h.AdminRequired)
	adminAPI.Post("/b2-cache/clear", h.HandleClearB2Cache)
	adminAPI.Get("/b2-cache/refresh", h.HandleRefreshB2Cache)
	adminAPI.Get("/embedding-cache/refresh", h.HandleRefreshEmbeddingCache)
	adminAPI.Post("/embedding-cache/query/clear", h.HandleClearQueryEmbeddingCache)
	adminAPI.Post("/embedding-cache/user/clear", h.HandleClearUserEmbeddingCache)
	adminAPI.Post("/embedding-cache/site/clear", h.HandleClearSiteEmbeddingCache)
	adminAPI.Post("/vehicle-cache/clear", h.HandleClearVehicleCache)
	adminAPI.Get("/vehicle-cache/refresh", h.HandleRefreshVehicleCache)
	adminAPI.Post("/part-cache/clear", h.HandleClearPartCache)
	adminAPI.Get("/part-cache/refresh", h.HandleRefreshPartCache)

	// User registration/authentication
	app.Get("/register", h.HandleRegistrationStep1)
	api.Post("/register/step1", h.HandleRegistrationStep1Submission)
	app.Get("/register/verify", h.HandleRegistrationVerification)
	api.Post("/register/verify", h.HandleRegistrationStep2Submission)
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
	app.Get("/messages/start/:adID", h.AuthRequired, h.HandleStartConversation)
	api.Get("/messages/:action", h.AuthRequired, h.HandleMessagesAPI)

	// Views for HTMX view switching
	app.Post("/view/list", h.HandleListView) // x
	app.Post("/view/tree", h.HandleTreeView) // x
	app.Post("/view/grid", h.HandleGridView) // x

	// Start background user embedding processor
	vector.StartUserBackgroundProcessor()

	// Start background vector processor for ads
	vector.StartBackgroundProcessor()

	// Initially process existing ads without vectors
	vector.ProcessAdsWithoutVectors()

	fmt.Printf("Starting server on port %s...\n", config.ServerPort)
	log.Fatal(app.Listen(":" + config.ServerPort))
}
