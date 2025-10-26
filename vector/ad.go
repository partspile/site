package vector

import (
	"fmt"
	"log"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/rock"
	"github.com/parts-pile/site/vehicle"
)

// Static queue for ad processing
var adQueue = make(chan ad.Ad, config.QdrantProcessingQueueSize)

// BuildAdEmbedding builds and stores an embedding for a single ad
func BuildAdEmbedding(adObj ad.AdDetail) error {
	return BuildAdEmbeddings([]ad.AdDetail{adObj})
}

// BuildAdEmbeddings builds and stores embeddings for multiple ads in batch
func BuildAdEmbeddings(ads []ad.AdDetail) error {
	if len(ads) == 0 {
		return nil
	}

	log.Printf("[BuildAdEmbeddings] Building embeddings for %d ads in batch", len(ads))

	// Build prompts for all ads
	var prompts []string
	for _, adObj := range ads {
		prompt := buildAdEmbeddingPrompt(adObj)
		prompts = append(prompts, prompt)
	}

	// Generate embeddings in batch
	log.Printf("[BuildAdEmbeddings] Generating batch embeddings for %d ads", len(ads))
	embeddings, err := EmbedTexts(prompts)
	if err != nil {
		log.Printf("[BuildAdEmbeddings] Failed to generate batch embeddings: %v", err)
		return err
	}

	// Build metadatas and ad IDs
	var adIDs []int
	var metadatas []map[string]interface{}

	for _, adObj := range ads {
		meta := buildAdEmbeddingMetadata(adObj)
		adIDs = append(adIDs, adObj.ID)
		metadatas = append(metadatas, meta)
	}

	// Batch upsert to Qdrant
	err = UpsertAdEmbeddings(adIDs, embeddings, metadatas)
	if err != nil {
		log.Printf("[BuildAdEmbeddings] Failed to batch upsert vectors: %v", err)
	} else {
		log.Printf("[BuildAdEmbeddings] Successfully batch upserted %d vectors", len(adIDs))
	}

	// Mark ads as having vectors in database
	err = ad.MarkAdsAsHavingVector(adIDs)
	if err != nil {
		log.Printf("[BuildAdEmbeddings] Failed to mark ads as having vector: %v", err)
	} else {
		log.Printf("[BuildAdEmbeddings] Successfully marked %d ads as having vector", len(adIDs))
	}

	return nil
}

// buildAdEmbeddingPrompt creates a prompt for generating embeddings
func buildAdEmbeddingPrompt(adObj ad.AdDetail) string {
	// Get parent company information for the make
	var parentCompanyStr, parentCompanyCountry string
	if adObj.Make != "" {
		if pcInfo, err := vehicle.GetParentCompanyInfoForMake(adObj.Make); err == nil && pcInfo != nil {
			parentCompanyStr = pcInfo.Name
			parentCompanyCountry = pcInfo.Country
		}
	}

	// Get rock count for this ad
	rockCount := 0
	if count, err := rock.GetAdRockCount(adObj.ID); err == nil {
		rockCount = count
	}

	// Include rock count in the embedding - ads with fewer rocks should rank higher
	rockContext := ""
	if rockCount == 0 {
		rockContext = "This ad has no reported issues (0 rocks thrown)."
	} else if rockCount == 1 {
		rockContext = "This ad has 1 reported issue (1 rock thrown)."
	} else {
		rockContext = fmt.Sprintf("This ad has %d reported issues (%d rocks thrown).",
			rockCount, rockCount)
	}

	promptTemplate := `Encode the following ad for semantic search. Focus on
	title and description, what the part is, what vehicles it fits, where
	the part is located, the price, how many rocks have been thown, and any
	relevant details for a buyer.  Return only the embedding vector.

Title: %s
Description: %s
Make: %s
Parent Company: %s
Parent Company Country: %s
Years: %s
Models: %s
Engines: %s
Category: %s
Location: %s, %s, %s
Quality Indicator: %s`

	return fmt.Sprintf(promptTemplate,
		adObj.Title,
		adObj.Description,
		adObj.Make,
		parentCompanyStr,
		parentCompanyCountry,
		joinStrings(adObj.Years),
		joinStrings(adObj.Models),
		joinStrings(adObj.Engines),
		adObj.PartCategory,
		adObj.City,
		adObj.AdminArea,
		adObj.Country,
		rockContext,
	)
}

// buildAdEmbeddingMetadata creates metadata for embeddings
func buildAdEmbeddingMetadata(adObj ad.AdDetail) map[string]interface{} {

	lat, lon, _ := ad.GetLatLon(adObj.LocationID)
	rockCount, _ := rock.GetAdRockCount(adObj.ID)

	metadata := map[string]interface{}{
		"ad_category_id": adObj.AdCategoryID,
		"make":           adObj.Make,
		"years":          adObj.Years,
		"models":         adObj.Models,
		"engines":        adObj.Engines,
		"category":       adObj.PartCategory,
		"subcategory":    adObj.PartSubcategory,
		"price":          adObj.Price,
		"rock_count":     rockCount,
	}

	// Add geo payload if we have coordinates
	if lat != 0 && lon != 0 {
		metadata["location"] = map[string]interface{}{
			"lat": lat,
			"lon": lon,
		}
	}

	return metadata
}

// Helper function for embedding generation
func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return fmt.Sprintf("%s", ss)
}

// StartBackgroundProcessor starts the background processor
func StartBackgroundProcessor() {
	go func() {
		log.Printf("[vector] Background vector processor started (queue-based)")

		const chunkSize = 50

		for {
			// Collect ads up to chunk size
			var ads []ad.Ad

			// Get the first ad (blocking)
			adObj := <-adQueue
			ads = append(ads, adObj)

			// Collect additional ads up to chunk size
			for i := 1; i < chunkSize; i++ {
				select {
				case adObj := <-adQueue:
					ads = append(ads, adObj)
				default:
					// No more ads available, break out of the inner loop
					goto processChunk
				}
			}

		processChunk:
			log.Printf("[vector] Processing chunk of %d ads from queue", len(ads))

			// Convert minimal ads to full ad details for processing
			var adDetails []ad.AdDetail
			for _, adObj := range ads {
				adDetail, err := ad.GetAdDetailByID(adObj.ID, 0)
				if err != nil {
					log.Printf("[vector] Failed to get ad detail for %d: %v", adObj.ID, err)
					continue
				}
				adDetails = append(adDetails, *adDetail)
			}

			if len(adDetails) == 0 {
				log.Printf("[vector] No valid ad details found for chunk")
				continue
			}

			err := BuildAdEmbeddings(adDetails)
			if err != nil {
				log.Printf("[vector] Error building embeddings for chunk: %v", err)
			} else {
				log.Printf("[vector] Successfully processed chunk of %d ads", len(ads))
			}

			// Sleep to avoid rate limits
			time.Sleep(config.QdrantProcessingSleepInterval)
		}
	}()
}

// QueueAd adds an ad to the processing queue
func QueueAd(adObj ad.Ad) {
	adQueue <- adObj
}

// ProcessAdsWithoutVectors loads ads without vectors and queues them for processing
func ProcessAdsWithoutVectors() {
	go func() {
		ads, err := ad.GetAdsWithoutVectors()
		if err != nil {
			log.Printf("[vector] Error getting ads without vectors: %v", err)
			return
		}

		if len(ads) == 0 {
			log.Printf("[vector] No ads without vectors found")
			return
		}

		log.Printf("[vector] Queueing %d ads without vectors for processing", len(ads))

		// Queue all ads for background processing
		for _, adObj := range ads {
			QueueAd(adObj)
		}
	}()
}
