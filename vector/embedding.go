package vector

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/rock"
	"github.com/parts-pile/site/vehicle"
	"github.com/qdrant/go-client/qdrant"
	genai "google.golang.org/genai"
)

// AdResult represents a search result from Qdrant
// TODO: Fill in fields as needed
type AdResult struct {
	ID       int
	Score    float32
	Metadata map[string]interface{}
}

var (
	geminiClient     *genai.Client
	qdrantClient     *qdrant.Client
	qdrantCollection string
	embeddingCache   *cache.Cache[[]float32] // Keep for backward compatibility during migration

	// New specialized caches
	queryEmbeddingCache *cache.Cache[[]float32] // String keys, 1 hour TTL
	userEmbeddingCache  *cache.Cache[[]float32] // User ID keys, 24 hour TTL
	siteEmbeddingCache  *cache.Cache[[]float32] // Campaign keys, 6 hour TTL
)

// InitEmbeddingCache initializes the embedding cache. This should be called during application startup.
// TODO: This function will be renamed to InitEmbeddingCaches in Phase 7
func InitEmbeddingCache() error {
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

	// Keep existing embeddingCache initialization for backward compatibility during migration
	embeddingCache, err = cache.New[[]float32](func(value []float32) int64 {
		return int64(len(value) * 4) // 4 bytes per float32
	}, "Embedding Query Cache")
	if err != nil {
		return fmt.Errorf("failed to initialize legacy embedding cache: %w", err)
	}

	return nil
}

// GetEmbeddingCacheStats returns cache statistics for admin monitoring
func GetEmbeddingCacheStats() map[string]interface{} {
	return embeddingCache.Stats()
}

// ClearEmbeddingCache clears all cached embeddings
func ClearEmbeddingCache() {
	embeddingCache.Clear()
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
		log.Printf("[embedding][query-cache] Cache hit for query: %.80q", text)
		return cached, nil
	}

	// Generate embedding
	embedding, err := EmbedText(text)
	if err != nil {
		return nil, err
	}

	// Cache the result with 1 hour TTL
	queryEmbeddingCache.SetWithTTL(text, embedding, int64(len(embedding)*4), time.Hour)
	log.Printf("[embedding][query-cache] Cached embedding for query: %.80q", text)

	return embedding, nil
}

// GetUserEmbedding retrieves a user's personalized embedding from the user cache
func GetUserEmbedding(userID int) ([]float32, error) {
	if userEmbeddingCache == nil {
		return nil, fmt.Errorf("user embedding cache not initialized")
	}

	key := fmt.Sprintf("user_%d", userID)
	if cached, found := userEmbeddingCache.Get(key); found {
		log.Printf("[embedding][user-cache] Cache hit for user %d", userID)
		return cached, nil
	}

	log.Printf("[embedding][user-cache] Cache miss for user %d", userID)
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
	log.Printf("[embedding][user-cache] Cached embedding for user %d", userID)
	return nil
}

// GetSiteEmbedding retrieves a site-level embedding from the site cache
func GetSiteEmbedding(campaignKey string) ([]float32, error) {
	if siteEmbeddingCache == nil {
		return nil, fmt.Errorf("site embedding cache not initialized")
	}

	key := fmt.Sprintf("site_%s", campaignKey)
	if cached, found := siteEmbeddingCache.Get(key); found {
		log.Printf("[embedding][site-cache] Cache hit for campaign %s", campaignKey)
		return cached, nil
	}

	log.Printf("[embedding][site-cache] Cache miss for campaign %s, generating embedding", campaignKey)
	embedding, err := calculateSiteLevelVector()
	if err != nil {
		return nil, err
	}

	// Cache the result with 6 hour TTL
	siteEmbeddingCache.SetWithTTL(key, embedding, int64(len(embedding)*4), 6*time.Hour)
	log.Printf("[embedding][site-cache] Cached embedding for campaign %s", campaignKey)

	return embedding, nil
}

