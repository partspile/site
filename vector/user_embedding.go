package vector

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/search"
	"github.com/qdrant/go-client/qdrant"
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
	bookmarkIDs, err := ad.GetBookmarkedAdIDsByUser(userID)
	if err != nil {
		return nil, fmt.Errorf("fetch bookmarks: %w", err)
	}
	log.Printf("[embedding][debug] userID=%d bookmarks: %v (count=%d)", userID, bookmarkIDs, len(bookmarkIDs))
	for _, adID := range bookmarkIDs {
		emb, err := GetAdEmbeddingFromQdrant(adID)
		if err != nil {
			log.Printf("[embedding][debug] Qdrant embedding missing for bookmarked adID=%d: %v", adID, err)
		}
		if err == nil && emb != nil {
			for i := 0; i < bookmarkWeight; i++ {
				vectors = append(vectors, emb)
				weights = append(weights, 1)
			}
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
	for _, adID := range clickedIDs {
		emb, err := GetAdEmbeddingFromQdrant(adID)
		if err != nil {
			log.Printf("[embedding][debug] Qdrant embedding missing for clicked adID=%d: %v", adID, err)
		}
		if err == nil && emb != nil {
			for i := 0; i < clickWeight; i++ {
				vectors = append(vectors, emb)
				weights = append(weights, 1)
			}
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

	// Add a rock preference vector to favor ads with fewer rocks
	rockPreferencePrompt := "Show me high-quality ads with fewer reported issues (rocks thrown). I prefer reliable, trustworthy listings."
	rockPreferenceEmbedding, err := GetQueryEmbedding(rockPreferencePrompt)
	if err == nil && rockPreferenceEmbedding != nil {
		// Add rock preference with high weight to ensure it influences results
		vectors = append(vectors, rockPreferenceEmbedding)
		weights = append(weights, 2.0) // Higher weight for rock preference
		log.Printf("[embedding][debug] userID=%d: Added rock preference vector", userID)
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

// GetAdEmbeddingFromQdrant fetches the embedding for an ad by ID from Qdrant.
func GetAdEmbeddingFromQdrant(adID int) ([]float32, error) {
	if qdrantClient == nil {
		return nil, fmt.Errorf("Qdrant client not initialized")
	}
	log.Printf("[qdrant] Fetching vector for ad %d", adID)
	ctx := context.Background()
	pointID := qdrant.NewIDNum(uint64(adID))
	resp, err := qdrantClient.Get(ctx, &qdrant.GetPoints{
		CollectionName: qdrantCollection,
		Ids:            []*qdrant.PointId{pointID},
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: true}},
	})
	if err != nil {
		log.Printf("[qdrant] Fetch error for ad %d: %v", adID, err)
		return nil, fmt.Errorf("Qdrant fetch error: %w", err)
	}
	log.Printf("[qdrant] Fetch response for ad %d: found %d points", adID, len(resp))
	if len(resp) == 0 {
		log.Printf("[qdrant] No points found for ad %d", adID)
		return nil, fmt.Errorf("vector not found for adID %d", adID)
	}

	point := resp[0]
	log.Printf("[qdrant] Point structure for ad %d: Vectors=%v", adID, point.Vectors != nil)
	if point.Vectors == nil {
		log.Printf("[qdrant] Vector values are nil for ad %d", adID)
		return nil, fmt.Errorf("no vector data for adID %d", adID)
	}

	// Extract vector data
	var vectorData []float32
	if point.Vectors != nil {
		if vectorOutput := point.Vectors.GetVector(); vectorOutput != nil {
			// Try to get the vector data directly from VectorOutput
			if data := vectorOutput.GetData(); len(data) > 0 {
				vectorData = data
			} else if dense := vectorOutput.GetDense(); dense != nil {
				vectorData = dense.Data
			}
		}
	}

	if vectorData == nil {
		log.Printf("[qdrant] No vector data found for ad %d", adID)
		return nil, fmt.Errorf("no vector data for adID %d", adID)
	}

	log.Printf("[qdrant] Successfully retrieved vector for ad %d with length %d", adID, len(vectorData))
	return vectorData, nil
}
