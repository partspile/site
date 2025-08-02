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

// SetupPayloadIndexes creates all necessary payload indexes
// Note: This is a placeholder for future implementation when we understand the correct Qdrant API
func SetupPayloadIndexes() error {
	log.Printf("[qdrant] Payload indexing setup skipped - to be implemented later")
	return nil
}