// EmbedTextCached generates an embedding for the given text using Gemini, with caching
// TODO: This function now uses the new query cache internally but maintains backward compatibility
func EmbedTextCached(text string) ([]float32, error) {
	// Use the new query cache for better performance and TTL management
	return GetQueryEmbedding(text)
}

// InitGeminiClient initializes the Gemini embedding client
func InitGeminiClient() error {
	apiKey := config.GeminiAPIKey
	if apiKey == "" {
		return fmt.Errorf("missing Gemini API key")
	}
	// The client gets the API key from the environment variable `GEMINI_API_KEY`
	client, err := genai.NewClient(context.Background(), nil)
	if err != nil {
		return err
	}
	geminiClient = client
	return nil
}

// InitQdrantClient initializes the Qdrant client and collection
func InitQdrantClient() error {
	host := config.QdrantHost
	if host == "" {
		return fmt.Errorf("missing Qdrant host")
	}
	apiKey := config.QdrantAPIKey
	collection := config.QdrantCollection
	if collection == "" {
		return fmt.Errorf("missing Qdrant collection name")
	}

	log.Printf("[qdrant] Initializing client with host: %s, collection: %s", host, collection)

	// Create Qdrant client configuration
	clientConfig := &qdrant.Config{
		APIKey:                 apiKey,
		UseTLS:                 true, // Qdrant Cloud requires TLS
		SkipCompatibilityCheck: true, // Skip version check for cloud service
	}
	log.Printf("[qdrant] Client config - Host: %s, Port: %d, UseTLS: %v", clientConfig.Host, clientConfig.Port, clientConfig.UseTLS)
	if host != "" {
		// For Qdrant Cloud, the host should be just the hostname without protocol
		// Remove any protocol prefix if present
		if strings.HasPrefix(host, "https://") {
			host = strings.TrimPrefix(host, "https://")
		} else if strings.HasPrefix(host, "http://") {
			host = strings.TrimPrefix(host, "http://")
		}
		clientConfig.Host = host
		clientConfig.Port = config.QdrantPort // Qdrant Cloud uses port 6334 for gRPC
	}

	// Create Qdrant client
	client, err := qdrant.NewClient(clientConfig)
	if err != nil {
		return err
	}
	qdrantClient = client
	qdrantCollection = collection

	// Check if collection exists, create if not
	ctx := context.Background()

	// Try to list collections first to test connection
	collections, err := client.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to connect to Qdrant: %w", err)
	}

	collectionExists := false
	for _, col := range collections {
		if col == collection {
			collectionExists = true
			break
		}
	}

	if !collectionExists {
		log.Printf("[qdrant] Creating collection: %s", collection)
		err = client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collection,
			VectorsConfig: &qdrant.VectorsConfig{
				Config: &qdrant.VectorsConfig_Params{
					Params: &qdrant.VectorParams{
						Size:     768, // Gemini embedding size
						Distance: qdrant.Distance_Cosine,
					},
				},
			},
		})
		if err != nil {
			return fmt.Errorf("failed to create collection: %w", err)
		}
		log.Printf("[qdrant] Successfully created collection: %s", collection)
	} else {
		log.Printf("[qdrant] Collection already exists: %s", collection)
	}

	return nil
}

// EmbedText generates an embedding for the given text using Gemini
func EmbedText(text string) ([]float32, error) {
	if geminiClient == nil {
		return nil, fmt.Errorf("Gemini client not initialized")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("cannot embed empty text")
	}
	log.Printf("[embedding] Calculating embedding vector for text: %.80q", text)
	ctx := context.Background()
	resp, err := geminiClient.Models.EmbedContent(ctx, config.GeminiEmbeddingModel, genai.Text(text), nil)
	if err != nil {
		return nil, fmt.Errorf("Gemini embedding API error: %w", err)
	}
	if resp == nil || len(resp.Embeddings) == 0 || resp.Embeddings[0] == nil {
		return nil, fmt.Errorf("no embedding returned from Gemini API")
	}
	return resp.Embeddings[0].Values, nil
}

