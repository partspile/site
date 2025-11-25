package vector

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/pgvector/pgvector-go"
)

// replacePlaceholders replaces $1, $2, etc. in a SQL query with $startIndex, $startIndex+1, etc.
func replacePlaceholders(query string, startIndex int) string {
	re := regexp.MustCompile(`\$(\d+)`)
	return re.ReplaceAllStringFunc(query, func(match string) string {
		numStr := match[1:]
		num, _ := strconv.Atoi(numStr)
		return fmt.Sprintf("$%d", num+startIndex-1)
	})
}

// convertMetadataToJSONB converts metadata from Go types to JSONB format
func convertMetadataToJSONB(metadata map[string]interface{}) ([]byte, error) {
	if metadata == nil {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(metadata)
}

// UpsertAdEmbeddings upserts multiple ads' embeddings and metadata into PostgreSQL
func UpsertAdEmbeddings(adIDs []int, embeddings [][]float32, metadatas []map[string]interface{}) error {
	if len(adIDs) == 0 {
		return nil
	}

	if len(adIDs) != len(embeddings) || len(adIDs) != len(metadatas) {
		return fmt.Errorf("mismatched array lengths: adIDs=%d, embeddings=%d, metadatas=%d",
			len(adIDs), len(embeddings), len(metadatas))
	}

	logger.Debug("Upserting vectors for ads in batch",
		"component", "postgres", "count", len(adIDs))

	// Process each ad embedding
	for i, adID := range adIDs {
		embedding := embeddings[i]
		metadata := metadatas[i]

		// Convert metadata to JSONB
		metadataJSON, err := convertMetadataToJSONB(metadata)
		if err != nil {
			logger.Error("Failed to convert metadata to JSONB",
				"error", err, "component", "postgres", "adID", adID)
			return fmt.Errorf("failed to convert metadata to JSONB: %w", err)
		}

		// Create pgvector.Vector from float32 slice
		vec := pgvector.NewVector(embedding)

		// Upsert embedding and metadata
		query := `
			UPDATE Ad 
			SET embedding = $1::vector, vector_metadata = $2::jsonb
			WHERE id = $3
		`
		result, err := db.Exec(query, vec, string(metadataJSON), adID)
		if err != nil {
			logger.Error("Failed to upsert vector",
				"error", err, "component", "postgres", "adID", adID)
			return fmt.Errorf("failed to upsert vector: %w", err)
		}

		rowsAffected, err := result.RowsAffected()
		if err != nil {
			logger.Error("Failed to get rows affected",
				"error", err, "component", "postgres", "adID", adID)
		} else if rowsAffected == 0 {
			// Ad doesn't exist, insert instead (shouldn't happen, but handle gracefully)
			logger.Warn("Ad not found for vector upsert, skipping",
				"component", "postgres", "adID", adID)
		}
	}

	logger.Debug("Successfully upserted vectors in batch",
		"component", "postgres", "count", len(adIDs))
	return nil
}

// DeleteAdEmbedding deletes an ad's embedding from PostgreSQL
func DeleteAdEmbedding(adID int) error {
	logger.Debug("Deleting vector for ad",
		"component", "postgres", "adID", adID)

	query := `UPDATE Ad SET embedding = NULL, vector_metadata = NULL WHERE id = $1`
	_, err := db.Exec(query, adID)
	if err != nil {
		logger.Error("Failed to delete vector",
			"error", err, "component", "postgres", "adID", adID)
		return fmt.Errorf("failed to delete vector: %w", err)
	}

	logger.Debug("Successfully deleted vector for ad",
		"component", "postgres", "adID", adID)
	return nil
}

// QuerySimilarAdIDs queries PostgreSQL for similar ad IDs given an embedding
// Returns a list of ad IDs, and a cursor for pagination
// whereClause should be a SQL WHERE clause string (without WHERE keyword) and args
func QuerySimilarAdIDs(embedding []float32, whereClause string, whereArgs []interface{}, topK int, cursor uint64, threshold float64) ([]int, uint64, error) {
	// Create pgvector.Vector from float32 slice
	vec := pgvector.NewVector(embedding)

	// Build query with vector similarity search
	query := `
		SELECT id 
		FROM Ad 
		WHERE embedding IS NOT NULL
		  AND deleted_at IS NULL
		  AND (embedding <=> $1::vector) < $2
	`

	// Build args: embedding vector, threshold
	args := []interface{}{vec, threshold}
	argIndex := 3

	// Add WHERE clause conditions if provided
	if whereClause != "" {
		// Replace $1, $2, etc. in whereClause with $3, $4, etc.
		whereClause = replacePlaceholders(whereClause, argIndex)
		query += " AND " + whereClause
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
	}

	query += fmt.Sprintf(`
		ORDER BY embedding <=> $1::vector
		LIMIT $%d OFFSET $%d
	`, argIndex, argIndex+1)
	args = append(args, topK, cursor)

	logger.Debug("Querying similar ads",
		"component", "postgres", "topK", topK, "cursor", cursor, "threshold", threshold)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query PostgreSQL: %w", err)
	}
	defer rows.Close()

	var results []int
	for rows.Next() {
		var adID int
		if err := rows.Scan(&adID); err != nil {
			logger.Error("Failed to scan ad ID",
				"error", err, "component", "postgres")
			continue
		}
		results = append(results, adID)
		logger.Debug("Added query result",
			"component", "postgres", "adID", adID)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating results: %w", err)
	}

	logger.Debug("Query returned results",
		"component", "postgres", "count", len(results), "requested", topK, "offset", cursor)

	// Generate next cursor if we have results
	if len(results) > 0 {
		cursor += uint64(len(results))
		logger.Debug("Generated next cursor",
			"component", "postgres", "cursor", cursor)
	} else {
		cursor = 0
		logger.Debug("No results, no next cursor generated",
			"component", "postgres")
	}

	return results, cursor, nil
}

