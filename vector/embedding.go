package vector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/vehicle"
	pinecone "github.com/pinecone-io/go-pinecone/v4/pinecone"
	genai "google.golang.org/genai"
	"google.golang.org/protobuf/types/known/structpb"
)

// AdResult represents a search result from Pinecone
// TODO: Fill in fields as needed
type AdResult struct {
	ID       string
	Score    float32
	Metadata map[string]interface{}
}

var (
	geminiClient   *genai.Client
	pineconeClient *pinecone.Client
	pineconeIndex  *pinecone.IndexConnection
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

// InitPineconeClient initializes the Pinecone client and index
func InitPineconeClient(apiKey, indexName string) error {
	if apiKey == "" {
		apiKey = config.PineconeAPIKey
	}
	if apiKey == "" {
		return fmt.Errorf("missing Pinecone API key")
	}
	if indexName == "" {
		indexName = config.PineconeIndex
	}
	if indexName == "" {
		return fmt.Errorf("missing Pinecone index name")
	}

	clientParams := pinecone.NewClientParams{
		ApiKey: apiKey,
	}
	pc, err := pinecone.NewClient(clientParams)
	if err != nil {
		return err
	}
	pineconeClient = pc

	// Describe the index to get the host
	idx, err := pc.DescribeIndex(context.Background(), indexName)
	if err != nil {
		return err
	}

	// Connect to the index using the host
	idxConn, err := pc.Index(pinecone.NewIndexConnParams{Host: idx.Host})
	if err != nil {
		return err
	}
	pineconeIndex = idxConn
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
	log.Printf("[embedding] Full prompt: %s", text)
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

// UpsertAdEmbedding upserts an ad's embedding and metadata into Pinecone
func UpsertAdEmbedding(adID int, embedding []float32, metadata map[string]interface{}) error {
	if pineconeIndex == nil {
		return fmt.Errorf("Pinecone index not initialized")
	}
	vectorID := fmt.Sprintf("%d", adID)
	log.Printf("[pinecone] Upserting vector for ad %d with ID %s", adID, vectorID)

	var metaStruct *structpb.Struct
	if metadata != nil {
		var err error
		metaStruct, err = structpb.NewStruct(metadata)
		if err != nil {
			return fmt.Errorf("failed to convert metadata: %w", err)
		}
		log.Printf("[pinecone] Metadata converted successfully for ad %d", adID)
	}

	// TODO: Confirm if Values should be []float32 or *[]float32 for Pinecone.Vector
	vector := &pinecone.Vector{
		Id:       vectorID,
		Values:   &embedding,
		Metadata: metaStruct,
	}
	log.Printf("[pinecone] Created vector object for ad %d with embedding length %d", adID, len(embedding))

	ctx := context.Background()
	resp, err := pineconeIndex.UpsertVectors(ctx, []*pinecone.Vector{vector})
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}

	log.Printf("[pinecone] Upsert response for ad %d: %+v", adID, resp)
	log.Printf("[pinecone] Successfully upserted vector for ad %d", adID)
	return nil
}

// QuerySimilarAds queries Pinecone for similar ads given an embedding
// Returns a list of AdResult, and a cursor for pagination
func QuerySimilarAds(embedding []float32, topK int, cursor string) ([]AdResult, string, error) {
	if pineconeIndex == nil {
		return nil, "", fmt.Errorf("Pinecone index not initialized")
	}
	ctx := context.Background()
	// TODO: Add metadata filter or namespace if needed
	// TODO: Use cursor for pagination if Pinecone supports it
	// If Vector expects *[]float32, take address; otherwise, use embedding directly
	resp, err := pineconeIndex.QueryByVectorValues(ctx, &pinecone.QueryByVectorValuesRequest{
		Vector:          embedding, // Pass as []float32, not &embedding
		TopK:            uint32(topK),
		IncludeValues:   false,
		IncludeMetadata: true,
		// Pagination/cursor support: Pinecone may use a NextPageToken or similar
		// PageToken: cursor, // Uncomment if supported
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to query Pinecone: %w", err)
	}
	var results []AdResult
	for _, match := range resp.Matches {
		adResult := AdResult{
			ID:       match.Vector.Id,
			Score:    match.Score,
			Metadata: match.Vector.Metadata.AsMap(),
		}
		results = append(results, adResult)
	}
	// TODO: Extract next cursor/page token if available
	return results, "", nil
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
		log.Printf("[site-level]   %d. Ad %d: %s (clicks: %d)",
			i+1, adObj.ID, adObj.Title, adObj.ClickCount)
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

// GetAdEmbedding retrieves the embedding for a given ad ID from Pinecone
func GetAdEmbedding(adID int) ([]float32, error) {
	if pineconeIndex == nil {
		return nil, fmt.Errorf("Pinecone index not initialized")
	}
	vectorID := fmt.Sprintf("%d", adID)
	log.Printf("[pinecone] Fetching vector for ad %d with ID %s", adID, vectorID)

	ctx := context.Background()
	resp, err := pineconeIndex.FetchVectors(ctx, []string{vectorID})
	if err != nil {
		log.Printf("[pinecone] Fetch error for ad %d: %v", adID, err)
		return nil, err
	}

	log.Printf("[pinecone] Fetch response for ad %d: %+v", adID, resp)
	log.Printf("[pinecone] Available vectors in response: %v", func() []string {
		keys := make([]string, 0, len(resp.Vectors))
		for k := range resp.Vectors {
			keys = append(keys, k)
		}
		return keys
	}())

	v, ok := resp.Vectors[vectorID]
	if !ok {
		log.Printf("[pinecone] Vector ID %s not found in response for ad %d", vectorID, adID)
		return nil, fmt.Errorf("no embedding found for ad %d", adID)
	}
	if v == nil {
		log.Printf("[pinecone] Vector object is nil for ad %d", adID)
		return nil, fmt.Errorf("no embedding found for ad %d", adID)
	}
	if v.Values == nil {
		log.Printf("[pinecone] Vector values are nil for ad %d", adID)
		return nil, fmt.Errorf("no embedding found for ad %d", adID)
	}

	log.Printf("[pinecone] Successfully retrieved vector for ad %d with length %d", adID, len(*v.Values))
	return *v.Values, nil
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

	// Store in Pinecone
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