// UpsertAdEmbedding upserts an ad's embedding and metadata into Qdrant
func UpsertAdEmbedding(adID int, embedding []float32, metadata map[string]interface{}) error {
	if qdrantClient == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}
	log.Printf("[qdrant] Upserting vector for ad %d", adID)

	// Convert metadata to Qdrant format
	var qdrantMetadata map[string]*qdrant.Value
	if metadata != nil {
		qdrantMetadata = make(map[string]*qdrant.Value)
		for k, v := range metadata {
			switch val := v.(type) {
			case string:
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: val}}
			case int:
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(val)}}
			case float64:
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: val}}
			case bool:
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_BoolValue{BoolValue: val}}
			case []string:
				// Handle array of strings (years, models, engines)
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_ListValue{ListValue: &qdrant.ListValue{
					Values: func() []*qdrant.Value {
						result := make([]*qdrant.Value, len(val))
						for i, s := range val {
							result[i] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: s}}
						}
						return result
					}(),
				}}}
			case map[string]interface{}:
				// Handle geo metadata (lat/lon coordinates)
				if k == "location" {
					geoStruct := &qdrant.Struct{Fields: make(map[string]*qdrant.Value)}
					for geoKey, geoVal := range val {
						if geoFloat, ok := geoVal.(float64); ok {
							geoStruct.Fields[geoKey] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: geoFloat}}
						}
					}
					qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_StructValue{StructValue: geoStruct}}
				} else {
					// Convert other maps to string
					qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", val)}}
				}
			default:
				// Convert to string for other types
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", val)}}
			}
		}
		log.Printf("[qdrant] Metadata prepared for ad %d", adID)
	}

	// Create Qdrant point with numeric ID instead of UUID
	point := &qdrant.PointStruct{
		Id: &qdrant.PointId{
			PointIdOptions: &qdrant.PointId_Num{Num: uint64(adID)},
		},
		Vectors: &qdrant.Vectors{
			VectorsOptions: &qdrant.Vectors_Vector{
				Vector: &qdrant.Vector{
					Vector: &qdrant.Vector_Dense{
						Dense: &qdrant.DenseVector{
							Data: embedding,
						},
					},
				},
			},
		},
		Payload: qdrantMetadata,
	}

	log.Printf("[qdrant] Upserting vector for ad %d", adID)

	ctx := context.Background()
	_, err := qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: qdrantCollection,
		Points:         []*qdrant.PointStruct{point},
	})
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	log.Printf("[qdrant] Successfully upserted vector for ad %d", adID)
	return nil
}

// DeleteAdEmbedding deletes an ad's embedding from Qdrant
func DeleteAdEmbedding(adID int) error {
	if qdrantClient == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}
	log.Printf("[qdrant] Deleting vector for ad %d", adID)

	ctx := context.Background()
	pointID := qdrant.NewIDNum(uint64(adID))
	_, err := qdrantClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: qdrantCollection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{pointID},
				},
			},
		},
	})
	if err != nil {
		log.Printf("[qdrant] Failed to delete vector for ad %d: %v", adID, err)
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	log.Printf("[qdrant] Successfully deleted vector for ad %d", adID)
	return nil
}

