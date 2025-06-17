package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/handlers"
	"github.com/parts-pile/site/part"
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
