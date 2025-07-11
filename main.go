package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/handlers"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/search"
	"github.com/parts-pile/site/ui"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vehicle"
)

func main() {
	// Initialize ads/project database
	if err := ad.InitDB("project.db"); err != nil {
		log.Fatalf("error initializing project database: %v", err)
	}

	// Initialize vehicle package with the same DB
	// (ensures vehicle uses project.db)
	vehicle.InitDB(ad.DB)

	// Initialize part package with the same DB
	part.InitDB(ad.DB)

	// Initialize user package with the same DB
	user.InitDB(ad.DB)

	// Initialize search package with the same DB
	search.InitDB(ad.DB)

	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
		BodyLimit:    20 * 1024 * 1024, // 20 MB
	})

	// Add session middleware
	store := session.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("session_store", store)
		return c.Next()
	})

	// Add rate limiter
	app.Use(limiter.New(limiter.Config{
		Max:        60,
		Expiration: 1 * time.Minute,
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
	app.Get("/tree", handlers.TreeView)
	app.Get("/tree/*", handlers.TreeView)
	app.Get("/tree-collapsed/*", handlers.HandleTreeCollapse)

	// Ad in-place expand/collapse partials for htmx
	app.Get("/ad/card/:id", handlers.HandleAdCardPartial)
	app.Get("/ad/detail/:id", handlers.HandleAdDetailPartial)
	app.Get("/ad/edit-partial/:id", handlers.AuthRequired, handlers.HandleEditAdPartial)
	app.Get("/ad/image/:adID/:idx", handlers.HandleAdImagePartial)

	// Ad management
	app.Get("/ad/:id", handlers.OptionalAuth, handlers.HandleViewAd)
	app.Get("/new-ad", handlers.AuthRequired, handlers.HandleNewAd)
	app.Get("/edit-ad/:id", handlers.AuthRequired, handlers.HandleEditAd)

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

	// User registration/authentication
	app.Get("/register", handlers.HandleRegister)
	api.Post("/register", handlers.HandleRegisterSubmission)
	app.Get("/login", handlers.HandleLogin)
	api.Post("/login", handlers.HandleLoginSubmission)
	app.Post("/logout", handlers.HandleLogout)

	// User settings
	app.Get("/settings", handlers.AuthRequired, handlers.HandleSettings)
	api.Post("/change-password", handlers.AuthRequired, handlers.HandleChangePassword)
	api.Post("/delete-account", handlers.AuthRequired, handlers.HandleDeleteAccount)
	app.Get("/settings/bookmarked-ads", handlers.AuthRequired, handlers.HandleBookmarkedAds)

	// Views for HTMX/direct navigation
	app.Get("/view/list", handlers.HandleListView)
	app.Get("/view/tree", handlers.HandleTreeViewContent)
	app.Get("/view/grid", handlers.HandleGridView)
	app.Get("/view/map", handlers.HandleMapView)
	app.Post("/view/list", handlers.HandleListView)
	app.Post("/view/tree", handlers.HandleTreeViewContent)
	app.Post("/view/grid", handlers.HandleGridView)
	app.Post("/view/map", handlers.HandleMapView)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	fmt.Printf("Starting server on port %s...\n", port)
	log.Fatal(app.Listen(":" + port))
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
