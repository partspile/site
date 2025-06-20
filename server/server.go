package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/handlers"
	"github.com/parts-pile/site/part"
	"github.com/parts-pile/site/user"
	"github.com/parts-pile/site/vehicle"
)

func Start() error {
	// Initialize ads/project database
	if err := ad.InitDB("project.db"); err != nil {
		return fmt.Errorf("error initializing project database: %v", err)
	}

	// Initialize vehicle package with the same DB
	// (ensures vehicle uses project.db)
	vehicle.InitDB(ad.DB)

	// Initialize part package with the same DB
	part.InitDB(ad.DB)

	// Initialize user package with the same DB
	user.InitDB(ad.DB)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Static file handler
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Main pages
	mux.HandleFunc("GET /{$}", handlers.HandleHome)
	mux.HandleFunc("GET /new-ad", handlers.HandleNewAd)
	mux.HandleFunc("GET /edit-ad/{id}", handlers.HandleEditAd)
	mux.HandleFunc("GET /ad/{id}", handlers.HandleViewAd)
	mux.HandleFunc("GET /search", handlers.HandleSearch)

	// User registration/authentication
	mux.HandleFunc("GET /register", handlers.HandleRegister)
	mux.HandleFunc("POST /api/register", handlers.HandleRegisterSubmission)
	mux.HandleFunc("GET /login", handlers.HandleLogin)
	mux.HandleFunc("POST /api/login", handlers.HandleLoginSubmission)
	mux.HandleFunc("POST /logout", handlers.HandleLogout)

	// User settings
	mux.HandleFunc("GET /settings", handlers.HandleSettings)
	mux.HandleFunc("POST /api/change-password", handlers.HandleChangePassword)
	mux.HandleFunc("POST /api/delete-account", handlers.HandleDeleteAccount)

	// Admin routes
	mux.HandleFunc("GET /admin", handlers.AdminRequired(handlers.HandleAdminDashboard))
	mux.HandleFunc("GET /admin/users", handlers.AdminRequired(handlers.HandleAdminUsers))
	mux.HandleFunc("POST /api/admin/users/set-admin", handlers.AdminRequired(handlers.HandleSetAdmin))
	mux.HandleFunc("GET /admin/ads", handlers.AdminRequired(handlers.HandleAdminAds))
	mux.HandleFunc("GET /admin/transactions", handlers.AdminRequired(handlers.HandleAdminTransactions))
	mux.HandleFunc("GET /admin/export", handlers.AdminRequired(handlers.HandleAdminExport))

	// API endpoints
	mux.HandleFunc("GET /api/makes", handlers.HandleMakes)
	mux.HandleFunc("GET /api/years", handlers.HandleYears)
	mux.HandleFunc("GET /api/models", handlers.HandleModels)
	mux.HandleFunc("GET /api/engines", handlers.HandleEngines)
	mux.HandleFunc("POST /api/new-ad", handlers.HandleNewAdSubmission)
	mux.HandleFunc("POST /api/update-ad", handlers.HandleUpdateAdSubmission)
	mux.HandleFunc("DELETE /delete-ad/{id}", handlers.HandleDeleteAd)

	fmt.Printf("Starting server on port %s...\n", port)
	return http.ListenAndServe(":"+port, mux)
}