// QuerySimilarAdIDs queries Qdrant for similar ad IDs given an embedding
// Returns a list of ad IDs, and a cursor for pagination
// If filter is provided, it will be applied to the search
func QuerySimilarAdIDs(embedding []float32, filter *qdrant.Filter, topK int, cursor string, threshold float64) ([]int, string, error) {
	if qdrantClient == nil {
		return nil, "", fmt.Errorf("Qdrant client not initialized")
	}
	ctx := context.Background()

	// Parse cursor if provided
	var offset uint64 = 0
	if cursor != "" {
		// Decode cursor: format is "offset" base64 encoded
		cursorBytes, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			cursorStr := string(cursorBytes)
			if offsetVal, err := strconv.ParseUint(cursorStr, 10, 64); err == nil {
				offset = offsetVal
			}
		}
		log.Printf("[qdrant] Parsed cursor: offset=%d", offset)
	} else {
		log.Printf("[qdrant] No cursor provided, starting from beginning")
	}

	// Create search request using Query method
	limit := uint64(topK)
	scoreThreshold := float32(threshold)
	queryRequest := &qdrant.QueryPoints{
		CollectionName: qdrantCollection,
		Query:          qdrant.NewQueryDense(embedding),
		Filter:         filter,
		Limit:          &limit,
		Offset:         &offset,
		ScoreThreshold: &scoreThreshold,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
		},
		WithVectors: &qdrant.WithVectorsSelector{
			SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: false},
		},
	}

	resp, err := qdrantClient.Query(ctx, queryRequest)
	if err != nil {
		if filter != nil {
			return nil, "", fmt.Errorf("failed to query Qdrant with filter: %w", err)
		}
		return nil, "", fmt.Errorf("failed to query Qdrant: %w", err)
	}
	log.Printf("[qdrant] Query returned %d results (requested %d, offset %d)", len(resp), topK, offset)

	var results []int
	for _, result := range resp {
		// Extract ID
		adID := int(result.Id.GetNum())
		results = append(results, adID)
		log.Printf("[qdrant] Added result with ID: %d, Score: %f", adID, result.Score)
	}

	// Generate next cursor if we have results
	var nextCursor string
	if len(results) > 0 {
		nextOffset := offset + uint64(len(results))
		nextCursor = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", nextOffset)))
		log.Printf("[qdrant] Generated next cursor: %s (offset: %d)", nextCursor, nextOffset)
	} else {
		log.Printf("[qdrant] No results, no next cursor generated")
	}

	return results, nextCursor, nil
}

