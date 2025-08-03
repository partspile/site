package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/vector"
)

func main() {
	// Initialize database
	if err := db.Init(config.DatabaseURL); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Get a few ads to test
	ads, err := ad.GetAllAds()
	if err != nil {
		log.Fatalf("Failed to get ads: %v", err)
	}

	if len(ads) == 0 {
		log.Fatalf("No ads found")
	}

	// Test the metadata function for the first few ads
	for i := 0; i < 3 && i < len(ads); i++ {
		adObj := ads[i]
		fmt.Printf("\n=== Ad %d: %s ===\n", adObj.ID, adObj.Title)

		// Test location lookup
		if adObj.LocationID > 0 {
			city, adminArea, country, _, lat, lon, err := ad.GetLocationWithCoords(adObj.LocationID)
			if err != nil {
				fmt.Printf("Location lookup error: %v\n", err)
			} else {
				fmt.Printf("Location: %s, %s, %s\n", city, adminArea, country)
				if lat != nil && lon != nil {
					fmt.Printf("Coordinates: %.6f, %.6f\n", *lat, *lon)
				} else {
					fmt.Printf("No coordinates found\n")
				}
			}
		}

		// Test metadata generation (without building actual embedding)
		metadata := vector.BuildAdEmbeddingMetadata(adObj)
		metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
		fmt.Printf("Metadata:\n%s\n", metadataJSON)
	}
}
