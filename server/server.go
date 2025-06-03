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
	// Load initial data
	if err := vehicle.LoadData(); err != nil {
		return fmt.Errorf("error loading vehicle data: %v", err)
	}

	if err := part.LoadData(); err != nil {
		return fmt.Errorf("error loading part data: %v", err)
	}

	// Load ads data
	if err := ad.InitDB("ads.db"); err != nil {
		return fmt.Errorf("error initializing ads database: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// Main pages
	mux.HandleFunc("GET /", handlers.HandleHome)
	mux.HandleFunc("GET /new-ad", handlers.HandleNewAd)
	mux.HandleFunc("GET /edit-ad/{id}", handlers.HandleEditAd)
	mux.HandleFunc("GET /ad/{id}", handlers.HandleViewAd)
	mux.HandleFunc("GET /search", handlers.HandleSearchPage)
	mux.HandleFunc("GET /ads", handlers.HandleAdsPage)

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