// QuerySimilarAds queries Qdrant for similar ads given an embedding
// Returns a list of AdResult, and a cursor for pagination
func QuerySimilarAds(embedding []float32, topK int, cursor string, threshold float64) ([]AdResult, string, error) {
	if qdrantClient == nil {
		return nil, "", fmt.Errorf("Qdrant client not initialized")
	}
	ctx := context.Background()

	// Parse cursor if provided
	var offset uint64 = 0
	if cursor != "" {
		// Decode cursor: format is "offset" base64 encoded
		cursorBytes, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			cursorStr := string(cursorBytes)
			if offsetVal, err := strconv.ParseUint(cursorStr, 10, 64); err == nil {
				offset = offsetVal
			}
		}
		log.Printf("[qdrant] Parsed cursor: offset=%d", offset)
	} else {
		log.Printf("[qdrant] No cursor provided, starting from beginning")
	}

	// Create search request using Query method
	limit := uint64(topK)
	scoreThreshold := float32(threshold)
	queryRequest := &qdrant.QueryPoints{
		CollectionName: qdrantCollection,
		Query:          qdrant.NewQueryDense(embedding),
		Limit:          &limit,
		Offset:         &offset,
		ScoreThreshold: &scoreThreshold,
		WithPayload: &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
		},
		WithVectors: &qdrant.WithVectorsSelector{
			SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: false},
		},
	}

	resp, err := qdrantClient.Query(ctx, queryRequest)
	if err != nil {
		return nil, "", fmt.Errorf("failed to query Qdrant: %w", err)
	}
	log.Printf("[qdrant] Query returned %d results (requested %d, offset %d)", len(resp), topK, offset)

	var results []AdResult
	for _, result := range resp {
		// Convert payload to map[string]interface{}
		metadata := make(map[string]interface{})
		for k, v := range result.Payload {
			switch val := v.Kind.(type) {
			case *qdrant.Value_StringValue:
				metadata[k] = val.StringValue
			case *qdrant.Value_IntegerValue:
				metadata[k] = val.IntegerValue
			case *qdrant.Value_DoubleValue:
				metadata[k] = val.DoubleValue
			case *qdrant.Value_BoolValue:
				metadata[k] = val.BoolValue
			default:
				metadata[k] = fmt.Sprintf("%v", val)
			}
		}

		// Extract ID - we stored with numeric IDs, so we need to get the numeric value
		var adID int
		if numID := result.Id.GetNum(); numID != 0 {
			adID = int(numID)
		} else {
			// Fallback to string ID if somehow we get a UUID
			if uuidStr := result.Id.GetUuid(); uuidStr != "" {
				// Try to parse as int if it's numeric
				if parsedID, err := strconv.Atoi(uuidStr); err == nil {
					adID = parsedID
				} else {
					adID = 0 // Invalid ID
				}
			}
		}

		adResult := AdResult{
			ID:       adID,
			Score:    float32(result.Score),
			Metadata: metadata,
		}
		results = append(results, adResult)
		log.Printf("[qdrant] Added result with ID: %d, Score: %f", adID, result.Score)
	}

	// Generate next cursor if we have results
	var nextCursor string
	if len(results) > 0 {
		nextOffset := offset + uint64(len(results))
		nextCursor = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", nextOffset)))
		log.Printf("[qdrant] Generated next cursor: %s (offset: %d)", nextCursor, nextOffset)
	} else {
		log.Printf("[qdrant] No results, no next cursor generated")
	}

	return results, nextCursor, nil
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
			log.Printf("[site-level]   %d. Ad %d: %s (clicks: %d)",
				i+1, adObj.ID, adObj.Title, adObj.ClickCount)
		}
	}
	if len(ads) > 5 {
		log.Printf("[site-level]   ... and %d more ads", len(ads)-5)
	}
	var vectors [][]float32
	var missingIDs []int
	for _, adObj := range ads {
		log.Printf("[site-level] Fetching existing embedding for ad %d (title: %s)", adObj.ID, adObj.Title)
		emb, err := GetAdEmbedding(adObj.ID)
		if err != nil || emb == nil {
			missingIDs = append(missingIDs, adObj.ID)
			log.Printf("[site-level] Missing embedding for ad %d: %v", adObj.ID, err)
			continue
		}
		log.Printf("[site-level] Found embedding for ad %d (length=%d)", adObj.ID, len(emb))
		vectors = append(vectors, emb)
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
	rockPreferenceEmbedding, err := EmbedTextCached(rockPreferencePrompt)
	if err == nil && rockPreferenceEmbedding != nil {
		// Blend the site-level vector with rock preference
		enhancedVectors := [][]float32{result, rockPreferenceEmbedding}
		enhancedWeights := []float32{1.0, 1.5} // Site-level gets weight 1.0, rock preference gets 1.5
		result = AggregateEmbeddings(enhancedVectors, enhancedWeights)
		log.Printf("[site-level] Enhanced site-level vector with rock preference")
	}

	return result, nil
}

// GetAdEmbedding retrieves the embedding for a given ad ID from Qdrant
func GetAdEmbedding(adID int) ([]float32, error) {
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
		return nil, err
	}

	log.Printf("[qdrant] Fetch response for ad %d: found %d points", adID, len(resp))
	if len(resp) == 0 {
		log.Printf("[qdrant] No points found for ad %d", adID)
		return nil, fmt.Errorf("no embedding found for ad %d", adID)
	}

	point := resp[0]
	log.Printf("[qdrant] Point structure for ad %d: Vectors=%v", adID, point.Vectors != nil)

	if point.Vectors == nil {
		log.Printf("[qdrant] Vector values are nil for ad %d", adID)
		return nil, fmt.Errorf("no embedding found for ad %d", adID)
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
		return nil, fmt.Errorf("no embedding found for ad %d", adID)
	}

	log.Printf("[qdrant] Successfully retrieved vector for ad %d with length %d", adID, len(vectorData))
	return vectorData, nil
}

