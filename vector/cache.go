package vector

import (
	"fmt"
	"strings"
	"time"

	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/logger"
)

var (
	// Specialized caches for different embedding types
	queryEmbeddingCache *cache.Cache[[]float32] // String keys, 1 hour TTL
	userEmbeddingCache  *cache.Cache[[]float32] // User ID keys, 24 hour TTL
	siteEmbeddingCache  *cache.Cache[[]float32] // Campaign keys, 6 hour TTL
)

// InitEmbeddingCaches initializes the specialized embedding caches. This should be called during application startup.
func InitEmbeddingCaches() error {
	var err error

	// Initialize query cache (1 hour TTL, smaller size)
	queryEmbeddingCache, err = cache.New[[]float32](func(value []float32) int64 {
		return int64(len(value) * 4) // 4 bytes per float32
	}, "Query Embedding Cache")
	if err != nil {
		return fmt.Errorf("failed to initialize query embedding cache: %w", err)
	}

	// Initialize user cache (24 hour TTL, larger size)
	userEmbeddingCache, err = cache.New[[]float32](func(value []float32) int64 {
		return int64(len(value) * 4) // 4 bytes per float32
	}, "User Embedding Cache")
	if err != nil {
		return fmt.Errorf("failed to initialize user embedding cache: %w", err)
	}

	// Initialize site cache (6 hour TTL, medium size)
	siteEmbeddingCache, err = cache.New[[]float32](func(value []float32) int64 {
		return int64(len(value) * 4) // 4 bytes per float32
	}, "Site Embedding Cache")
	if err != nil {
		return fmt.Errorf("failed to initialize site embedding cache: %w", err)
	}

	return nil
}

// GetEmbeddingCacheStats returns cache statistics for admin monitoring
func GetEmbeddingCacheStats() map[string]interface{} {
	stats := make(map[string]interface{})

	if queryEmbeddingCache != nil {
		stats["query"] = queryEmbeddingCache.Stats()
	}
	if userEmbeddingCache != nil {
		stats["user"] = userEmbeddingCache.Stats()
	}
	if siteEmbeddingCache != nil {
		stats["site"] = siteEmbeddingCache.Stats()
	}

	return stats
}

// ClearQueryEmbeddingCache clears only the query embedding cache
func ClearQueryEmbeddingCache() {
	if queryEmbeddingCache != nil {
		queryEmbeddingCache.Clear()
	}
}

// ClearUserEmbeddingCache clears only the user embedding cache
func ClearUserEmbeddingCache() {
	if userEmbeddingCache != nil {
		userEmbeddingCache.Clear()
	}
}

// ClearSiteEmbeddingCache clears only the site embedding cache
func ClearSiteEmbeddingCache() {
	if siteEmbeddingCache != nil {
		siteEmbeddingCache.Clear()
	}
}

// ClearEmbeddingCache clears all embedding caches
func ClearEmbeddingCache() {
	ClearQueryEmbeddingCache()
	ClearUserEmbeddingCache()
	ClearSiteEmbeddingCache()
}

// GetQueryEmbedding retrieves or generates an embedding for the given text using the query cache
func GetQueryEmbedding(text string) ([]float32, error) {
	if queryEmbeddingCache == nil {
		return nil, fmt.Errorf("query embedding cache not initialized")
	}

	// Trim whitespace and check for empty
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("cannot embed empty text")
	}

	// Check cache first
	if cached, found := queryEmbeddingCache.Get(text); found {
		logger.Debug("Cache hit for query",
			"component", "embedding", "subcomponent",
			"query-cache", "query", text)
		return cached, nil
	}

	// Generate embedding
	embedding, err := EmbedText(text)
	if err != nil {
		return nil, err
	}

	// Cache the result with 1 hour TTL
	queryEmbeddingCache.SetWithTTL(text, embedding, int64(len(embedding)*4), time.Hour)
	logger.Debug("Cached embedding for query",
		"component", "embedding", "subcomponent",
		"query-cache", "query", text)

	return embedding, nil
}

