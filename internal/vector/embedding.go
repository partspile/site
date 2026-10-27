package vector

import (
	"fmt"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/logger"
)

type AdActivity struct {
	AdID         int       `json:"ad_id"`
	Timestamp    string    `json:"timestamp"`
	Embedding    []float32 `json:"embedding"`
	ActivityType string    `json:"activity_type"`
	Weight       float32   `json:"weight"`
}

func AggregateEmbeddings(vectors [][]float32, weights []float32) []float32 {
	if len(vectors) == 0 || len(weights) == 0 || len(vectors) != len(weights) {
		return nil
	}
	vecLen := len(vectors[0])
	result := make([]float32, vecLen)
	var totalWeight float32
	for i, vec := range vectors {
		if len(vec) != vecLen {
			continue
		}
		w := weights[i]
		totalWeight += w
		for j := range vec {
			result[j] += vec[j] * w
		}
	}
	if totalWeight == 0 {
		return nil
	}
	for j := range result {
		result[j] /= totalWeight
	}
	return result
}

func calculateWeightedVectors(activities []AdActivity) ([][]float32, []float32) {
	var vectors [][]float32
	var weights []float32
	for _, act := range activities {
		if len(act.Embedding) == 0 {
			continue
		}
		vectors = append(vectors, act.Embedding)
		weights = append(weights, act.Weight)
	}
	return vectors, weights
}

func calculateSiteLevelVector(categoryID int, campaignKey string) ([]float32, error) {
	if categoryID <= 0 {
		return nil, fmt.Errorf("categoryID must be greater than 0")
	}
	limit := config.VectorSiteEmbeddingLimit
	activities, err := getSiteActivities(categoryID, campaignKey, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch site activities: %w", err)
	}
	if len(activities) > 0 {
		vectors, weights := calculateWeightedVectors(activities)
		emb := AggregateEmbeddings(vectors, weights)
		if emb != nil {
			logger.Debug("site embedding aggregated",
				"categoryID", categoryID, "activities", len(activities))
			return emb, nil
		}
	}
	emb, err := aggregateRecentAdEmbeddings(categoryID)
	if err != nil {
		return nil, err
	}
	return emb, nil
}

func aggregateRecentAdEmbeddings(categoryID int) ([]float32, error) {
	emb, err := averageRecentAdEmbeddings(categoryID)
	if err == nil && emb != nil {
		return emb, nil
	}
	if categoryID <= 0 {
		return nil, ErrNoEmbeddedAds
	}
	emb, err = averageRecentAdEmbeddings(0)
	if err != nil {
		return nil, err
	}
	if emb == nil {
		return nil, ErrNoEmbeddedAds
	}
	logger.Debug("site embedding from recent ads",
		"categoryID", categoryID, "source", "site-wide")
	return emb, nil
}

func averageRecentAdEmbeddings(categoryID int) ([]float32, error) {
	ids, err := recentAdEmbeddingIDs(categoryID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrNoEmbeddedAds
	}
	embeddings, err := GetAdEmbeddings(ids)
	if err != nil {
		return nil, err
	}
	var vectors [][]float32
	weights := make([]float32, 0, len(embeddings))
	for _, emb := range embeddings {
		if len(emb) == 0 {
			continue
		}
		vectors = append(vectors, emb)
		weights = append(weights, 1)
	}
	if len(vectors) == 0 {
		return nil, ErrNoEmbeddedAds
	}
	return AggregateEmbeddings(vectors, weights), nil
}

func recentAdEmbeddingIDs(categoryID int) ([]int, error) {
	var ids []int
	err := db.QueryJSON(&ids, `
		SELECT COALESCE(json_agg(id), '[]'::json)
		FROM (
			SELECT id FROM ads
			WHERE inactive_at IS NULL AND deleted_at IS NULL
			  AND embedding IS NOT NULL
			  AND ($1 = 0 OR category_id = $1)
			ORDER BY created_at DESC
			LIMIT $2
		) recent`,
		categoryID, config.VectorSystemEmbeddingLimit,
	)
	return ids, err
}
