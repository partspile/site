package vector

import (
	"fmt"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/logger"
)

// AdResult represents a search result from Qdrant
type AdResult struct {
	ID       int
	Score    float32
	Metadata map[string]interface{}
}

// AggregateEmbeddings computes a weighted mean of multiple embeddings
func AggregateEmbeddings(vectors [][]float32, weights []float32) []float32 {
	if len(vectors) == 0 || len(weights) == 0 || len(vectors) != len(weights) {
		return nil
	}
	vecLen := len(vectors[0])
	result := make([]float32, vecLen)
	var totalWeight float32
	for i, vec := range vectors {
		if len(vec) != vecLen {
			// Skip vectors of mismatched length
			continue
		}
		w := weights[i]
		totalWeight += w
		for j := 0; j < vecLen; j++ {
			result[j] += vec[j] * w
		}
	}
	if totalWeight == 0 {
		return nil
	}
	for j := 0; j < vecLen; j++ {
		result[j] /= totalWeight
	}
	return result
}

// calculateSiteLevelVector averages the embeddings of the most popular ads
func calculateSiteLevelVector() ([]float32, error) {
	ads := ad.GetMostPopularAds(50)
	logger.Debug("Calculating site-level vector from popular ads",
		"component", "site-level", "count", len(ads))

	// Extract ad IDs for batch retrieval
	var adIDs []int
	for _, adObj := range ads {
		adIDs = append(adIDs, adObj.ID)
	}

	// Fetch all embeddings in batch
	logger.Debug("Fetching embeddings for ads in batch",
		"component", "site-level", "count", len(adIDs))
	embeddings, err := GetAdEmbeddings(adIDs)
	if err != nil {
		logger.Error("Batch fetch error",
			"error", err, "component", "site-level")
		return nil, err
	}

	// Process results
	var vectors [][]float32
	var missingIDs []int
	for i, embedding := range embeddings {
		adID := adIDs[i]
		if embedding == nil {
			missingIDs = append(missingIDs, adID)
			logger.Debug("Missing embedding for ad",
				"component", "site-level", "adID", adID)
			continue
		}
		logger.Debug("Found embedding for ad",
			"component", "site-level", "adID", adID,
			"length", len(embedding))
		vectors = append(vectors, embedding)
	}

	logger.Debug("Found embeddings",
		"component", "site-level", "found", len(vectors),
		"missing", len(missingIDs), "missingIDs", missingIDs)
	if len(vectors) == 0 {
		return nil, fmt.Errorf("no embeddings found for popular ads")
	}

	// Use unweighted mean for now
	weights := make([]float32, len(vectors))
	for i := range weights {
		weights[i] = 1.0
	}
	logger.Debug("Aggregating embeddings",
		"component", "site-level", "count", len(vectors))
	result := AggregateEmbeddings(vectors, weights)
	logger.Debug("AggregateEmbeddings returned",
		"component", "site-level", "resultNonNil", result != nil,
		"length", len(result))
	if result == nil {
		return nil, fmt.Errorf("AggregateEmbeddings returned nil")
	}

	return result, nil
}
