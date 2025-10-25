package vector

import (
	"fmt"
	"log"

	"github.com/parts-pile/site/ad"
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
	log.Printf("[site-level] Calculating site-level vector from %d popular ads", len(ads))
	log.Printf("[site-level] Popular ads being used:")
	for i, adObj := range ads {
		if i < 5 { // Only show first 5 for brevity
			log.Printf("[site-level]   %d. Ad %d: %s",
				i+1, adObj.ID, adObj.Title)
		}
	}
	if len(ads) > 5 {
		log.Printf("[site-level]   ... and %d more ads", len(ads)-5)
	}

	// Extract ad IDs for batch retrieval
	var adIDs []int
	for _, adObj := range ads {
		adIDs = append(adIDs, adObj.ID)
	}

	// Fetch all embeddings in batch
	log.Printf("[site-level] Fetching embeddings for %d ads in batch", len(adIDs))
	embeddings, err := GetAdEmbeddings(adIDs)
	if err != nil {
		log.Printf("[site-level] Batch fetch error: %v", err)
		return nil, err
	}

	// Process results
	var vectors [][]float32
	var missingIDs []int
	for i, embedding := range embeddings {
		adID := adIDs[i]
		if embedding == nil {
			missingIDs = append(missingIDs, adID)
			log.Printf("[site-level] Missing embedding for ad %d", adID)
			continue
		}
		log.Printf("[site-level] Found embedding for ad %d (length=%d)", adID, len(embedding))
		vectors = append(vectors, embedding)
	}

	log.Printf("[site-level] Found embeddings for %d ads, missing for %d ads", len(vectors), len(missingIDs))
	if len(missingIDs) > 0 {
		log.Printf("[site-level] Missing embeddings for ad IDs: %v", missingIDs)
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("no embeddings found for popular ads")
	}

	// Use unweighted mean for now
	weights := make([]float32, len(vectors))
	for i := range weights {
		weights[i] = 1.0
	}
	log.Printf("[site-level] Aggregating %d embeddings with equal weights", len(vectors))
	result := AggregateEmbeddings(vectors, weights)
	log.Printf("[site-level] AggregateEmbeddings returned result=%v (length=%d)", result != nil, len(result))
	if result == nil {
		return nil, fmt.Errorf("AggregateEmbeddings returned nil")
	}

	// Enhance site-level vector with rock preference for anonymous users
	rockPreferencePrompt := "Show me high-quality ads with fewer reported issues (rocks thrown). I prefer reliable, trustworthy listings."
	rockPreferenceEmbedding, err := GetQueryEmbedding(rockPreferencePrompt)
	if err == nil && rockPreferenceEmbedding != nil {
		// Blend the site-level vector with rock preference
		enhancedVectors := [][]float32{result, rockPreferenceEmbedding}
		enhancedWeights := []float32{1.0, 1.5} // Site-level gets weight 1.0, rock preference gets 1.5
		result = AggregateEmbeddings(enhancedVectors, enhancedWeights)
		log.Printf("[site-level] Enhanced site-level vector with rock preference")
	}

	return result, nil
}