// BuildAdEmbedding builds and stores an embedding for a single ad
func BuildAdEmbedding(adObj ad.Ad) error {
	log.Printf("[BuildAdEmbedding] Building embedding for ad %d: %s", adObj.ID, adObj.Title)

	// Build the prompt for embedding
	prompt := buildAdEmbeddingPrompt(adObj)

	// Generate embedding
	embedding, err := EmbedText(prompt)
	if err != nil {
		log.Printf("[BuildAdEmbedding] Failed to generate embedding for ad %d: %v", adObj.ID, err)
		return err
	}

	// Build metadata
	meta := BuildAdEmbeddingMetadata(adObj)

	// Store in Qdrant
	err = UpsertAdEmbedding(adObj.ID, embedding, meta)
	if err != nil {
		log.Printf("[BuildAdEmbedding] Failed to store embedding for ad %d: %v", adObj.ID, err)
		return err
	}

	// Mark ad as having vector in database
	err = ad.MarkAdAsHavingVector(adObj.ID)
	if err != nil {
		log.Printf("[BuildAdEmbedding] Failed to mark ad %d as having vector: %v", adObj.ID, err)
		return err
	}

	log.Printf("[BuildAdEmbedding] Successfully built and stored embedding for ad %d", adObj.ID)
	return nil
}

// buildAdEmbeddingPrompt creates a prompt for generating embeddings
func buildAdEmbeddingPrompt(adObj ad.Ad) string {
	// Get parent company information for the make
	var parentCompanyStr, parentCompanyCountry string
	if adObj.Make != "" {
		if pcInfo, err := vehicle.GetParentCompanyInfoForMake(adObj.Make); err == nil && pcInfo != nil {
			parentCompanyStr = pcInfo.Name
			parentCompanyCountry = pcInfo.Country
		}
	}

	// Get rock count for this ad
	rockCount := 0
	if count, err := rock.GetAdRockCount(adObj.ID); err == nil {
		rockCount = count
	}

	// Include rock count in the embedding - ads with fewer rocks should rank higher
	rockContext := ""
	if rockCount == 0 {
		rockContext = "This ad has no reported issues (0 rocks thrown)."
	} else if rockCount == 1 {
		rockContext = "This ad has 1 reported issue (1 rock thrown)."
	} else {
		rockContext = fmt.Sprintf("This ad has %d reported issues (%d rocks thrown).", rockCount, rockCount)
	}

	return fmt.Sprintf(`Encode the following ad for semantic search. Focus on what the part is, what vehicles it fits, and any relevant details for a buyer. Return only the embedding vector.\n\nTitle: %s\nDescription: %s\nMake: %s\nParent Company: %s\nParent Company Country: %s\nYears: %s\nModels: %s\nEngines: %s\nCategory: %s\nLocation: %s, %s, %s\nQuality Indicator: %s`,
		adObj.Title,
		adObj.Description,
		adObj.Make,
		parentCompanyStr,
		parentCompanyCountry,
		joinStrings(adObj.Years),
		joinStrings(adObj.Models),
		joinStrings(adObj.Engines),
		adObj.Category,
		adObj.City,
		adObj.AdminArea,
		adObj.Country,
		rockContext,
	)
}

