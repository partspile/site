package vector

import (
	"context"
	"fmt"
	"strings"

	"github.com/parts-pile/site/config"
	"github.com/parts-pile/site/logger"
	"github.com/qdrant/go-client/qdrant"
)

var qdrantClient *qdrant.Client

// InitQdrantClient initializes the Qdrant client
func InitQdrantClient() error {

	host := config.QdrantHost
	if host == "" {
		return fmt.Errorf("missing Qdrant host")
	}
	apiKey := config.QdrantAPIKey
	if apiKey == "" {
		return fmt.Errorf("missing Qdrant API key")
	}

	// Remove any protocol prefix if present
	if strings.HasPrefix(host, "https://") {
		host = strings.TrimPrefix(host, "https://")
	} else if strings.HasPrefix(host, "http://") {
		host = strings.TrimPrefix(host, "http://")
	}

	// Create Qdrant client configuration
	clientConfig := &qdrant.Config{
		APIKey:                 apiKey,
		Host:                   host,
		Port:                   config.QdrantPort, // Qdrant Cloud uses port 6334 for gRPC
		UseTLS:                 true,              // Qdrant Cloud requires TLS
		SkipCompatibilityCheck: true,              // Skip version check for cloud service
	}

	logger.Debug("Qdrant client config",
		"host", clientConfig.Host, "port", clientConfig.Port)

	// Create Qdrant client
	client, err := qdrant.NewClient(clientConfig)
	if err != nil {
		return err
	}
	qdrantClient = client

	return nil
}

// convertMetadataToQdrant converts metadata from Go types to Qdrant values
// Handles dynamic fields based on field configuration
func convertMetadataToQdrant(metadata map[string]interface{}) map[string]*qdrant.Value {
	if metadata == nil {
		return nil
	}

	qdrantMetadata := make(map[string]*qdrant.Value)
	for k, v := range metadata {
		switch k {
		case "location":
			// Handle geo location struct (special case - not a spec field)
			if geoMap, ok := v.(map[string]interface{}); ok {
				geoStruct := &qdrant.Struct{Fields: make(map[string]*qdrant.Value)}
				for geoKey, geoVal := range geoMap {
					if geoFloat, ok := geoVal.(float64); ok {
						geoStruct.Fields[geoKey] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: geoFloat}}
					}
				}
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_StructValue{StructValue: geoStruct}}
			}
		case "ad_category_id", "rock_count":
			// Handle integer fields (non-spec fields)
			if intVal, ok := v.(int); ok {
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(intVal)}}
			}
		case "price":
			// Handle float fields (non-spec fields)
			if floatVal, ok := v.(float64); ok {
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: floatVal}}
			}
		case "category", "subcategory":
			// Handle part category/subcategory (non-spec fields)
			if strVal, ok := v.(string); ok {
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: strVal}}
			}
		default:
			// For all spec fields and unknown fields, infer type from Go value
			// This allows new fields to work without code changes
			switch val := v.(type) {
			case string:
				// Single-value keyword fields (make, breed, part_subcategory, etc.)
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: val}}
			case []string:
				// Array fields (years, models, engines, or any future keyword_array fields)
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_ListValue{ListValue: &qdrant.ListValue{
					Values: func() []*qdrant.Value {
						result := make([]*qdrant.Value, len(val))
						for j, s := range val {
							result[j] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: s}}
						}
						return result
					}(),
				}}}
			case int:
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(val)}}
			case float64:
				qdrantMetadata[k] = &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: val}}
			}
		}
	}
	return qdrantMetadata
}

// UpsertAdEmbeddings upserts multiple ads' embeddings and metadata into Qdrant
func UpsertAdEmbeddings(adIDs []int, embeddings [][]float32, metadatas []map[string]interface{}) error {
	if qdrantClient == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}

	if len(adIDs) == 0 {
		return nil
	}

	if len(adIDs) != len(embeddings) || len(adIDs) != len(metadatas) {
		return fmt.Errorf("mismatched array lengths: adIDs=%d, embeddings=%d, metadatas=%d",
			len(adIDs), len(embeddings), len(metadatas))
	}

	logger.Debug("Upserting vectors for ads in batch",
		"component", "qdrant", "count", len(adIDs))

	// Convert all metadata to Qdrant format and create points
	var points []*qdrant.PointStruct
	for i, adID := range adIDs {
		embedding := embeddings[i]
		metadata := metadatas[i]

		// Convert metadata to Qdrant format
		qdrantMetadata := convertMetadataToQdrant(metadata)

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

		points = append(points, point)
	}

	logger.Debug("Upserting vectors in batch",
		"component", "qdrant", "count", len(points))

	ctx := context.Background()
	_, err := qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: config.QdrantCollection,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert vectors: %w", err)
	}

	logger.Debug("Successfully upserted vectors in batch",
		"component", "qdrant", "count", len(points))
	return nil
}

