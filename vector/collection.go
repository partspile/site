package vector

import (
	"context"
	"fmt"
	"log"

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
	} else {
		log.Printf("[qdrant] Collection already exists: %s", collectionName)
	}

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
	}

	log.Printf("[qdrant] Setting up payload indexes for collection: %s", collectionName)

	for _, field := range indexFields {
		log.Printf("[qdrant] Creating index for field: %s (type: %v)", field.fieldName, field.fieldSchema)

		_, err := qdrantClient.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: collectionName,
			FieldName:      field.fieldName,
			FieldType:      &field.fieldSchema,
		})

		if err != nil {
			// Check if index already exists (common error)
			if fmt.Sprintf("%v", err) == "rpc error: code = AlreadyExists desc = Index already exists" {
				log.Printf("[qdrant] Index for field %s already exists", field.fieldName)
			} else {
				log.Printf("[qdrant] Failed to create index for field %s: %v", field.fieldName, err)
				return fmt.Errorf("failed to create index for field %s: %w", field.fieldName, err)
			}
		} else {
			log.Printf("[qdrant] Successfully created index for field: %s", field.fieldName)
		}
	}

	log.Printf("[qdrant] Payload indexes setup completed")
	return nil
}
