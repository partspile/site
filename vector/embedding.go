package vector

import (
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/parts-pile/site/ad"
	"github.com/parts-pile/site/cache"
	"github.com/parts-pile/site/rock"
	"github.com/parts-pile/site/vehicle"
)

// AdResult represents a search result from Qdrant
type AdResult struct {
	ID       int
	Score    float32
	Metadata map[string]interface{}
}

var (
	// Specialized caches for different embedding types
	queryEmbeddingCache *cache.Cache[[]float32] // String keys, 1 hour TTL
	userEmbeddingCache  *cache.Cache[[]float32] // User ID keys, 24 hour TTL
	siteEmbeddingCache  *cache.Cache[[]float32] // Campaign keys, 6 hour TTL
)

// EncodeCursor encodes an offset into a base64-encoded cursor string
func EncodeCursor(offset uint64) string {
	if offset == 0 {
		return ""
	}
	offsetStr := strconv.FormatUint(offset, 10)
	return base64.StdEncoding.EncodeToString([]byte(offsetStr))
}

// DecodeCursor decodes a base64-encoded cursor string into an offset
func DecodeCursor(cursor string) uint64 {
	if cursor == "" {
		log.Printf("[qdrant] No cursor provided, starting from beginning")
		return 0
	}

	// Decode cursor: format is "offset" base64 encoded
	cursorBytes, err := base64.StdEncoding.DecodeString(cursor)
	if err != nil {
		log.Printf("[qdrant] Failed to decode cursor: %v", err)
		return 0
	}

	cursorStr := string(cursorBytes)
	offsetVal, err := strconv.ParseUint(cursorStr, 10, 64)
	if err != nil {
		log.Printf("[qdrant] Failed to parse cursor offset: %v", err)
		return 0
	}

	log.Printf("[qdrant] Parsed cursor: offset=%d", offsetVal)
	return offsetVal
}

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

