package vector

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/parts-pile/site/config"
	"github.com/qdrant/go-client/qdrant"
)

var (
	qdrantClient     *qdrant.Client
	qdrantCollection string
)

// InitQdrantClient initializes the Qdrant client and collection
func InitQdrantClient() error {
	host := config.QdrantHost
	if host == "" {
		return fmt.Errorf("missing Qdrant host")
	}
	apiKey := config.QdrantAPIKey
	if apiKey == "" {
		return fmt.Errorf("missing Qdrant API key")
	}
	collection := config.QdrantCollection
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
		clientConfig.Port = config.QdrantPort // Qdrant Cloud uses port 6334 for gRPC
	}

	// Create Qdrant client
	client, err := qdrant.NewClient(clientConfig)
	if err != nil {
		return err
	}
	qdrantClient = client
	qdrantCollection = collection

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
		CollectionName: qdrantCollection,
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
		CollectionName: qdrantCollection,
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
func QuerySimilarAdIDs(embedding []float32, filter *qdrant.Filter, topK int, cursor string, threshold float64) ([]int, string, error) {
	if qdrantClient == nil {
		return nil, "", fmt.Errorf("Qdrant client not initialized")
	}
	ctx := context.Background()

	// Parse cursor if provided
	offset := DecodeCursor(cursor)

	// Create search request using Query method
	limit := uint64(topK)
	scoreThreshold := float32(threshold)
	queryRequest := &qdrant.QueryPoints{
		CollectionName: qdrantCollection,
		Query:          qdrant.NewQueryDense(embedding),
		Filter:         filter,
		Limit:          &limit,
		Offset:         &offset,
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
			return nil, "", fmt.Errorf("failed to query Qdrant with filter: %w", err)
		}
		return nil, "", fmt.Errorf("failed to query Qdrant: %w", err)
	}
	log.Printf("[qdrant] Query returned %d results (requested %d, offset %d)", len(resp), topK, offset)

	var results []int
	for _, result := range resp {
		// Extract ID
		adID := int(result.Id.GetNum())
		results = append(results, adID)
		log.Printf("[qdrant] Added result with ID: %d, Score: %f", adID, result.Score)
	}

	// Generate next cursor if we have results
	var nextCursor string
	if len(results) > 0 {
		nextOffset := offset + uint64(len(results))
		nextCursor = EncodeCursor(nextOffset)
		log.Printf("[qdrant] Generated next cursor: %s (offset: %d)", nextCursor, nextOffset)
	} else {
		log.Printf("[qdrant] No results, no next cursor generated")
	}

	return results, nextCursor, nil
}

// GetAdEmbedding retrieves the embedding for a given ad ID from Qdrant
func GetAdEmbedding(adID int) ([]float32, error) {
	embeddings, err := GetAdEmbeddings([]int{adID})
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding found for ad %d", adID)
	}
	return embeddings[0], nil
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
		CollectionName: qdrantCollection,
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

