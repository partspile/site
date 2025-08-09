package vector

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/parts-pile/site/config"
	"github.com/qdrant/go-client/qdrant"
)

// EnsureCollectionExists creates the Qdrant collection if it doesn't exist
func EnsureCollectionExists() error {
	if qdrantClient == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}

	collectionName := config.QdrantCollection
	ctx := context.Background()

	// Check if collection exists
	collections, err := qdrantClient.ListCollections(ctx)
	if err != nil {
		return fmt.Errorf("failed to get collections: %w", err)
	}

	collectionExists := false
	for _, col := range collections {
		if col == collectionName {
			collectionExists = true
			break
		}
	}

	if !collectionExists {
		log.Printf("[qdrant] Creating collection: %s", collectionName)
		err = qdrantClient.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: collectionName,
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
		log.Printf("[qdrant] Successfully created collection: %s", collectionName)
	}
	// Note: Collection already exists logging is handled in InitQdrantClient

	return nil
}

// SetupPayloadIndexes creates all necessary payload indexes for filtering
func SetupPayloadIndexes() error {
	if qdrantClient == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}

	collectionName := config.QdrantCollection
	ctx := context.Background()

	// Define the fields that need indexes for filtering
	indexFields := []struct {
		fieldName   string
		fieldSchema qdrant.FieldType
	}{
		{"make", qdrant.FieldType_FieldTypeKeyword},
		{"years", qdrant.FieldType_FieldTypeKeyword},
		{"models", qdrant.FieldType_FieldTypeKeyword},
		{"engines", qdrant.FieldType_FieldTypeKeyword},
		{"category", qdrant.FieldType_FieldTypeKeyword},
		{"subcategory", qdrant.FieldType_FieldTypeKeyword},
		{"price", qdrant.FieldType_FieldTypeFloat},
		{"location", qdrant.FieldType_FieldTypeGeo},
	}

	log.Printf("[qdrant] Setting up payload indexes for collection: %s", collectionName)

	// Get existing collection info to check what indexes already exist
	collectionInfo, err := qdrantClient.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		log.Printf("[qdrant] Failed to get collection info: %v, will try to create indexes anyway", err)
		collectionInfo = nil
	}

	// Create a set of existing indexed fields
	existingIndexedFields := make(map[string]bool)
	if collectionInfo != nil && collectionInfo.PayloadSchema != nil {
		for fieldName, schemaInfo := range collectionInfo.PayloadSchema {
			if schemaInfo.Params != nil {
				existingIndexedFields[fieldName] = true
			}
		}
	}

	// Track which indexes we create
	createdIndexes := make([]string, 0)

	for _, field := range indexFields {
		// Check if index already exists
		if existingIndexedFields[field.fieldName] {
			log.Printf("[qdrant] Index for field %s already exists, skipping", field.fieldName)
			continue
		}

		log.Printf("[qdrant] Creating index for field: %s", field.fieldName)

		// Create the appropriate index parameters based on field type
		var fieldIndexParams *qdrant.PayloadIndexParams
		wait := true

		switch field.fieldSchema {
		case qdrant.FieldType_FieldTypeKeyword:
			fieldIndexParams = qdrant.NewPayloadIndexParamsKeyword(&qdrant.KeywordIndexParams{})
		case qdrant.FieldType_FieldTypeFloat:
			fieldIndexParams = qdrant.NewPayloadIndexParamsFloat(&qdrant.FloatIndexParams{})
		case qdrant.FieldType_FieldTypeGeo:
			fieldIndexParams = qdrant.NewPayloadIndexParamsGeo(&qdrant.GeoIndexParams{})
		default:
			// For other types, use keyword as default
			fieldIndexParams = qdrant.NewPayloadIndexParamsKeyword(&qdrant.KeywordIndexParams{})
		}

		// Try to create the index
		_, err := qdrantClient.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName:   collectionName,
			FieldName:        field.fieldName,
			FieldType:        &field.fieldSchema,
			FieldIndexParams: fieldIndexParams,
			Wait:             &wait,
		})

		if err != nil {
			// Check if index already exists - look for various error messages
			errorStr := fmt.Sprintf("%v", err)
			if strings.Contains(strings.ToLower(errorStr), "already exists") ||
				strings.Contains(strings.ToLower(errorStr), "alreadyexists") ||
				strings.Contains(strings.ToLower(errorStr), "index already exists") ||
				strings.Contains(strings.ToLower(errorStr), "field already exists") {
				log.Printf("[qdrant] Index for field %s already exists", field.fieldName)
			} else {
				log.Printf("[qdrant] Failed to create index for field %s: %v", field.fieldName, err)
				return fmt.Errorf("failed to create index for field %s: %w", field.fieldName, err)
			}
		} else {
			log.Printf("[qdrant] Successfully created index for field: %s", field.fieldName)
			createdIndexes = append(createdIndexes, field.fieldName)
		}
	}

	// Verify that indexes were actually created by checking collection info again
	if len(createdIndexes) > 0 {
		log.Printf("[qdrant] Verifying that %d indexes were created...", len(createdIndexes))

		// Wait for indexes to be created and become available
		maxRetries := config.QdrantMaxRetries
		retryDelay := config.QdrantRetryDelay

		for retry := 0; retry < maxRetries; retry++ {
			time.Sleep(retryDelay)

			// Check collection info again
			verifyInfo, err := qdrantClient.GetCollectionInfo(ctx, collectionName)
			if err != nil {
				log.Printf("[qdrant] Failed to verify collection info (attempt %d/%d): %v", retry+1, maxRetries, err)
				continue
			}

			if verifyInfo.PayloadSchema != nil {
				allIndexesCreated := true

				for _, field := range indexFields {
					if schemaInfo, exists := verifyInfo.PayloadSchema[field.fieldName]; exists {
						if schemaInfo.Params == nil {
							allIndexesCreated = false
						}
					} else {
						allIndexesCreated = false
					}
				}

				if allIndexesCreated {
					log.Printf("[qdrant] All indexes successfully created and verified")
					break
				}
			}

			if retry == maxRetries-1 {
				log.Printf("[qdrant] Warning: Some indexes may not be fully created after %d attempts", maxRetries)
			}
		}
	} else {
		log.Printf("[qdrant] No new indexes were created (all already existed)")
	}

	log.Printf("[qdrant] Payload indexes setup completed")
	return nil
}