// ClearEmbeddingCache clears all cached embeddings
func ClearEmbeddingCache() {
	if queryEmbeddingCache != nil {
		queryEmbeddingCache.Clear()
	}
	if userEmbeddingCache != nil {
		userEmbeddingCache.Clear()
	}
	if siteEmbeddingCache != nil {
		siteEmbeddingCache.Clear()
	}
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
	var textIndices []int // Track original indices for result ordering
	for i, text := range texts {
		text = strings.TrimSpace(text)
		if text == "" {
			return nil, fmt.Errorf("cannot embed empty text at index %d", i)
		}
		validTexts = append(validTexts, text)
		textIndices = append(textIndices, i)
	}

	// Check cache for each text and collect uncached ones
	var cachedEmbeddings [][]float32
	var uncachedTexts []string
	var uncachedIndices []int

	for i, text := range validTexts {
		if cached, found := queryEmbeddingCache.Get(text); found {
			log.Printf("[embedding][query-cache] Cache hit for query: %.80q", text)
			cachedEmbeddings = append(cachedEmbeddings, cached)
		} else {
			uncachedTexts = append(uncachedTexts, text)
			uncachedIndices = append(uncachedIndices, i)
		}
	}

	// Generate embeddings for uncached texts in batch
	var newEmbeddings [][]float32
	if len(uncachedTexts) > 0 {
		log.Printf("[embedding][query-cache] Cache miss for %d queries, generating batch embeddings", len(uncachedTexts))
		embeddings, err := EmbedTexts(uncachedTexts)
		if err != nil {
			return nil, err
		}

		// Cache the new embeddings
		for i, embedding := range embeddings {
			text := uncachedTexts[i]
			queryEmbeddingCache.SetWithTTL(text, embedding, int64(len(embedding)*4), time.Hour)
			log.Printf("[embedding][query-cache] Cached embedding for query: %.80q", text)
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

	log.Printf("[embedding][query-cache] Batch processed %d queries (%d cached, %d generated)",
		len(validTexts), len(cachedEmbeddings), len(newEmbeddings))
	return result, nil
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

	// Extract ad IDs for batch retrieval
	var adIDs []int
	for _, adObj := range ads {
		adIDs = append(adIDs, adObj.ID)
	}

	// Fetch all embeddings in batch
	log.Printf("[site-level] Fetching embeddings for %d ads in batch", len(adIDs))
	embeddings, err := GetAdEmbeddings(adIDs)
	if err != nil {
		log.Printf("[site-level] Batch fetch error: %v", err)
		return nil, err
	}

	// Process results
	var vectors [][]float32
	var missingIDs []int
	for i, embedding := range embeddings {
		adID := adIDs[i]
		if embedding == nil {
			missingIDs = append(missingIDs, adID)
			log.Printf("[site-level] Missing embedding for ad %d", adID)
			continue
		}
		log.Printf("[site-level] Found embedding for ad %d (length=%d)", adID, len(embedding))
		vectors = append(vectors, embedding)
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
	rockPreferenceEmbedding, err := GetQueryEmbedding(rockPreferencePrompt)
	if err == nil && rockPreferenceEmbedding != nil {
		// Blend the site-level vector with rock preference
		enhancedVectors := [][]float32{result, rockPreferenceEmbedding}
		enhancedWeights := []float32{1.0, 1.5} // Site-level gets weight 1.0, rock preference gets 1.5
		result = AggregateEmbeddings(enhancedVectors, enhancedWeights)
		log.Printf("[site-level] Enhanced site-level vector with rock preference")
	}

	return result, nil
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

// BuildAdEmbeddings builds and stores embeddings for multiple ads in batch
func BuildAdEmbeddings(ads []ad.Ad) error {
	if len(ads) == 0 {
		return nil
	}

	log.Printf("[BuildAdEmbeddings] Building embeddings for %d ads in batch", len(ads))

	// Build prompts for all ads
	var prompts []string
	var validAds []ad.Ad
	for _, adObj := range ads {
		prompt := buildAdEmbeddingPrompt(adObj)
		if prompt != "" {
			prompts = append(prompts, prompt)
			validAds = append(validAds, adObj)
		} else {
			log.Printf("[BuildAdEmbeddings] Skipping ad %d due to empty prompt", adObj.ID)
		}
	}

	if len(prompts) == 0 {
		return fmt.Errorf("no valid prompts generated for any ads")
	}

	// Generate embeddings in batch
	log.Printf("[BuildAdEmbeddings] Generating batch embeddings for %d ads", len(validAds))
	embeddings, err := EmbedTexts(prompts)
	if err != nil {
		log.Printf("[BuildAdEmbeddings] Failed to generate batch embeddings: %v", err)
		return err
	}

	// Process each ad with its embedding
	var successCount, errorCount int
	var adIDs []int
	var adEmbeddings [][]float32
	var adMetadatas []map[string]interface{}

	for i, adObj := range validAds {
		if i >= len(embeddings) {
			log.Printf("[BuildAdEmbeddings] Missing embedding for ad %d", adObj.ID)
			errorCount++
			continue
		}

		embedding := embeddings[i]
		if embedding == nil {
			log.Printf("[BuildAdEmbeddings] Nil embedding for ad %d", adObj.ID)
			errorCount++
			continue
		}

		// Build metadata
		meta := BuildAdEmbeddingMetadata(adObj)

		// Collect for batch upsert
		adIDs = append(adIDs, adObj.ID)
		adEmbeddings = append(adEmbeddings, embedding)
		adMetadatas = append(adMetadatas, meta)
	}

	// Batch upsert to Qdrant
	if len(adIDs) > 0 {
		err := UpsertAdEmbeddings(adIDs, adEmbeddings, adMetadatas)
		if err != nil {
			log.Printf("[BuildAdEmbeddings] Failed to batch upsert vectors: %v", err)
			errorCount += len(adIDs)
		} else {
			log.Printf("[BuildAdEmbeddings] Successfully batch upserted %d vectors", len(adIDs))
		}
	}

	// Mark ads as having vectors in database (batch operation)
	err = ad.MarkAdsAsHavingVector(adIDs)
	if err != nil {
		log.Printf("[BuildAdEmbeddings] Failed to mark ads as having vector: %v", err)
		errorCount += len(adIDs)
	} else {
		log.Printf("[BuildAdEmbeddings] Successfully marked %d ads as having vector", len(adIDs))
	}

	// Calculate final success count (ads that were successfully processed)
	successCount = len(adIDs) - errorCount

	log.Printf("[BuildAdEmbeddings] Batch processing complete: %d successful, %d errors", successCount, errorCount)

	if errorCount > 0 {
		return fmt.Errorf("batch processing completed with %d errors", errorCount)
	}
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
