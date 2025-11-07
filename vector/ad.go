package vector

import (
	"fmt"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/rock"
	"github.com/parts-pile/site/spec"
)

// Static queue for ad processing
var adQueue = make(chan int, config.QdrantProcessingQueueSize)

// BuildAdEmbedding builds and stores an embedding for a single ad
func BuildAdEmbedding(adObj ad.AdDetail) error {
	return BuildAdEmbeddings([]ad.AdDetail{adObj})
}

// BuildAdEmbeddings builds and stores embeddings for multiple ads in batch
func BuildAdEmbeddings(ads []ad.AdDetail) error {
	if len(ads) == 0 {
		return nil
	}

	logger.Debug("Building embeddings for ads in batch",
		"count", len(ads))

	// Build prompts for all ads
	var prompts []string
	for _, adObj := range ads {
		prompt := buildAdEmbeddingPrompt(adObj)
		prompts = append(prompts, prompt)
	}

	// Generate embeddings in batch
	logger.Debug("Generating batch embeddings",
		"count", len(ads))
	embeddings, err := EmbedTexts(prompts)
	if err != nil {
		logger.Error("Failed to generate batch embeddings",
			"error", err, "count", len(ads))
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
		logger.Error("Failed to batch upsert vectors",
			"error", err, "count", len(adIDs))
	} else {
		logger.Debug("Successfully batch upserted vectors",
			"count", len(adIDs))
	}

	// Mark ads as having vectors in database
	err = ad.MarkAdsAsHavingVector(adIDs)
	if err != nil {
		logger.Error("Failed to mark ads as having vector",
			"error", err, "count", len(adIDs))
	} else {
		logger.Debug("Successfully marked ads as having vector",
			"count", len(adIDs))
	}

	return nil
}

// buildAdEmbeddingPrompt creates a prompt for generating embeddings
func buildAdEmbeddingPrompt(adObj ad.AdDetail) string {
	// Get parent company information for the make
	var parentCompanyStr, parentCompanyCountry string
	if adObj.Make != "" {
		if pcInfo, err := spec.GetParentCompanyInfoForMake(adObj.Make); err == nil && pcInfo != nil {
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

// buildAdEmbeddingMetadata creates metadata for embeddings dynamically based on category field configuration
func buildAdEmbeddingMetadata(adObj ad.AdDetail) map[string]interface{} {
	lat, lon, _ := ad.GetLatLon(adObj.LocationID)
	rockCount, _ := rock.GetAdRockCount(adObj.ID)

	// Get category fields to determine which fields to include
	fieldConfigs, err := spec.GetCategoryFields(adObj.AdCategoryID)
	if err != nil {
		logger.Warn("Failed to get category fields for metadata",
			"error", err, "component", "vector", "adID", adObj.ID)
		// Fallback to empty metadata
		return map[string]interface{}{
			"ad_category_id": adObj.AdCategoryID,
			"price":          adObj.Price,
			"rock_count":     rockCount,
		}
	}

	metadata := map[string]interface{}{
		"ad_category_id": adObj.AdCategoryID,
		"price":          adObj.Price,
		"rock_count":     rockCount,
	}

	// Add part category/subcategory (these are not spec fields but are used)
	if adObj.PartCategory != "" {
		metadata["category"] = adObj.PartCategory
	}
	if adObj.PartSubcategory != "" {
		metadata["subcategory"] = adObj.PartSubcategory
	}

	// Add spec fields dynamically based on category configuration
	// This is a temporary compatibility layer - AdDetail has hard-coded fields
	for _, fieldConfig := range fieldConfigs {
		// Determine Qdrant metadata key (plural for arrays)
		qdrantKey := fieldConfig.Name
		if fieldConfig.QdrantType == "keyword_array" {
			qdrantKey = fieldConfig.Name + "s" // years, models, engines
		}

		// Extract value from AdDetail based on field name
		// TODO: Make AdDetail more flexible to support arbitrary fields
		switch fieldConfig.Name {
		case "make":
			if adObj.Make != "" {
				metadata[qdrantKey] = adObj.Make
			}
		case "year":
			if len(adObj.Years) > 0 {
				metadata[qdrantKey] = adObj.Years
			}
		case "model":
			if len(adObj.Models) > 0 {
				metadata[qdrantKey] = adObj.Models
			}
		case "engine":
			if len(adObj.Engines) > 0 {
				metadata[qdrantKey] = adObj.Engines
			}
		case "breed":
			// TODO: Add breed field to AdDetail when needed
			// For now, skip
		default:
			// For future fields, log a warning but don't fail
			logger.Debug("Field not yet supported in AdDetail",
				"component", "vector", "field", fieldConfig.Name, "adID", adObj.ID)
		}
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
		logger.Info("Background vector processor started",
			"component", "vector", "queueSize", config.QdrantProcessingQueueSize)

		const chunkSize = 50

		for {
			// Collect ad IDs up to chunk size
			var adIDs []int

			// Get the first ad ID (blocking)
			adID := <-adQueue
			adIDs = append(adIDs, adID)

			// Collect additional ad IDs up to chunk size
			for i := 1; i < chunkSize; i++ {
				select {
				case adID := <-adQueue:
					adIDs = append(adIDs, adID)
				default:
					// No more ads available, break out of the inner loop
					goto processChunk
				}
			}

		processChunk:
			logger.Info("Processing chunk of ads from queue",
				"component", "vector", "count", len(adIDs))

			// Convert ad IDs to full ad details for processing
			var adDetails []ad.AdDetail
			for _, adID := range adIDs {
				adDetail, err := ad.GetAdDetailByID(adID, 0)
				if err != nil {
					logger.Error("Failed to get ad detail",
						"error", err, "adID", adID, "component", "vector")
					continue
				}
				adDetails = append(adDetails, *adDetail)
			}

			if len(adDetails) == 0 {
				logger.Debug("No valid ad details found for chunk",
					"component", "vector")
				continue
			}

			err := BuildAdEmbeddings(adDetails)
			if err != nil {
				logger.Error("Error building embeddings for chunk",
					"error", err, "count", len(adIDs), "component", "vector")
			} else {
				logger.Info("Successfully processed chunk of ads",
					"component", "vector", "count", len(adIDs))
			}

			// Sleep to avoid rate limits
			time.Sleep(config.QdrantProcessingSleepInterval)
		}
	}()
}

// QueueAd adds an ad ID to the processing queue
func QueueAd(adID int) {
	adQueue <- adID
}

// ProcessAdsWithoutVectors loads ads without vectors and queues them for processing
func ProcessAdsWithoutVectors() {
	go func() {
		adIDs, err := ad.GetAdsWithoutVectors()
		if err != nil {
			logger.Error("Error getting ads without vectors",
				"error", err, "component", "vector")
			return
		}

		if len(adIDs) == 0 {
			logger.Debug("No ads without vectors found",
				"component", "vector")
			return
		}

		logger.Info("Queueing ads without vectors for processing",
			"component", "vector", "count", len(adIDs))

		// Queue all ad IDs for background processing
		for _, adID := range adIDs {
			QueueAd(adID)
		}
	}()
}