// TestPayloadIndexes performs a simple test to verify that payload indexes are working
func TestPayloadIndexes() error {
	if qdrantClient == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}

	collectionName := config.QdrantCollection
	ctx := context.Background()

	// Test 1: Simple search with no filters
	limit := uint64(5)
	queryRequest := &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQueryDense(make([]float32, 768)), // Dummy vector
		Limit:          &limit,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	}

	_, err := qdrantClient.Query(ctx, queryRequest)
	if err != nil {
		return fmt.Errorf("basic search test failed: %w", err)
	}

	// Test 2: Search with metadata filter (make filter)
	filter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "make",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{Keyword: "BMW"},
						},
					},
				},
			},
		},
	}

	queryRequestWithFilter := &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQueryDense(make([]float32, 768)), // Dummy vector
		Limit:          &limit,
		Filter:         filter,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	}

	_, err = qdrantClient.Query(ctx, queryRequestWithFilter)
	if err != nil {
		return fmt.Errorf("search with filter test failed: %w", err)
	}

	// Test 3: Search with category filter
	categoryFilter := &qdrant.Filter{
		Must: []*qdrant.Condition{
			{
				ConditionOneOf: &qdrant.Condition_Field{
					Field: &qdrant.FieldCondition{
						Key: "category",
						Match: &qdrant.Match{
							MatchValue: &qdrant.Match_Keyword{Keyword: "Engine"},
						},
					},
				},
			},
		},
	}

	queryRequestWithCategoryFilter := &qdrant.QueryPoints{
		CollectionName: collectionName,
		Query:          qdrant.NewQueryDense(make([]float32, 768)), // Dummy vector
		Limit:          &limit,
		Filter:         categoryFilter,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
	}

	_, err = qdrantClient.Query(ctx, queryRequestWithCategoryFilter)
	if err != nil {
		return fmt.Errorf("search with category filter test failed: %w", err)
	}

	return nil
}
