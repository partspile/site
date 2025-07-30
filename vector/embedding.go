package vector

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/vehicle"
	"github.com/qdrant/go-client/qdrant"
	genai "google.golang.org/genai"
)

// AdResult represents a search result from Qdrant
// TODO: Fill in fields as needed
type AdResult struct {
	ID       string
	Score    float32
	Metadata map[string]interface{}
}

var (
	geminiClient     *genai.Client
	qdrantClient     *qdrant.Client
	qdrantCollection string
)

// InitGeminiClient initializes the Gemini embedding client
func InitGeminiClient(apiKey string) error {
	if apiKey == "" {
		apiKey = config.GeminiAPIKey
	}
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
func InitQdrantClient(host, apiKey, collection string) error {
	if host == "" {
		host = config.QdrantHost
	}
	if host == "" {
		return fmt.Errorf("missing Qdrant host")
	}
	if apiKey == "" {
		apiKey = config.QdrantAPIKey
	}
	if collection == "" {
		collection = config.QdrantCollection
	}
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
		clientConfig.Port = 6334 // Qdrant Cloud uses port 6334 for gRPC
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
	resp, err := geminiClient.Models.EmbedContent(ctx, "embedding-001", genai.Text(text), nil)
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

// QuerySimilarAds queries Qdrant for similar ads given an embedding
// Returns a list of AdResult, and a cursor for pagination
func QuerySimilarAds(embedding []float32, topK int, cursor string) ([]AdResult, string, error) {
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
	scoreThreshold := float32(config.VectorSearchThreshold)
	queryRequest := &qdrant.QueryPoints{
		CollectionName: qdrantCollection,
		Query:          qdrant.NewQueryDense(embedding),
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
		var adID string
		if numID := result.Id.GetNum(); numID != 0 {
			adID = fmt.Sprintf("%d", numID)
		} else {
			adID = result.Id.GetUuid()
		}

		adResult := AdResult{
			ID:       adID,
			Score:    float32(result.Score),
			Metadata: metadata,
		}
		results = append(results, adResult)
		log.Printf("[qdrant] Added result with ID: %s, Score: %f", adID, result.Score)
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

// Site-level vector for anonymous default feed
var (
	siteLevelVector         []float32
	siteLevelVectorLastCalc time.Time
	siteLevelVectorTTL      = 10 * time.Minute
)

// GetSiteLevelVector returns the cached site-level vector, recalculating if needed
func GetSiteLevelVector() ([]float32, error) {
	if siteLevelVector != nil && time.Since(siteLevelVectorLastCalc) < siteLevelVectorTTL {
		return siteLevelVector, nil
	}
	vec, err := CalculateSiteLevelVector()
	if err != nil {
		return nil, err
	}
	siteLevelVector = vec
	siteLevelVectorLastCalc = time.Now()
	return vec, nil
}

// CalculateSiteLevelVector averages the embeddings of the most popular ads
func CalculateSiteLevelVector() ([]float32, error) {
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
	meta := buildAdEmbeddingMetadata(adObj)

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

	return fmt.Sprintf(`Encode the following ad for semantic search. Focus on what the part is, what vehicles it fits, and any relevant details for a buyer. Return only the embedding vector.\n\nTitle: %s\nDescription: %s\nMake: %s\nParent Company: %s\nParent Company Country: %s\nYears: %s\nModels: %s\nEngines: %s\nCategory: %s\nSubCategory: %s\nLocation: %s, %s, %s`,
		adObj.Title,
		adObj.Description,
		adObj.Make,
		parentCompanyStr,
		parentCompanyCountry,
		joinStrings(adObj.Years),
		joinStrings(adObj.Models),
		joinStrings(adObj.Engines),
		adObj.Category,
		adObj.SubCategory,
		adObj.City,
		adObj.AdminArea,
		adObj.Country,
	)
}

// buildAdEmbeddingMetadata creates metadata for embeddings
func buildAdEmbeddingMetadata(adObj ad.Ad) map[string]interface{} {
	return map[string]interface{}{
		"created_at":  adObj.CreatedAt.Format(time.RFC3339),
		"click_count": adObj.ClickCount,
	}
}

// Helper function for embedding generation
func joinStrings(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return fmt.Sprintf("%s", ss)
}

// StartBackgroundVectorProcessor starts a background goroutine that processes ads without vectors
func StartBackgroundVectorProcessor() {
	go func() {
		log.Printf("[BackgroundVectorProcessor] Starting background vector processor")
		for {
			// Get ads without vectors (database already filters them)
			ads, err := ad.GetAdsWithoutVectors()
			if err != nil {
				log.Printf("[BackgroundVectorProcessor] Failed to get ads: %v", err)
				time.Sleep(30 * time.Second)
				continue
			}

			if len(ads) == 0 {
				log.Printf("[BackgroundVectorProcessor] No ads without vectors to process, sleeping for 15 minutes")
				time.Sleep(15 * time.Minute)
				continue
			}

			log.Printf("[BackgroundVectorProcessor] Found %d ads without vectors to process", len(ads))

			// Process each ad
			processed := 0
			for i, adObj := range ads {
				log.Printf("[BackgroundVectorProcessor] Processing ad %d/%d: %s", i+1, len(ads), adObj.Title)

				// Build embedding
				err := BuildAdEmbedding(adObj)
				if err != nil {
					log.Printf("[BackgroundVectorProcessor] Failed to build embedding for ad %d: %v", adObj.ID, err)
					continue
				}

				processed++

				// Sleep to avoid rate limits
				time.Sleep(100 * time.Millisecond)
			}

			if processed > 0 {
				log.Printf("[BackgroundVectorProcessor] Processed %d ads, sleeping for 5 minutes", processed)
				time.Sleep(5 * time.Minute)
			} else {
				log.Printf("[BackgroundVectorProcessor] No ads needed processing, sleeping for 15 minutes")
				time.Sleep(15 * time.Minute)
			}
		}
	}()
}
