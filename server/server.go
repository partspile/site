package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/sfeldma/parts-pile/site/handlers"
	"github.com/sfeldma/parts-pile/site/vehicle"
)

func Start() error {
	// Load vehicle data
	data, err := os.ReadFile("make-year-model.json")
	if err != nil {
		return fmt.Errorf("error reading vehicle data: %v", err)
	}

	if err := json.Unmarshal(data, &vehicle.Data); err != nil {
		return fmt.Errorf("error parsing vehicle data: %v", err)
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

	// API endpoints
	mux.HandleFunc("GET /api/makes", handlers.HandleMakes)
	mux.HandleFunc("GET /api/years/{make}", handlers.HandleYears)
	mux.HandleFunc("GET /api/models", handlers.HandleModels)
	mux.HandleFunc("GET /api/engines", handlers.HandleEngines)
	mux.HandleFunc("POST /api/new-ad", handlers.HandleNewAdSubmission)
	mux.HandleFunc("POST /api/update-ad", handlers.HandleUpdateAdSubmission)
	mux.HandleFunc("DELETE /delete-ad/{id}", handlers.HandleDeleteAd)

	fmt.Printf("Starting server on port %s...\n", port)
	return http.ListenAndServe(":"+port, mux)
}
