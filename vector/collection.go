package vector

import (
	"context"
	"fmt"
	"log"

	"github.com/parts-pile/site/config"
	"github.com/qdrant/go-client/qdrant"
)

// QdrantFieldSchema defines the schema for a field in Qdrant
type QdrantFieldSchema struct {
	Name       string
	QdrantType qdrant.FieldType
	IsArray    bool
	IsStruct   bool
}

// GetIndexedFields returns the list of fields that need indexes
func GetIndexedFields() []QdrantFieldSchema {
	return []QdrantFieldSchema{
		{Name: "make", QdrantType: qdrant.FieldType_FieldTypeKeyword},
		{Name: "years", QdrantType: qdrant.FieldType_FieldTypeKeyword, IsArray: true},
		{Name: "models", QdrantType: qdrant.FieldType_FieldTypeKeyword, IsArray: true},
		{Name: "engines", QdrantType: qdrant.FieldType_FieldTypeKeyword, IsArray: true},
		{Name: "category", QdrantType: qdrant.FieldType_FieldTypeKeyword},
		{Name: "subcategory", QdrantType: qdrant.FieldType_FieldTypeKeyword},
		{Name: "ad_category_id", QdrantType: qdrant.FieldType_FieldTypeInteger},
		{Name: "price", QdrantType: qdrant.FieldType_FieldTypeFloat},
		{Name: "rock_count", QdrantType: qdrant.FieldType_FieldTypeInteger},
		{Name: "location", QdrantType: qdrant.FieldType_FieldTypeGeo, IsStruct: true},
	}
}

// InitQdrantCollection creates the Qdrant collection if it doesn't exist
func InitQdrantCollection() error {
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
						Size:     config.GeminiEmbeddingDimensions,
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

	return nil
}

// InitQdrantIndexes creates all necessary payload indexes for filtering
func InitQdrantIndexes() error {
	if qdrantClient == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}

	collectionName := config.QdrantCollection
	ctx := context.Background()

	// Get the fields that need indexes for filtering
	indexFields := GetIndexedFields()

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

	for _, field := range indexFields {
		// Check if index already exists
		if existingIndexedFields[field.Name] {
			log.Printf("[qdrant] Index for field %s already exists, skipping", field.Name)
			continue
		}

		log.Printf("[qdrant] Creating index for field: %s", field.Name)

		// Create the appropriate index parameters based on field type
		var fieldIndexParams *qdrant.PayloadIndexParams
		wait := true

		switch field.QdrantType {
		case qdrant.FieldType_FieldTypeKeyword:
			fieldIndexParams = qdrant.NewPayloadIndexParamsKeyword(&qdrant.KeywordIndexParams{})
		case qdrant.FieldType_FieldTypeInteger:
			fieldIndexParams = qdrant.NewPayloadIndexParamsInt(&qdrant.IntegerIndexParams{})
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
			FieldName:        field.Name,
			FieldType:        &field.QdrantType,
			FieldIndexParams: fieldIndexParams,
			Wait:             &wait,
		})

		if err != nil {
			log.Printf("[qdrant] Failed to create index for %s: %v", field.Name, err)
			return fmt.Errorf("failed to create index for field %s: %w", field.Name, err)
		}

		log.Printf("[qdrant] Successfully created index for field: %s", field.Name)
	}

	log.Printf("[qdrant] Payload indexes setup completed")
	return nil
}