// BuildAdEmbeddingMetadata creates metadata for embeddings
func BuildAdEmbeddingMetadata(adObj ad.Ad) map[string]interface{} {
	// Get location data for geo filtering
	var lat, lon float64
	if adObj.LocationID != 0 {
		// Get coordinates from Location table
		_, _, _, _, lat, lon, _ = ad.GetLocation(adObj.LocationID)
	}

	// Get tree path data for navigation filtering
	var make string
	var years, models, engines []string

	// Use the vehicle data already populated in the ad object
	make = adObj.Make
	years = adObj.Years
	models = adObj.Models
	engines = adObj.Engines

	// Get category data for filtering
	var category, subcategory string
	if adObj.SubCategoryID != 0 {
		// Since SubCategory field no longer exists, we'll need to look up the name
		// For now, just use the category if available
		category = adObj.Category
		// TODO: Look up subcategory name from SubCategoryID if needed
	}

	// Get rock count for this ad
	rockCount := 0
	if count, err := rock.GetAdRockCount(adObj.ID); err == nil {
		rockCount = count
	}

	metadata := map[string]interface{}{
		// Tree navigation (string values for filtering)
		"make":        make,
		"years":       years,
		"models":      models,
		"engines":     engines,
		"category":    category,
		"subcategory": subcategory,

		// Price for filtering/sorting
		"price": adObj.Price,

		// Rock count for quality-based ranking
		"rock_count": rockCount,
	}

	// Add geo payload if we have coordinates
	if lat != 0 && lon != 0 {
		metadata["location"] = map[string]interface{}{
			"lat": lat,
			"lon": lon,
		}
	}

	return metadata
}

// Helper function for embedding generation
func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return fmt.Sprintf("%s", ss)
}

// Helper function to check if slice contains string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// geocodeLocation converts location text to lat/lon coordinates
func geocodeLocation(locationText string) (float64, float64) {
	// TODO: Implement geocoding service
	// For now, return default coordinates
	return 0, 0
}

// QuerySimilarAdsWithFilter queries Qdrant with filters
func QuerySimilarAdsWithFilter(embedding []float32, filter *qdrant.Filter, topK int, cursor string, threshold float64) ([]AdResult, string, error) {
	if qdrantClient == nil {
		return nil, "", fmt.Errorf("Qdrant client not initialized")
	}

	ctx := context.Background()

	// Parse cursor if provided
	var offset uint64 = 0
	if cursor != "" {
		cursorBytes, err := base64.StdEncoding.DecodeString(cursor)
		if err == nil {
			cursorStr := string(cursorBytes)
			if offsetVal, err := strconv.ParseUint(cursorStr, 10, 64); err == nil {
				offset = offsetVal
			}
		}
	}

	limit := uint64(topK)

	// Always use similarity threshold for vector search
	scoreThreshold := float32(threshold)
	queryRequest := &qdrant.QueryPoints{
		CollectionName: qdrantCollection,
		Query:          qdrant.NewQueryDense(embedding),
		Filter:         filter,
		Limit:          &limit,
		Offset:         &offset,
		ScoreThreshold: &scoreThreshold,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: false}},
	}

	resp, err := qdrantClient.Query(ctx, queryRequest)
	if err != nil {
		return nil, "", fmt.Errorf("failed to query Qdrant: %w", err)
	}

	var results []AdResult
	for _, result := range resp {
		metadata := make(map[string]interface{})
		for k, v := range result.Payload {
			switch val := v.Kind.(type) {
			case *qdrant.Value_StringValue:
				metadata[k] = val.StringValue
			case *qdrant.Value_IntegerValue:
				metadata[k] = val.IntegerValue
			case *qdrant.Value_DoubleValue:
				metadata[k] = val.DoubleValue
			case *qdrant.Value_BoolValue:
				metadata[k] = val.BoolValue
			default:
				metadata[k] = fmt.Sprintf("%v", val)
			}
		}

		var adID int
		if numID := result.Id.GetNum(); numID != 0 {
			adID = int(numID)
		} else {
			// Fallback to string ID if somehow we get a UUID
			if uuidStr := result.Id.GetUuid(); uuidStr != "" {
				// Try to parse as int if it's numeric
				if parsedID, err := strconv.Atoi(uuidStr); err == nil {
					adID = parsedID
				} else {
					adID = 0 // Invalid ID
				}
			}
		}

		adResult := AdResult{
			ID:       adID,
			Score:    float32(result.Score),
			Metadata: metadata,
		}
		results = append(results, adResult)
	}

	// Generate next cursor
	var nextCursor string
	if len(results) > 0 {
		nextOffset := offset + uint64(len(results))
		nextCursor = base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", nextOffset)))
	}

	return results, nextCursor, nil
}

