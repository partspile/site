package vector

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/parts-pile/site/config"
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

	log.Printf("[qdrant] Initializing client with host: %s", host)

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

	log.Printf("[qdrant] Client config: %+v", clientConfig)

	// Create Qdrant client
	client, err := qdrant.NewClient(clientConfig)
	if err != nil {
		return err
	}
	qdrantClient = client

	return nil
}

// UpsertAdEmbedding upserts an ad's embedding and metadata into Qdrant
func UpsertAdEmbedding(adID int, embedding []float32, metadata map[string]interface{}) error {
	return UpsertAdEmbeddings([]int{adID}, [][]float32{embedding}, []map[string]interface{}{metadata})
}

// UpsertAdEmbeddings upserts multiple ads' embeddings and metadata into Qdrant in a single API call
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

	log.Printf("[qdrant] Upserting vectors for %d ads in batch", len(adIDs))

	// Convert all metadata to Qdrant format and create points
	var points []*qdrant.PointStruct
	for i, adID := range adIDs {
		embedding := embeddings[i]
		metadata := metadatas[i]

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
							for j, s := range val {
								result[j] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: s}}
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

		points = append(points, point)
	}

	log.Printf("[qdrant] Upserting %d vectors in batch", len(points))

	ctx := context.Background()
	_, err := qdrantClient.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: config.QdrantCollection,
		Points:         points,
	})
	if err != nil {
		return fmt.Errorf("failed to upsert vectors: %w", err)
	}

	log.Printf("[qdrant] Successfully upserted %d vectors in batch", len(points))
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
		log.Printf("[qdrant] Failed to delete vector for ad %d: %v", adID, err)
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	log.Printf("[qdrant] Successfully deleted vector for ad %d", adID)
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
	log.Printf("[qdrant] Query returned %d results (requested %d, offset %d)", len(resp), topK, cursor)

	var results []int
	for _, result := range resp {
		// Extract ID
		adID := int(result.Id.GetNum())
		results = append(results, adID)
		log.Printf("[qdrant] Added result with ID: %d, Score: %f", adID, result.Score)
	}

	// Generate next cursor if we have results
	if len(results) > 0 {
		cursor += uint64(len(results))
		log.Printf("[qdrant] Generated next cursor: %d", cursor)
	} else {
		cursor = 0
		log.Printf("[qdrant] No results, no next cursor generated")
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

	log.Printf("[qdrant] Fetching vectors for %d ads in batch", len(adIDs))

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
		log.Printf("[qdrant] Batch fetch error for %d ads: %v", len(adIDs), err)
		return nil, fmt.Errorf("Qdrant batch fetch error: %w", err)
	}

	log.Printf("[qdrant] Batch fetch response: found %d points", len(resp))

	// Create a map of ad ID to embedding for easy lookup
	embeddingMap := make(map[int][]float32)
	for _, point := range resp {
		adID := int(point.Id.GetNum())

		if point.Vectors == nil {
			log.Printf("[qdrant] Vector values are nil for ad %d", adID)
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
			log.Printf("[qdrant] Successfully retrieved vector for ad %d with length %d", adID, len(vectorData))
		} else {
			log.Printf("[qdrant] No vector data found for ad %d", adID)
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

	log.Printf("[qdrant] Successfully retrieved %d embeddings in batch", len(result))
	return result, nil
}
