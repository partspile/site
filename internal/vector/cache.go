package vector

import (
	"errors"
	"fmt"
	"strings"

	"github.com/partspile/site/internal/cache"
	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/logger"
)

var (
	queryEmbeddingCache *cache.Cache[[]float32]
	userEmbeddingCache  *cache.Cache[[]float32]
	siteEmbeddingCache  *cache.Cache[[]float32]
)

func InitEmbeddingCaches() error {
	cost := func(v []float32) int64 { return int64(len(v) * 4) }
	var err error
	queryEmbeddingCache, err = cache.New(cost, "Query Embedding Cache")
	if err != nil {
		return fmt.Errorf("query cache: %w", err)
	}
	userEmbeddingCache, err = cache.New(cost, "User Embedding Cache")
	if err != nil {
		return fmt.Errorf("user cache: %w", err)
	}
	siteEmbeddingCache, err = cache.New(cost, "Site Embedding Cache")
	if err != nil {
		return fmt.Errorf("site cache: %w", err)
	}
	return nil
}

func GetQueryEmbedding(userQuery string) ([]float32, error) {
	userQuery = strings.TrimSpace(userQuery)
	if userQuery == "" {
		return nil, fmt.Errorf("cannot embed empty query")
	}
	key := userQuery
	if cached, ok := queryEmbeddingCache.Get(key); ok {
		logger.Debug("query embedding cache hit", "query", userQuery)
		return cached, nil
	}
	logger.Debug("query embedding cache miss", "query", userQuery)
	embedding, err := embedQuery(userQuery)
	if err != nil {
		logger.Debug("query embedding failed",
			"query", userQuery,
			"error", err,
		)
		return nil, err
	}
	queryEmbeddingCache.SetWithTTL(
		key, embedding, int64(len(embedding)*4),
		config.VectorQueryEmbeddingCacheTTL,
	)
	return embedding, nil
}

func GetUserPersonalizedEmbedding(userID, categoryID int) ([]float32, error) {
	if cached, err := getUserEmbedding(userID, categoryID); err == nil {
		logger.Debug("user embedding cache hit",
			"userID", userID,
			"categoryID", categoryID,
		)
		return cached, nil
	}
	activities, err := getUserActivities(
		userID, categoryID, config.VectorUserEmbeddingLimit,
	)
	if err != nil {
		logger.Debug("user embedding activity load failed",
			"userID", userID,
			"categoryID", categoryID,
			"error", err,
		)
		return nil, err
	}
	if len(activities) == 0 {
		logger.Debug("user embedding no activity",
			"userID", userID,
			"categoryID", categoryID,
		)
		return nil, ErrNoUserActivity
	}
	vectors, weights := calculateWeightedVectors(activities)
	emb := AggregateEmbeddings(vectors, weights)
	if emb == nil {
		return nil, ErrNoUserActivity
	}
	_ = setUserEmbedding(userID, categoryID, emb)
	logger.Debug("user embedding computed",
		"userID", userID,
		"categoryID", categoryID,
		"activities", len(activities),
	)
	return emb, nil
}

func getUserEmbedding(userID, categoryID int) ([]float32, error) {
	key := fmt.Sprintf("user_%d_cat_%d", userID, categoryID)
	cached, ok := userEmbeddingCache.Get(key)
	if !ok {
		return nil, fmt.Errorf("user embedding not found in cache")
	}
	return cached, nil
}

func setUserEmbedding(userID, categoryID int, embedding []float32) error {
	key := fmt.Sprintf("user_%d_cat_%d", userID, categoryID)
	userEmbeddingCache.SetWithTTL(
		key, embedding, int64(len(embedding)*4),
		config.VectorUserEmbeddingCacheTTL,
	)
	return nil
}

func GetSiteEmbedding(categoryID int, campaignKey string) ([]float32, error) {
	if categoryID <= 0 {
		return nil, fmt.Errorf("categoryID must be greater than 0")
	}
	key := fmt.Sprintf("site_cat_%d_%s", categoryID, campaignKey)
	if cached, ok := siteEmbeddingCache.Get(key); ok {
		logger.Debug("site embedding cache hit",
			"categoryID", categoryID,
			"campaignKey", campaignKey,
		)
		return cached, nil
	}
	logger.Debug("site embedding cache miss",
		"categoryID", categoryID,
		"campaignKey", campaignKey,
	)
	embedding, err := calculateSiteLevelVector(categoryID, campaignKey)
	if err != nil {
		logger.Debug("site embedding failed",
			"categoryID", categoryID,
			"campaignKey", campaignKey,
			"error", err,
		)
		return nil, err
	}
	siteEmbeddingCache.SetWithTTL(
		key, embedding, int64(len(embedding)*4),
		config.VectorSiteEmbeddingCacheTTL,
	)
	return embedding, nil
}

func ResolveSearchEmbedding(userID, categoryID int, query string) ([]float32, error) {
	query = strings.TrimSpace(query)
	if query != "" {
		return GetQueryEmbedding(query)
	}
	if userID != 0 {
		emb, err := GetUserPersonalizedEmbedding(userID, categoryID)
		if err == nil {
			return emb, nil
		}
		if !errors.Is(err, ErrNoUserActivity) {
			return nil, err
		}
		logger.Debug("search embedding user fallback",
			"categoryID", categoryID,
			"userID", userID,
			"reason", "no user activity",
		)
	}
	return GetSiteEmbedding(categoryID, "default")
}
