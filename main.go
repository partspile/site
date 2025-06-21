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
	"github.com/parts-pile/site/components"
	"github.com/parts-pile/site/handlers"
	"github.com/parts-pile/site/part"
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
		Max:        20,
		Expiration: 30 * time.Second,
	}))

	// Add logger middleware
	app.Use(logger.New())

	// Static file handler
	app.Static("/static", "./static")

	// Main pages
	// Using an adaptor for now to get things working.
	app.Get("/", handlers.HandleHome)
	app.Get("/new-ad", handlers.AuthRequired, handlers.HandleNewAd)
	app.Get("/edit-ad/:id", handlers.AuthRequired, handlers.HandleEditAd)
	app.Get("/ad/:id", handlers.HandleViewAd)
	app.Get("/search", handlers.HandleSearch)
	app.Get("/search-page", handlers.HandleSearchPage)

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
	admin.Get("/export", handlers.HandleAdminExport)

	// Other Admin routes
	app.Post("/api/admin/users/set-admin", handlers.AdminRequired, handlers.HandleSetAdmin)

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
	// This component needs to be created
	return components.ErrorPage(code, err.Error()).Render(ctx)
}
