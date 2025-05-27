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
	mux.HandleFunc("/", handlers.HandleHome)
	mux.HandleFunc("/new-ad", handlers.HandleNewAd)
	mux.HandleFunc("/edit-ad/", handlers.HandleEditAd)
	mux.HandleFunc("/api/makes", handlers.HandleMakes)
	mux.HandleFunc("/api/years", handlers.HandleYears)
	mux.HandleFunc("/api/models", handlers.HandleModels)
	mux.HandleFunc("/api/engines", handlers.HandleEngines)
	mux.HandleFunc("/api/new-ad", handlers.HandleNewAdSubmission)
	mux.HandleFunc("/api/update-ad", handlers.HandleUpdateAdSubmission)
	mux.HandleFunc("/ad/", handlers.HandleViewAd)

	fmt.Printf("Starting server on port %s...\n", port)
	return http.ListenAndServe(":"+port, mux)
}
