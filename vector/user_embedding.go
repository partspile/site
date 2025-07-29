package vector

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"strings"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/search"
	"github.com/qdrant/go-client/qdrant"
)

// LoadUserEmbeddingFromDB loads the user's embedding from the UserEmbedding table.
func LoadUserEmbeddingFromDB(userID int) ([]float32, error) {
	row := db.QueryRow(`SELECT embedding FROM UserEmbedding WHERE user_id = ?`, userID)
	var blob []byte
	err := row.Scan(&blob)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if len(blob)%4 != 0 {
		return nil, fmt.Errorf("invalid embedding blob length")
	}
	vec := make([]float32, len(blob)/4)
	for i := range vec {
		vec[i] = math32frombytes(blob[i*4 : (i+1)*4])
	}
	return vec, nil
}

// SaveUserEmbeddingToDB upserts the user's embedding into the UserEmbedding table.
func SaveUserEmbeddingToDB(userID int, embedding []float32) error {
	if len(embedding) == 0 {
		return fmt.Errorf("embedding is empty")
	}
	blob := make([]byte, 4*len(embedding))
	for i, v := range embedding {
		binary.LittleEndian.PutUint32(blob[i*4:(i+1)*4], math32tobytes(v))
	}
	_, err := db.Exec(`INSERT INTO UserEmbedding (user_id, embedding, updated_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET embedding=excluded.embedding, updated_at=CURRENT_TIMESTAMP`, userID, blob)
	if err == nil {
		log.Printf("[embedding][info] Saved user embedding for userID=%d", userID)
	}
	return err
}

// math32frombytes converts 4 bytes to float32
func math32frombytes(b []byte) float32 {
	return float32(binary.LittleEndian.Uint32(b))
}

// math32tobytes converts float32 to uint32 for storage
func math32tobytes(f float32) uint32 {
	return math.Float32bits(f)
}

// GetUserPersonalizedEmbedding loads from DB unless forceRecompute is true or not found.
func GetUserPersonalizedEmbedding(userID int, forceRecompute bool) ([]float32, error) {
	if !forceRecompute {
		emb, err := LoadUserEmbeddingFromDB(userID)
		if err != nil {
			return nil, err
		}
		if emb != nil {
			return emb, nil
		}
	}
	log.Printf("[embedding] Calculating personalized user embedding for userID=%d", userID)
	const (
		bookmarkWeight = 3
		clickWeight    = 2
		searchWeight   = 1
		limit          = 10
	)
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
	clickedIDs, err := ad.GetRecentlyClickedAdIDsByUser(userID, limit)
	if err != nil {
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
	for _, s := range searches {
		if strings.TrimSpace(s.QueryString) == "" {
			log.Printf("[embedding][debug] Skipping empty search query")
			continue
		}
		log.Printf("[embedding][debug] Generating embedding for user search query: %s", s.QueryString)
		emb, err := EmbedText(s.QueryString)
		if err != nil {
			log.Printf("[embedding][debug] Gemini embedding error for query=%q: %v", s.QueryString, err)
		}
		if err == nil && emb != nil {
			for i := 0; i < searchWeight; i++ {
				vectors = append(vectors, emb)
				weights = append(weights, 1)
			}
		}
	}
	log.Printf("[embedding][debug] userID=%d vectors aggregated: %d, weights: %d", userID, len(vectors), len(weights))
	if len(vectors) == 0 {
		log.Printf("[embedding][warn] userID=%d: No vectors could be aggregated from user activity", userID)
		return nil, fmt.Errorf("no user activity to aggregate")
	}
	emb := AggregateEmbeddings(vectors, weights)
	err = SaveUserEmbeddingToDB(userID, emb)
	if err != nil {
		log.Printf("[embedding][warn] failed to save user embedding for userID=%d: %v", userID, err)
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