// QuerySimilarAdsWithFilter queries Qdrant with filters
func QuerySimilarAdsWithFilter(embedding []float32, filter *qdrant.Filter, topK int, cursor string, threshold float64) ([]AdResult, string, error) {
	if qdrantClient == nil {
		return nil, "", fmt.Errorf("Qdrant client not initialized")
	}

	ctx := context.Background()

	// Parse cursor if provided
	offset := DecodeCursor(cursor)

	limit := uint64(topK)

	// Always use similarity threshold for vector search
	scoreThreshold := float32(threshold)
	queryRequest := &qdrant.QueryPoints{
		CollectionName: qdrantCollection,
		Query:          qdrant.NewQueryDense(embedding),
		Filter:         filter,
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

	var results []AdResult
	for _, result := range resp {
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

		var adID int
		if numID := result.Id.GetNum(); numID != 0 {
			adID = int(numID)
		} else {
			// Fallback to string ID if somehow we get a UUID
			if uuidStr := result.Id.GetUuid(); uuidStr != "" {
				// Try to parse as int if it's numeric
				if parsedID, err := strconv.Atoi(uuidStr); err == nil {
					adID = parsedID
				} else {
					adID = 0 // Invalid ID
				}
			}
		}

		adResult := AdResult{
			ID:       adID,
			Score:    float32(result.Score),
			Metadata: metadata,
		}
		results = append(results, adResult)
	}

	// Generate next cursor
	var nextCursor string
	if len(results) > 0 {
		nextOffset := offset + uint64(len(results))
		nextCursor = EncodeCursor(nextOffset)
	}

	return results, nextCursor, nil
}

// BuildTreeFilter creates a filter for tree navigation
func BuildTreeFilter(treePath map[string]string) *qdrant.Filter {
	var conditions []*qdrant.Condition

	if make, ok := treePath["make"]; ok && make != "" {
		// URL decode the make value
		decodedMake, err := url.QueryUnescape(make)
		if err != nil {
			decodedMake = make // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "make",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: decodedMake}},
				},
			},
		})
	}

	if year, ok := treePath["year"]; ok && year != "" {
		// URL decode the year value
		decodedYear, err := url.QueryUnescape(year)
		if err != nil {
			decodedYear = year // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "years",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keywords{Keywords: &qdrant.RepeatedStrings{Strings: []string{decodedYear}}}},
				},
			},
		})
	}

	if model, ok := treePath["model"]; ok && model != "" {
		// URL decode the model value
		decodedModel, err := url.QueryUnescape(model)
		if err != nil {
			decodedModel = model // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "models",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keywords{Keywords: &qdrant.RepeatedStrings{Strings: []string{decodedModel}}}},
				},
			},
		})
	}

	if engine, ok := treePath["engine"]; ok && engine != "" {
		// URL decode the engine value
		decodedEngine, err := url.QueryUnescape(engine)
		if err != nil {
			decodedEngine = engine // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "engines",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keywords{Keywords: &qdrant.RepeatedStrings{Strings: []string{decodedEngine}}}},
				},
			},
		})
	}

	if category, ok := treePath["category"]; ok && category != "" {
		// URL decode the category value
		decodedCategory, err := url.QueryUnescape(category)
		if err != nil {
			decodedCategory = category // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "category",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: decodedCategory}},
				},
			},
		})
	}

	if subcategory, ok := treePath["subcategory"]; ok && subcategory != "" {
		// URL decode the subcategory value
		decodedSubCategory, err := url.QueryUnescape(subcategory)
		if err != nil {
			decodedSubCategory = subcategory // fallback to original if decoding fails
		}
		conditions = append(conditions, &qdrant.Condition{
			ConditionOneOf: &qdrant.Condition_Field{
				Field: &qdrant.FieldCondition{
					Key:   "subcategory",
					Match: &qdrant.Match{MatchValue: &qdrant.Match_Keyword{Keyword: decodedSubCategory}},
				},
			},
		})
	}

	if len(conditions) == 0 {
		return nil
	}

	return &qdrant.Filter{
		Must: conditions,
	}
}

// BuildGeoFilter creates a geo filter for location-based search
// Note: This is a placeholder - geo filtering may require different Qdrant API calls
func BuildGeoFilter(lat, lon float64, radiusMeters float64) *qdrant.Filter {
	// TODO: Implement proper geo filtering when we understand the Qdrant API
	log.Printf("[vector] Geo filtering not yet implemented")
	return nil
}

// BuildBoundingBoxGeoFilter creates a geo filter for bounding box search
func BuildBoundingBoxGeoFilter(minLat, maxLat, minLon, maxLon float64) *qdrant.Filter {
	log.Printf("[vector] Building bounding box filter: lat[%.6f,%.6f], lon[%.6f,%.6f]", minLat, maxLat, minLon, maxLon)

	// Create geo bounding box filter using Qdrant's native geo filtering
	// Note: The order is topLeft.lat, topLeft.lon, bottomRight.lat, bottomRight.lon
	// topLeft = maxLat, minLon (northwest corner)
	// bottomRight = minLat, maxLon (southeast corner)
	geoCondition := qdrant.NewGeoBoundingBox("location", maxLat, minLon, minLat, maxLon)

	conditions := []*qdrant.Condition{
		geoCondition,
	}

	// Create filter with conditions
	filter := &qdrant.Filter{
		Must: conditions,
	}

	log.Printf("[vector] Created Qdrant geo bounding box filter")
	return filter
}
