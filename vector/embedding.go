package vector

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"database/sql"

	"github.com/parts-pile/site/ad"
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

var db *sql.DB

// InitDB sets the database connection for the vector package
func InitDB(database *sql.DB) {
	db = database
}

// InitGeminiClient initializes the Gemini embedding client
func InitGeminiClient(apiKey string) error {
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
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
		apiKey = os.Getenv("PINECONE_API_KEY")
	}
	if apiKey == "" {
		return fmt.Errorf("missing Pinecone API key")
	}
	if indexName == "" {
		indexName = os.Getenv("PINECONE_INDEX")
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

// UpsertAdEmbedding upserts an ad's embedding and metadata into Pinecone
func UpsertAdEmbedding(adID int, embedding []float32, metadata map[string]interface{}) error {
	if pineconeIndex == nil {
		return fmt.Errorf("Pinecone index not initialized")
	}
	vectorID := fmt.Sprintf("%d", adID)
	var metaStruct *structpb.Struct
	if metadata != nil {
		var err error
		metaStruct, err = structpb.NewStruct(metadata)
		if err != nil {
			return fmt.Errorf("failed to convert metadata: %w", err)
		}
	}
	// TODO: Confirm if Values should be []float32 or *[]float32 for Pinecone.Vector
	vector := &pinecone.Vector{
		Id:       vectorID,
		Values:   &embedding,
		Metadata: metaStruct,
	}
	ctx := context.Background()
	_, err := pineconeIndex.UpsertVectors(ctx, []*pinecone.Vector{vector})
	if err != nil {
		return fmt.Errorf("failed to upsert vector: %w", err)
	}
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
	var vectors [][]float32
	var missingIDs []int
	for _, adObj := range ads {
		emb, err := GetAdEmbedding(adObj.ID)
		if err != nil || emb == nil {
			missingIDs = append(missingIDs, adObj.ID)
			continue
		}
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
	ctx := context.Background()
	resp, err := pineconeIndex.FetchVectors(ctx, []string{vectorID})
	if err != nil {
		return nil, err
	}
	v, ok := resp.Vectors[vectorID]
	if !ok || v == nil || v.Values == nil {
		return nil, fmt.Errorf("no embedding found for ad %d", adID)
	}
	return *v.Values, nil
}
