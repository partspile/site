package vector

import (
	"context"
	"fmt"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/parts-pile/site/spec"
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
// This queries SpecificationField table to get ALL spec fields dynamically
func GetIndexedFields() []QdrantFieldSchema {
	// Get all field definitions from cache
	// We need to query the database to get all fields
	// For now, use a helper function that gets all fields from cache
	allFields := getAllSpecFields()

	// Build QdrantFieldSchema list
	var indexedFields []QdrantFieldSchema

	// Add non-spec fields that are always indexed
	indexedFields = append(indexedFields,
		QdrantFieldSchema{Name: "ad_category_id", QdrantType: qdrant.FieldType_FieldTypeInteger},
		QdrantFieldSchema{Name: "price", QdrantType: qdrant.FieldType_FieldTypeFloat},
		QdrantFieldSchema{Name: "rock_count", QdrantType: qdrant.FieldType_FieldTypeInteger},
		QdrantFieldSchema{Name: "location", QdrantType: qdrant.FieldType_FieldTypeGeo, IsStruct: true},
	)

	// Add part category/subcategory (these are not in SpecificationField but are used)
	indexedFields = append(indexedFields,
		QdrantFieldSchema{Name: "category", QdrantType: qdrant.FieldType_FieldTypeKeyword},
		QdrantFieldSchema{Name: "subcategory", QdrantType: qdrant.FieldType_FieldTypeKeyword},
	)

	// Add all spec fields from SpecificationField table
	for _, fieldDef := range allFields {
		var qdrantType qdrant.FieldType
		isArray := false

		switch fieldDef.QdrantType {
		case "keyword":
			qdrantType = qdrant.FieldType_FieldTypeKeyword
		case "keyword_array":
			qdrantType = qdrant.FieldType_FieldTypeKeyword
			isArray = true
		case "integer":
			qdrantType = qdrant.FieldType_FieldTypeInteger
		case "float":
			qdrantType = qdrant.FieldType_FieldTypeFloat
		case "geo":
			qdrantType = qdrant.FieldType_FieldTypeGeo
		default:
			// Default to keyword
			qdrantType = qdrant.FieldType_FieldTypeKeyword
		}

		indexedFields = append(indexedFields, QdrantFieldSchema{
			Name:       fieldDef.Name,
			QdrantType: qdrantType,
			IsArray:    isArray,
		})
	}

	return indexedFields
}

// getAllSpecFields gets all field definitions from the database
func getAllSpecFields() []spec.FieldConfig {
	// Query all SpecificationField records
	// Since we can't easily access the cache from here, we'll query the DB
	// In practice, this should use the cache, but for initialization we query DB
	query := `
		SELECT id, name, display_name, field_type, table_name, table_field_name, qdrant_type
		FROM SpecificationField
		ORDER BY id
	`

	type dbField struct {
		ID             int    `db:"id"`
		Name           string `db:"name"`
		DisplayName    string `db:"display_name"`
		FieldType      string `db:"field_type"`
		TableName      string `db:"table_name"`
		TableFieldName string `db:"table_field_name"`
		QdrantType     string `db:"qdrant_type"`
	}

	var dbFields []dbField
	err := db.Select(&dbFields, query)
	if err != nil {
		logger.Warn("Failed to query SpecificationField for Qdrant indexes",
			"error", err, "component", "qdrant")
		return []spec.FieldConfig{}
	}

	var fieldConfigs []spec.FieldConfig
	for _, f := range dbFields {
		fieldConfigs = append(fieldConfigs, spec.FieldConfig{
			ID:             f.ID,
			Name:           f.Name,
			DisplayName:    f.DisplayName,
			FieldType:      f.FieldType,
			TableName:      f.TableName,
			TableFieldName: f.TableFieldName,
			QdrantType:     f.QdrantType,
		})
	}

	return fieldConfigs
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
		logger.Info("Creating collection",
			"component", "qdrant", "collectionName", collectionName)
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
		logger.Info("Successfully created collection",
			"component", "qdrant", "collectionName", collectionName)
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

	logger.Info("Setting up payload indexes for collection",
		"component", "qdrant", "collectionName", collectionName)

	// Get existing collection info to check what indexes already exist
	collectionInfo, err := qdrantClient.GetCollectionInfo(ctx, collectionName)
	if err != nil {
		logger.Warn("Failed to get collection info, will try to create indexes anyway",
			"error", err, "component", "qdrant")
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

	// Track all indexes (existing and newly created) for summary logging
	type indexInfo struct {
		Name     string
		Type     string
		IsArray  bool
		IsStruct bool
		Status   string // "existing" or "created"
		Params   string
	}
	var allIndexes []indexInfo

	for _, field := range indexFields {
		// Determine index type string and params
		var typeStr, paramsStr string
		switch field.QdrantType {
		case qdrant.FieldType_FieldTypeKeyword:
			typeStr = "keyword"
			paramsStr = "{}"
		case qdrant.FieldType_FieldTypeInteger:
			typeStr = "integer"
			paramsStr = "{}"
		case qdrant.FieldType_FieldTypeFloat:
			typeStr = "float"
			paramsStr = "{}"
		case qdrant.FieldType_FieldTypeGeo:
			typeStr = "geo"
			paramsStr = "{}"
		default:
			typeStr = "keyword"
			paramsStr = "{}"
		}

		// Check if index already exists
		if existingIndexedFields[field.Name] {
			logger.Debug("Index for field already exists, skipping",
				"component", "qdrant", "field", field.Name)
			allIndexes = append(allIndexes, indexInfo{
				Name:     field.Name,
				Type:     typeStr,
				IsArray:  field.IsArray,
				IsStruct: field.IsStruct,
				Status:   "existing",
				Params:   paramsStr,
			})
			continue
		}

		logger.Info("Creating index for field",
			"component", "qdrant", "field", field.Name, "type", typeStr)

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
			logger.Error("Failed to create index for field",
				"error", err, "component", "qdrant", "field", field.Name)
			return fmt.Errorf("failed to create index for field %s: %w", field.Name, err)
		}

		logger.Info("Successfully created index for field",
			"component", "qdrant", "field", field.Name)

		allIndexes = append(allIndexes, indexInfo{
			Name:     field.Name,
			Type:     typeStr,
			IsArray:  field.IsArray,
			IsStruct: field.IsStruct,
			Status:   "created",
			Params:   paramsStr,
		})
	}

	// Log summary of all indexes
	logger.Info("Qdrant collection indexes summary",
		"component", "qdrant",
		"collectionName", collectionName,
		"totalIndexes", len(allIndexes))

	// Log each index separately for better readability
	for _, idx := range allIndexes {
		attrs := []any{
			"component", "qdrant",
			"field", idx.Name,
			"type", idx.Type,
			"status", idx.Status,
		}
		if idx.IsArray {
			attrs = append(attrs, "isArray", true)
		}
		if idx.IsStruct {
			attrs = append(attrs, "isStruct", true)
		}
		logger.Info("Qdrant index", attrs...)
	}

	logger.Info("Payload indexes setup completed",
		"component", "qdrant", "collectionName", collectionName)
	return nil
}