// DeleteAdEmbedding deletes an ad's embedding from Qdrant
func DeleteAdEmbedding(adID int) error {
	if qdrantClient == nil {
		return fmt.Errorf("Qdrant client not initialized")
	}
	logger.Debug("Deleting vector for ad",
		"component", "qdrant", "adID", adID)

	ctx := context.Background()
	pointID := qdrant.NewIDNum(uint64(adID))
	_, err := qdrantClient.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: config.QdrantCollection,
		Points: &qdrant.PointsSelector{
			PointsSelectorOneOf: &qdrant.PointsSelector_Points{
				Points: &qdrant.PointsIdsList{
					Ids: []*qdrant.PointId{pointID},
				},
			},
		},
	})
	if err != nil {
		logger.Error("Failed to delete vector",
			"error", err, "component", "qdrant", "adID", adID)
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	logger.Debug("Successfully deleted vector for ad",
		"component", "qdrant", "adID", adID)
	return nil
}

// QuerySimilarAdIDs queries Qdrant for similar ad IDs given an embedding
// Returns a list of ad IDs, and a cursor for pagination
// If filter is provided, it will be applied to the search
func QuerySimilarAdIDs(embedding []float32, filter *qdrant.Filter, topK int, cursor uint64, threshold float64) ([]int, uint64, error) {
	if qdrantClient == nil {
		return nil, 0, fmt.Errorf("Qdrant client not initialized")
	}
	ctx := context.Background()

	// Create search request using Query method
	limit := uint64(topK)
	scoreThreshold := float32(threshold)
	queryRequest := &qdrant.QueryPoints{
		CollectionName: config.QdrantCollection,
		Query:          qdrant.NewQueryDense(embedding),
		Filter:         filter,
		Limit:          &limit,
		Offset:         &cursor,
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
			return nil, 0, fmt.Errorf("failed to query Qdrant with filter: %w", err)
		}
		return nil, 0, fmt.Errorf("failed to query Qdrant: %w", err)
	}
	logger.Debug("Query returned results",
		"component", "qdrant", "count", len(resp), "requested", topK, "offset", cursor)

	var results []int
	for _, result := range resp {
		// Extract ID
		adID := int(result.Id.GetNum())
		results = append(results, adID)
		logger.Debug("Added query result",
			"component", "qdrant", "adID", adID, "score", result.Score)
	}

	// Generate next cursor if we have results
	if len(results) > 0 {
		cursor += uint64(len(results))
		logger.Debug("Generated next cursor",
			"component", "qdrant", "cursor", cursor)
	} else {
		cursor = 0
		logger.Debug("No results, no next cursor generated",
			"component", "qdrant")
	}

	return results, cursor, nil
}

// GetAdEmbeddings retrieves embeddings for multiple ad IDs from Qdrant in a single API call
func GetAdEmbeddings(adIDs []int) ([][]float32, error) {
	if qdrantClient == nil {
		return nil, fmt.Errorf("Qdrant client not initialized")
	}

	if len(adIDs) == 0 {
		return nil, nil
	}

	logger.Debug("Fetching vectors for ads in batch",
		"component", "qdrant", "count", len(adIDs))

	// Create point IDs for batch retrieval
	var pointIDs []*qdrant.PointId
	for _, adID := range adIDs {
		pointIDs = append(pointIDs, qdrant.NewIDNum(uint64(adID)))
	}

	ctx := context.Background()
	resp, err := qdrantClient.Get(ctx, &qdrant.GetPoints{
		CollectionName: config.QdrantCollection,
		Ids:            pointIDs,
		WithPayload:    &qdrant.WithPayloadSelector{SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true}},
		WithVectors:    &qdrant.WithVectorsSelector{SelectorOptions: &qdrant.WithVectorsSelector_Enable{Enable: true}},
	})
	if err != nil {
		logger.Error("Batch fetch error",
			"error", err, "component", "qdrant", "count", len(adIDs))
		return nil, fmt.Errorf("Qdrant batch fetch error: %w", err)
	}

	logger.Debug("Batch fetch response",
		"component", "qdrant", "pointsFound", len(resp))

	// Create a map of ad ID to embedding for easy lookup
	embeddingMap := make(map[int][]float32)
	for _, point := range resp {
		adID := int(point.Id.GetNum())

		if point.Vectors == nil {
			logger.Debug("Vector values are nil for ad",
				"component", "qdrant", "adID", adID)
			continue
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

		if vectorData != nil {
			embeddingMap[adID] = vectorData
			logger.Debug("Successfully retrieved vector for ad",
				"component", "qdrant", "adID", adID, "length", len(vectorData))
		} else {
			logger.Debug("No vector data found for ad",
				"component", "qdrant", "adID", adID)
		}
	}

	// Return embeddings in the same order as requested ad IDs
	var result [][]float32
	for _, adID := range adIDs {
		if embedding, exists := embeddingMap[adID]; exists {
			result = append(result, embedding)
		} else {
			result = append(result, nil) // Keep order, but mark as missing
		}
	}

	logger.Debug("Successfully retrieved embeddings in batch",
		"component", "qdrant", "count", len(result))
	return result, nil
}