// GetQueryEmbeddings retrieves or generates embeddings for multiple texts using the query cache
// This function optimizes by batching uncached texts into a single API call
func GetQueryEmbeddings(texts []string) ([][]float32, error) {
	if queryEmbeddingCache == nil {
		return nil, fmt.Errorf("query embedding cache not initialized")
	}

	if len(texts) == 0 {
		return nil, fmt.Errorf("cannot embed empty text array")
	}

	// Trim whitespace and validate all texts
	var validTexts []string
	for i, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("cannot embed empty text at index %d", i)
		}
		validTexts = append(validTexts, text)
	}

	// Check cache for each text and collect uncached ones
	var cachedEmbeddings [][]float32
	var uncachedTexts []string
	var uncachedIndices []int

	for i, text := range validTexts {
		if cached, found := queryEmbeddingCache.Get(text); found {
			logger.Debug("Cache hit for query",
				"component", "embedding", "subcomponent",
				"query-cache", "query", text)
			cachedEmbeddings = append(cachedEmbeddings, cached)
		} else {
			uncachedTexts = append(uncachedTexts, text)
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// Generate embeddings for uncached texts in batch
	var newEmbeddings [][]float32
	if len(uncachedTexts) > 0 {
		logger.Debug("Cache miss, generating batch embeddings",
			"component", "embedding", "subcomponent",
			"query-cache", "count", len(uncachedTexts))
		embeddings, err := EmbedTexts(uncachedTexts)
		if err != nil {
			return nil, err
		}

		// Cache the new embeddings
		for i, embedding := range embeddings {
			text := uncachedTexts[i]
			queryEmbeddingCache.SetWithTTL(text, embedding, int64(len(embedding)*4), time.Hour)
			logger.Debug("Cached embedding for query",
				"component", "embedding", "subcomponent",
				"query-cache", "query", text)
		}
		newEmbeddings = embeddings
	}

	// Combine cached and new embeddings in the correct order
	result := make([][]float32, len(validTexts))
	cachedIdx := 0
	newIdx := 0

	for i := range validTexts {
		if newIdx < len(uncachedIndices) && i == uncachedIndices[newIdx] {
			// This was a cache miss, use new embedding
			result[i] = newEmbeddings[newIdx]
			newIdx++
		} else {
			// This was a cache hit, use cached embedding
			result[i] = cachedEmbeddings[cachedIdx]
			cachedIdx++
		}
	}

	logger.Debug("Batch processed queries",
		"component", "embedding", "subcomponent",
		"query-cache", "total", len(validTexts),
		"cached", len(cachedEmbeddings), "generated",
		len(newEmbeddings))
	return result, nil
}

// GetUserEmbedding retrieves a user's personalized embedding from the user cache
func GetUserEmbedding(userID int) ([]float32, error) {
	if userEmbeddingCache == nil {
		return nil, fmt.Errorf("user embedding cache not initialized")
	}

	key := fmt.Sprintf("user_%d", userID)
	if cached, found := userEmbeddingCache.Get(key); found {
		logger.Debug("Cache hit for user",
			"component", "embedding", "subcomponent",
			"user-cache", "userID", userID)
		return cached, nil
	}

	logger.Debug("Cache miss for user",
		"component", "embedding", "subcomponent",
		"user-cache", "userID", userID)
	return nil, fmt.Errorf("user embedding not found in cache")
}

// SetUserEmbedding stores a user's personalized embedding in the user cache
func SetUserEmbedding(userID int, embedding []float32) error {
	if userEmbeddingCache == nil {
		return fmt.Errorf("user embedding cache not initialized")
	}

	key := fmt.Sprintf("user_%d", userID)
	// Cache with 24 hour TTL
	userEmbeddingCache.SetWithTTL(key, embedding, int64(len(embedding)*4), 24*time.Hour)
	logger.Debug("Cached embedding for user",
		"component", "embedding", "subcomponent",
		"user-cache", "userID", userID)
	return nil
}

// GetSiteEmbedding retrieves a site-level embedding from the site cache
func GetSiteEmbedding(campaignKey string) ([]float32, error) {
	if siteEmbeddingCache == nil {
		return nil, fmt.Errorf("site embedding cache not initialized")
	}

	key := fmt.Sprintf("site_%s", campaignKey)
	if cached, found := siteEmbeddingCache.Get(key); found {
		logger.Debug("Cache hit for campaign",
			"component", "embedding", "subcomponent",
			"site-cache", "campaignKey", campaignKey)
		return cached, nil
	}

	logger.Debug("Cache miss for campaign, generating embedding",
		"component", "embedding", "subcomponent",
		"site-cache", "campaignKey", campaignKey)
	embedding, err := calculateSiteLevelVector()
	if err != nil {
		return nil, err
	}

	// Cache the result with 6 hour TTL
	siteEmbeddingCache.SetWithTTL(key, embedding, int64(len(embedding)*4), 6*time.Hour)
	logger.Debug("Cached embedding for campaign",
		"component", "embedding", "subcomponent",
		"site-cache", "campaignKey", campaignKey)

	return embedding, nil
}
