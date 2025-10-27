package vector

import (
	"fmt"
	"log"
	"strings"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/search"
)

// GetUserPersonalizedEmbedding loads from cache first, then generates new embedding if not found.
func GetUserPersonalizedEmbedding(userID int, forceRecompute bool) ([]float32, error) {
	log.Printf("[DEBUG] GetUserPersonalizedEmbedding called with userID=%d, forceRecompute=%v", userID, forceRecompute)

	if !forceRecompute {
		// Try cache first
		cached, err := GetUserEmbedding(userID)
		if err == nil && cached != nil {
			log.Printf("[DEBUG] Cache hit for userID=%d", userID)
			return cached, nil
		}

		// Cache miss, will generate new embedding below
		log.Printf("[DEBUG] Cache miss for userID=%d, will generate new embedding", userID)
	}
	log.Printf("[embedding] Calculating personalized user embedding for userID=%d", userID)
	const (
		bookmarkWeight = 3
		clickWeight    = 2
		searchWeight   = 1
		limit          = config.QdrantUserEmbeddingLimit
	)
	log.Printf("[DEBUG] Using limit=%d for userID=%d", limit, userID)

	var vectors [][]float32
	var weights []float32
	bookmarkIDs, err := ad.GetBookmarkedAdIDs(userID)
	if err != nil {
		return nil, fmt.Errorf("fetch bookmarks: %w", err)
	}
	log.Printf("[embedding][debug] userID=%d bookmarks: %v (count=%d)", userID, bookmarkIDs, len(bookmarkIDs))

	// Batch fetch bookmark embeddings
	var bookmarkEmbeddings [][]float32
	if len(bookmarkIDs) > 0 {
		embeddings, err := GetAdEmbeddings(bookmarkIDs)
		if err != nil {
			log.Printf("[embedding][debug] Batch bookmark embedding error: %v", err)
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
			log.Printf("[embedding][debug] Qdrant embedding missing for bookmarked adID=%d", adID)
		}
		if len(vectors) >= limit*bookmarkWeight {
			break
		}
	}

	log.Printf("[DEBUG] About to call GetRecentlyClickedAdIDsByUser for userID=%d with limit=%d", userID, limit)
	clickedIDs, err := ad.GetRecentlyClickedAdIDsByUser(userID, limit)
	if err != nil {
		log.Printf("[DEBUG] GetRecentlyClickedAdIDsByUser error: %v", err)
		return nil, fmt.Errorf("fetch clicks: %w", err)
	}
	log.Printf("[embedding][debug] userID=%d clicked: %v (count=%d)", userID, clickedIDs, len(clickedIDs))

	// Batch fetch click embeddings
	var clickEmbeddings [][]float32
	if len(clickedIDs) > 0 {
		embeddings, err := GetAdEmbeddings(clickedIDs)
		if err != nil {
			log.Printf("[embedding][debug] Batch click embedding error: %v", err)
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
			log.Printf("[embedding][debug] Qdrant embedding missing for clicked adID=%d", adID)
		}
	}
	searches, err := search.GetRecentUserSearches(userID, limit)
	if err != nil {
		return nil, fmt.Errorf("fetch searches: %w", err)
	}
	log.Printf("[embedding][debug] userID=%d recent searches: %v (count=%d)", userID, searches, len(searches))

	// Collect valid search queries for batch processing
	var searchQueries []string
	var validSearches []search.UserSearch
	for _, s := range searches {
		if strings.TrimSpace(s.QueryString) != "" {
			searchQueries = append(searchQueries, s.QueryString)
			validSearches = append(validSearches, s)
		} else {
			log.Printf("[embedding][debug] Skipping empty search query")
		}
	}

	// Generate embeddings for all search queries in batch
	var searchEmbeddings [][]float32
	if len(searchQueries) > 0 {
		log.Printf("[embedding][debug] Generating batch embeddings for %d user search queries", len(searchQueries))
		embeddings, err := GetQueryEmbeddings(searchQueries)
		if err != nil {
			log.Printf("[embedding][debug] Batch embedding error for user searches: %v", err)
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
			log.Printf("[embedding][debug] Added embedding for search query: %s", validSearches[i].QueryString)
		} else {
			log.Printf("[embedding][debug] Failed to generate embedding for search query: %s", validSearches[i].QueryString)
		}
	}

	log.Printf("[embedding][debug] userID=%d vectors aggregated: %d, weights: %d", userID, len(vectors), len(weights))
	if len(vectors) == 0 {
		log.Printf("[embedding][warn] userID=%d: No vectors could be aggregated from user activity", userID)
		return nil, fmt.Errorf("no user activity to aggregate")
	}

	emb := AggregateEmbeddings(vectors, weights)

	// Cache the result for future use
	if err := SetUserEmbedding(userID, emb); err != nil {
		log.Printf("[embedding][warn] failed to cache user embedding for userID=%d: %v", userID, err)
	}

	log.Printf("[embedding][info] userID=%d: Successfully created user embedding from %d vectors (bookmarks: %d, clicks: %d, searches: %d)",
		userID, len(vectors), len(bookmarkIDs), len(clickedIDs), len(searches))
	return emb, nil
}