// BuildTreeFilter creates a filter for tree navigation
func BuildTreeFilter(treePath map[string]string) *qdrant.Filter {
	var conditions []*qdrant.Condition

	if make, ok := treePath["make"]; ok && make != "" {
		// URL decode the make value
		decodedMake, err := url.QueryUnescape(make)
		if err != nil {
			decodedMake = make // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "make",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: decodedMake}},
				},
			},
		})
	}

	if year, ok := treePath["year"]; ok && year != "" {
		// URL decode the year value
		decodedYear, err := url.QueryUnescape(year)
		if err != nil {
			decodedYear = year // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "years",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keywords{Keywords: &qdrant.RepeatedStrings{Strings: []string{decodedYear}}}},
				},
			},
		})
	}

	if model, ok := treePath["model"]; ok && model != "" {
		// URL decode the model value
		decodedModel, err := url.QueryUnescape(model)
		if err != nil {
			decodedModel = model // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "models",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keywords{Keywords: &qdrant.RepeatedStrings{Strings: []string{decodedModel}}}},
				},
			},
		})
	}

	if engine, ok := treePath["engine"]; ok && engine != "" {
		// URL decode the engine value
		decodedEngine, err := url.QueryUnescape(engine)
		if err != nil {
			decodedEngine = engine // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "engines",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keywords{Keywords: &qdrant.RepeatedStrings{Strings: []string{decodedEngine}}}},
				},
			},
		})
	}

	if category, ok := treePath["category"]; ok && category != "" {
		// URL decode the category value
		decodedCategory, err := url.QueryUnescape(category)
		if err != nil {
			decodedCategory = category // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "category",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: decodedCategory}},
				},
			},
		})
	}

	if subcategory, ok := treePath["subcategory"]; ok && subcategory != "" {
		// URL decode the subcategory value
		decodedSubCategory, err := url.QueryUnescape(subcategory)
		if err != nil {
			decodedSubCategory = subcategory // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "subcategory",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: decodedSubCategory}},
				},
			},
		})
	}

	if len(conditions) == 0 {
		return nil
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}

// BuildGeoFilter creates a geo filter for location-based search
// Note: This is a placeholder - geo filtering may require different Qdrant API calls
func BuildGeoFilter(lat, lon float64, radiusMeters float64) *qdrant.Filter {
	// TODO: Implement proper geo filtering when we understand the Qdrant API
	log.Printf("[vector] Geo filtering not yet implemented")
	return nil
}

// BuildBoundingBoxGeoFilter creates a geo filter for bounding box search
func BuildBoundingBoxGeoFilter(minLat, maxLat, minLon, maxLon float64) *qdrant.Filter {
	log.Printf("[vector] Building bounding box filter: lat[%.6f,%.6f], lon[%.6f,%.6f]", minLat, maxLat, minLon, maxLon)

	// Create geo bounding box filter using Qdrant's native geo filtering
	// Note: The order is topLeft.lat, topLeft.lon, bottomRight.lat, bottomRight.lon
	// topLeft = maxLat, minLon (northwest corner)
	// bottomRight = minLat, maxLon (southeast corner)
	geoCondition := qdrant.NewGeoBoundingBox("location", maxLat, minLon, minLat, maxLon)

	conditions := []*qdrant.Condition{
		geoCondition,
	}

	// Create filter with conditions
	filter := &qdrant.Filter{
		Must: conditions,
	}

	log.Printf("[vector] Created Qdrant geo bounding box filter")
	return filter
}
