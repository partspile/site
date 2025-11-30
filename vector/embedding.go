package vector

import (
	"fmt"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/logger"
)

// AdResult represents a search result from PostgreSQL vector search
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

// calculateWeightedVectors processes activities and returns vectors with age-based weights
// Uses hyperbolic decay with 30-day half-life: newer activities get higher weights
func calculateWeightedVectors(activities []AdActivity) ([][]float32, []float32) {
	now := time.Now()
	var vectors [][]float32
	var weights []float32

	const halfLifeDays = 30.0

	for _, act := range activities {
		// Calculate age in days
		ageDays := now.Sub(act.Timestamp).Hours() / 24.0

		// Weight calculation: hyperbolic decay with half-life of 30 days
		// Newer items get higher weights, weight decreases as age increases
		// Weight = 1 / (1 + ageDays / halfLifeDays)
		// This gives: ageDays=0 -> weight=1, ageDays=30 -> weight=0.5
		// The absolute value doesn't matter since weights are normalized in aggregation
		weight := float32(1.0 / (1.0 + ageDays/halfLifeDays))

		vectors = append(vectors, act.Embedding)
		weights = append(weights, weight)
	}

	return vectors, weights
}

// calculateSiteLevelVector calculates site-level embedding for a category and optional campaign using the same algorithm as user embeddings
// Gets embeddings for top N most recent ads with bookmarks or clicks (across all users) and weights by age
func calculateSiteLevelVector(categoryID int, campaignKey string) ([]float32, error) {
	if categoryID <= 0 {
		return nil, fmt.Errorf("categoryID must be greater than 0")
	}

	logger.Debug("Calculating site-level vector",
		"component", "site-level", "categoryID", categoryID, "campaignKey", campaignKey)

	// Fetch combined bookmarks and clicks for all users in this category, sorted by timestamp (newest first), limited
	limit := config.VectorSystemEmbeddingLimit
	activities, err := getSiteActivities(categoryID, campaignKey, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch site activities: %w", err)
	}

	if len(activities) == 0 {
		// No activities found, generate embedding from a simple prompt
		categoryName := ad.GetCategoryName(categoryID)
		prompt := fmt.Sprintf("any ad in the %s category", categoryName)
		embedding, err := EmbedText(prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to generate fallback embedding: %w", err)
		}
		logger.Info("Generated site-level embedding from fallback prompt",
			"component", "site-level", "categoryID", categoryID, "campaignKey", campaignKey, "prompt", prompt)
		return embedding, nil
	}

	logger.Debug("Site activities",
		"component", "site-level", "categoryID", categoryID, "campaignKey", campaignKey, "count", len(activities))

	vectors, weights := calculateWeightedVectors(activities)

	logger.Debug("Vectors aggregated",
		"component", "site-level", "categoryID", categoryID, "vectors", len(vectors),
		"weights", len(weights))

	emb := AggregateEmbeddings(vectors, weights)
	if emb == nil {
		return nil, fmt.Errorf("AggregateEmbeddings returned nil")
	}

	logger.Info("Successfully created site-level embedding",
		"component", "site-level", "categoryID", categoryID, "campaignKey", campaignKey, "vectors", len(vectors),
		"activities", len(activities))

	return emb, nil
}
