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

	app := fiber.New(fiber.Config{
		ErrorHandler: customErrorHandler,
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

	// Handle Chrome DevTools requests
	app.Get("/.well-known/appspecific/com.chrome.devtools.json", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	// Static file handler
	app.Static("/", "./static")

	// Main pages
	// Using an adaptor for now to get things working.
	app.Get("/", handlers.HandleHome)
	app.Get("/new-ad", handlers.AuthRequired, handlers.HandleNewAd)
	app.Get("/edit-ad/:id", handlers.AuthRequired, handlers.HandleEditAd)
	app.Get("/ad/:id", handlers.OptionalAuth, handlers.HandleViewAd)
	app.Get("/search", handlers.HandleSearch)
	app.Get("/search-page", handlers.HandleSearchPage)
	app.Get("/tree", handlers.TreeView)
	app.Get("/tree/*", handlers.TreeView)
	app.Get("/tree-collapsed/*", handlers.HandleTreeCollapse)

	// User registration/authentication
	app.Get("/register", handlers.HandleRegister)
	app.Post("/api/register", handlers.HandleRegisterSubmission)
	app.Get("/login", handlers.HandleLogin)
	app.Post("/api/login", handlers.HandleLoginSubmission)
	app.Post("/logout", handlers.HandleLogout)

	// User settings
	app.Get("/settings", handlers.AuthRequired, handlers.HandleSettings)
	app.Post("/api/change-password", handlers.AuthRequired, handlers.HandleChangePassword)
	app.Post("/api/delete-account", handlers.AuthRequired, handlers.HandleDeleteAccount)

	// Admin routes
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

	// Other Admin routes
	app.Post("/api/admin/users/set-admin", handlers.AdminRequired, handlers.HandleSetAdmin)
	app.Delete("/api/admin/users/kill/:id", handlers.AdminRequired, handlers.HandleKillUser)
	app.Post("/api/admin/users/resurrect/:id", handlers.AdminRequired, handlers.HandleResurrectUser)
	app.Delete("/api/admin/ads/kill/:id", handlers.AdminRequired, handlers.HandleKillAd)
	app.Post("/api/admin/ads/resurrect/:id", handlers.AdminRequired, handlers.HandleResurrectAd)
	app.Get("/api/admin/export/users", handlers.AdminRequired, handlers.HandleAdminExportUsers)
	app.Get("/api/admin/export/ads", handlers.AdminRequired, handlers.HandleAdminExportAds)
	app.Get("/api/admin/export/transactions", handlers.AdminRequired, handlers.HandleAdminExportTransactions)

	// HTMX view routes
	app.Get("/htmx/view/list", handlers.HandleListView)
	app.Get("/htmx/view/tree", handlers.HandleTreeViewContent)

	// API endpoints
	app.Get("/api/makes", handlers.HandleMakes)
	app.Get("/api/years", handlers.HandleYears)
	app.Get("/api/models", handlers.HandleModels)
	app.Get("/api/engines", handlers.HandleEngines)
	app.Post("/api/new-ad", handlers.AuthRequired, handlers.HandleNewAdSubmission)
	app.Post("/api/update-ad", handlers.AuthRequired, handlers.HandleUpdateAdSubmission)
	app.Delete("/delete-ad/:id", handlers.AuthRequired, handlers.HandleDeleteAd)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
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
