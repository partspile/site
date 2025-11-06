package vector

import (
	"fmt"
	"strings"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/search"
)

// GetUserPersonalizedEmbedding loads from cache first, then generates new embedding if not found.
func GetUserPersonalizedEmbedding(userID int, forceRecompute bool) ([]float32, error) {
	logger.Debug("GetUserPersonalizedEmbedding called",
		"userID", userID, "forceRecompute", forceRecompute)

	if !forceRecompute {
		// Try cache first
		cached, err := GetUserEmbedding(userID)
		if err == nil && cached != nil {
			logger.Debug("Cache hit for user embedding",
				"userID", userID)
			return cached, nil
		}

		// Cache miss, will generate new embedding below
		logger.Debug("Cache miss for user embedding, will generate new",
			"userID", userID)
	}
	logger.Info("Calculating personalized user embedding",
		"userID", userID)
	const (
		bookmarkWeight = 3
		clickWeight    = 2
		searchWeight   = 1
		limit          = config.QdrantUserEmbeddingLimit
	)
	logger.Debug("Using limit for user embedding",
		"limit", limit, "userID", userID)

	var vectors [][]float32
	var weights []float32
	bookmarkIDs, err := ad.GetBookmarkedAdIDs(userID)
	if err != nil {
		return nil, fmt.Errorf("fetch bookmarks: %w", err)
	}
	logger.Debug("User bookmarks",
		"userID", userID, "bookmarkIDs", bookmarkIDs, "count",
		len(bookmarkIDs))

	// Batch fetch bookmark embeddings
	var bookmarkEmbeddings [][]float32
	if len(bookmarkIDs) > 0 {
		embeddings, err := GetAdEmbeddings(bookmarkIDs)
		if err != nil {
			logger.Debug("Batch bookmark embedding error",
				"error", err, "userID", userID)
		} else {
			bookmarkEmbeddings = embeddings
		}
	}

	for i, adID := range bookmarkIDs {
		if i < len(bookmarkEmbeddings) && bookmarkEmbeddings[i] != nil {
			emb := bookmarkEmbeddings[i]
			for j := 0; j < bookmarkWeight; j++ {
				vectors = append(vectors, emb)
				weights = append(weights, 1)
			}
		} else {
			logger.Debug("Qdrant embedding missing for bookmarked ad",
				"adID", adID, "userID", userID)
		}
		if len(vectors) >= limit*bookmarkWeight {
			break
		}
	}

	logger.Debug("About to call GetRecentlyClickedAdIDsByUser",
		"userID", userID, "limit", limit)
	clickedIDs, err := ad.GetRecentlyClickedAdIDsByUser(userID, limit)
	if err != nil {
		logger.Error("GetRecentlyClickedAdIDsByUser error",
			"error", err, "userID", userID)
		return nil, fmt.Errorf("fetch clicks: %w", err)
	}
	logger.Debug("User clicked ads",
		"userID", userID, "clickedIDs", clickedIDs, "count",
		len(clickedIDs))

	// Batch fetch click embeddings
	var clickEmbeddings [][]float32
	if len(clickedIDs) > 0 {
		embeddings, err := GetAdEmbeddings(clickedIDs)
		if err != nil {
			logger.Debug("Batch click embedding error",
				"error", err, "userID", userID)
		} else {
			clickEmbeddings = embeddings
		}
	}

	for i, adID := range clickedIDs {
		if i < len(clickEmbeddings) && clickEmbeddings[i] != nil {
			emb := clickEmbeddings[i]
			for j := 0; j < clickWeight; j++ {
				vectors = append(vectors, emb)
				weights = append(weights, 1)
			}
		} else {
			logger.Debug("Qdrant embedding missing for clicked ad",
				"adID", adID, "userID", userID)
		}
	}
	searches, err := search.GetRecentUserSearches(userID, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch searches: %w", err)
	}
	logger.Debug("User recent searches",
		"userID", userID, "searches", searches, "count", len(searches))

	// Collect valid search queries for batch processing
	var searchQueries []string
	var validSearches []search.UserSearch
	for _, s := range searches {
		if strings.TrimSpace(s.QueryString) != "" {
			searchQueries = append(searchQueries, s.QueryString)
			validSearches = append(validSearches, s)
		} else {
			logger.Debug("Skipping empty search query",
				"userID", userID)
		}
	}

	// Generate embeddings for all search queries in batch
	var searchEmbeddings [][]float32
	if len(searchQueries) > 0 {
		logger.Debug("Generating batch embeddings for user search queries",
			"userID", userID, "count", len(searchQueries))
		embeddings, err := GetQueryEmbeddings(searchQueries)
		if err != nil {
			logger.Debug("Batch embedding error for user searches",
				"error", err, "userID", userID)
		} else {
			searchEmbeddings = embeddings
		}
	}

	// Add search embeddings to vectors with appropriate weights
	for i, emb := range searchEmbeddings {
		if emb != nil {
			for j := 0; j < searchWeight; j++ {
				vectors = append(vectors, emb)
				weights = append(weights, 1)
			}
			logger.Debug("Added embedding for search query",
				"userID", userID, "query", validSearches[i].QueryString)
		} else {
			logger.Debug("Failed to generate embedding for search query",
				"userID", userID, "query", validSearches[i].QueryString)
		}
	}

	logger.Debug("Vectors aggregated",
		"userID", userID, "vectors", len(vectors), "weights",
		len(weights))
	if len(vectors) == 0 {
		logger.Warn("No vectors could be aggregated from user activity",
			"userID", userID)
		return nil, fmt.Errorf("no user activity to aggregate")
	}

	emb := AggregateEmbeddings(vectors, weights)

	// Cache the result for future use
	if err := SetUserEmbedding(userID, emb); err != nil {
		logger.Warn("Failed to cache user embedding",
			"error", err, "userID", userID)
	}

	logger.Info("Successfully created user embedding",
		"userID", userID, "vectors", len(vectors), "bookmarks",
		len(bookmarkIDs), "clicks", len(clickedIDs), "searches",
		len(searches))
	return emb, nil
}
