package vector

import (
	"fmt"
	"strings"
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
	// Get category fields to determine which fields to include
	fields := spec.GetCategoryFields(adObj.AdCategoryID)

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

	// Build prompt header
	var promptParts []string
	promptParts = append(promptParts, `Encode the following ad for semantic search. Focus on
title and description, what the part is, what vehicles it fits, where
the part is located, the price, how many rocks have been thown, and any
relevant details for a buyer. When considering vehicle makes, consider the
parent company and country of origin of the make, as well as sibling makes.
For example, Ford and Mercury are related by both being under Ford Motor Company.
Return only the embedding vector.

Title: `+adObj.Title+`
Description: `+adObj.Description)

	// Add spec fields dynamically - each field generates its own snippet
	for _, field := range fields {
		fieldConfig := field.Config()
		fieldValues, exists := adObj.SpecData[fieldConfig.Name]
		if !exists {
			continue
		}

		snippet := spec.BuildFieldPromptSnippet(field, fieldValues)
		if snippet != "" {
			promptParts = append(promptParts, snippet)
		}
	}

	// Add part category/subcategory if available (not in fieldConfigs but always present for parts)
	if catValues, exists := adObj.SpecData["part_category"]; exists && len(catValues) > 0 {
		promptParts = append(promptParts, fmt.Sprintf("Category: %s\n", catValues[0]))
	}
	if subcatValues, exists := adObj.SpecData["part_subcategory"]; exists && len(subcatValues) > 0 {
		promptParts = append(promptParts, fmt.Sprintf("Subcategory: %s\n", subcatValues[0]))
	}

	// Add location and quality indicator
	promptParts = append(promptParts, fmt.Sprintf("Location: %s, %s, %s\n", adObj.City, adObj.AdminArea, adObj.Country))
	promptParts = append(promptParts, fmt.Sprintf("Quality Indicator: %s", rockContext))

	return strings.Join(promptParts, "\n")
}

// buildAdEmbeddingMetadata creates metadata for embeddings dynamically based on category field configuration
func buildAdEmbeddingMetadata(adObj ad.AdDetail) map[string]interface{} {
	// Initialize metadata with non-SpecData fields
	lat, lon, _ := ad.GetLatLon(adObj.LocationID)
	rockCount, _ := rock.GetAdRockCount(adObj.ID)

	metadata := map[string]interface{}{
		"ad_category_id": adObj.AdCategoryID,
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

	// Get category fields to determine which fields to include
	fields := spec.GetCategoryFields(adObj.AdCategoryID)

	// Add spec fields dynamically based on category configuration
	for _, field := range fields {
		config := field.Config()
		fieldName := config.Name
		// Extract value from SpecData
		fieldValues, exists := adObj.SpecData[fieldName]
		if !exists || len(fieldValues) == 0 {
			continue
		}

		// Filter out empty values
		nonEmptyValues := []string{}
		for _, val := range fieldValues {
			if val != "" {
				nonEmptyValues = append(nonEmptyValues, val)
			}
		}
		if len(nonEmptyValues) == 0 {
			continue
		}

		// Add to metadata based on field type - use fieldName directly
		if config.FieldMultiValue {
			// Multi-value field - store as array
			metadata[fieldName] = nonEmptyValues
		} else {
			// Single-value field - store as string
			metadata[fieldName] = nonEmptyValues[0]
		}
	}

	return metadata
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
