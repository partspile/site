package main

import (
	"log"
	"time"

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

	// Initialize vector clients
	if err := vector.InitGeminiClient(); err != nil {
		log.Fatalf("Failed to initialize Gemini client: %v", err)
	}

	if err := vector.InitQdrantClient(); err != nil {
		log.Fatalf("Failed to initialize Qdrant client: %v", err)
	}

	// Get all ads
	ads, err := ad.GetAllAds()
	if err != nil {
		log.Fatalf("Failed to get ads: %v", err)
	}

	log.Printf("Rebuilding vectors for %d ads", len(ads))

	// Process each ad
	for i, adObj := range ads {
		log.Printf("Processing ad %d/%d: %s (ID: %d)", i+1, len(ads), adObj.Title, adObj.ID)

		err := vector.BuildAdEmbedding(adObj)
		if err != nil {
			log.Printf("Failed to rebuild embedding for ad %d: %v", adObj.ID, err)
		} else {
			log.Printf("Successfully rebuilt embedding for ad %d", adObj.ID)
		}

		// Sleep to avoid rate limits
		time.Sleep(config.QdrantProcessingSleepInterval)
	}

	log.Printf("Vector rebuild complete!")
}