// GetAdEmbeddings retrieves embeddings for multiple ad IDs from PostgreSQL
func GetAdEmbeddings(adIDs []int) ([][]float32, error) {
	if len(adIDs) == 0 {
		return nil, nil
	}

	logger.Debug("Fetching vectors for ads in batch",
		"component", "postgres", "count", len(adIDs))

	// Build query with IN clause using placeholders
	placeholders := make([]string, len(adIDs))
	args := make([]interface{}, len(adIDs))
	for i, adID := range adIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = adID
	}

	query := fmt.Sprintf(`
		SELECT id, embedding 
		FROM Ad 
		WHERE id IN (%s)
		  AND embedding IS NOT NULL
	`, strings.Join(placeholders, ","))

	rows, err := db.Query(query, args...)
	if err != nil {
		logger.Error("Batch fetch error",
			"error", err, "component", "postgres", "count", len(adIDs))
		return nil, fmt.Errorf("PostgreSQL batch fetch error: %w", err)
	}
	defer rows.Close()

	// Create a map of ad ID to embedding for easy lookup
	embeddingMap := make(map[int][]float32)
	for rows.Next() {
		var adID int
		var vec pgvector.Vector
		if err := rows.Scan(&adID, &vec); err != nil {
			logger.Error("Failed to scan embedding",
				"error", err, "component", "postgres", "adID", adID)
			continue
		}

		// Convert pgvector.Vector to []float32
		// pgvector.Vector has a Slice() method that returns []float32
		embeddingF32 := vec.Slice()

		embeddingMap[adID] = embeddingF32
		logger.Debug("Successfully retrieved vector for ad",
			"component", "postgres", "adID", adID, "length", len(embeddingF32))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating results: %w", err)
	}

	logger.Debug("Batch fetch response",
		"component", "postgres", "pointsFound", len(embeddingMap))

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
		"component", "postgres", "count", len(result))
	return result, nil
}
