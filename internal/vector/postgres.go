package vector

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/logger"
	"github.com/pgvector/pgvector-go"
)

func replacePlaceholders(query string, startIndex int) string {
	re := regexp.MustCompile(`\$(\d+)`)
	return re.ReplaceAllStringFunc(query, func(match string) string {
		num, _ := strconv.Atoi(match[1:])
		return fmt.Sprintf("$%d", num+startIndex-1)
	})
}

func UpsertAdEmbeddings(adIDs []int, embeddings [][]float32,
	metadatas []map[string]any) error {
	if len(adIDs) == 0 {
		return nil
	}
	if len(adIDs) != len(embeddings) || len(adIDs) != len(metadatas) {
		return fmt.Errorf("mismatched upsert lengths")
	}
	for i, adID := range adIDs {
		metaJSON, err := json.Marshal(metadatas[i])
		if err != nil {
			return fmt.Errorf("metadata json: %w", err)
		}
		vec := pgvector.NewVector(embeddings[i])
		_, err = db.Exec(
			`UPDATE ads SET embedding = $1::vector, vector_metadata = $2::jsonb
			 WHERE id = $3`,
			vec, string(metaJSON), adID,
		)
		if err != nil {
			return fmt.Errorf("upsert vector ad %d: %w", adID, err)
		}
	}
	return nil
}

func DeleteAdEmbedding(adID int) error {
	_, err := db.Exec(
		`UPDATE ads SET embedding = NULL, vector_metadata = NULL WHERE id = $1`,
		adID,
	)
	return err
}

// QuerySimilarAdIDs returns nearest ad IDs under threshold matching whereClause.
// whereClause placeholders start at $1 and are remapped after the embedding
// and threshold args.
func QuerySimilarAdIDs(embedding []float32, whereClause string, whereArgs []any,
	limit, offset int, threshold float64) ([]int, error) {
	vec := pgvector.NewVector(embedding)
	args := []any{vec, threshold}
	argIndex := 3
	filterWhere := whereClause
	whereSQL := `embedding IS NOT NULL
		  AND inactive_at IS NULL AND deleted_at IS NULL
		  AND (embedding <=> $1::vector) < $2`
	if whereClause != "" {
		whereClause = replacePlaceholders(whereClause, argIndex)
		whereSQL += " AND " + whereClause
		args = append(args, whereArgs...)
		argIndex += len(whereArgs)
	}
	limitPH := argIndex
	offsetPH := argIndex + 1
	args = append(args, limit, offset)

	query := fmt.Sprintf(`
		SELECT id
		FROM ads
		WHERE %s
		ORDER BY embedding <=> $1::vector
		LIMIT $%d OFFSET $%d`,
		whereSQL, limitPH, offsetPH,
	)

	logger.Debug("vector search sql",
		"sql", strings.Join(strings.Fields(query), " "),
		"argCount", len(args),
	)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("vector search: %w", err)
	}
	defer rows.Close()

	var ids []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			logger.Warn("vector search scan", "error", err)
			continue
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	logger.Debug("vector search query",
		"threshold", threshold,
		"limit", limit,
		"offset", offset,
		"where", whereClause,
		"resultCount", len(ids),
		"resultIDs", ids,
	)
	if len(ids) == 0 {
		logVectorSearchDiagnostics(vec, filterWhere, whereArgs, threshold)
	}
	return ids, nil
}

func GetAdEmbeddings(adIDs []int) ([][]float32, error) {
	if len(adIDs) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(adIDs))
	args := make([]any, len(adIDs))
	for i, id := range adIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	query := fmt.Sprintf(`
		SELECT id, embedding
		FROM ads
		WHERE id IN (%s) AND embedding IS NOT NULL`,
		strings.Join(placeholders, ","),
	)
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	byID := make(map[int][]float32, len(adIDs))
	for rows.Next() {
		var id int
		var vec pgvector.Vector
		if err := rows.Scan(&id, &vec); err != nil {
			continue
		}
		byID[id] = vec.Slice()
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([][]float32, len(adIDs))
	for i, id := range adIDs {
		out[i] = byID[id]
	}
	return out, nil
}

func logVectorSearchDiagnostics(vec pgvector.Vector, whereClause string,
	whereArgs []any, threshold float64) {
	baseWhere := `embedding IS NOT NULL AND inactive_at IS NULL AND deleted_at IS NULL`
	if whereClause != "" {
		filterClause := replacePlaceholders(whereClause, 3)
		baseWhere += " AND " + filterClause
	}

	countQuery := fmt.Sprintf(`
		SELECT
			COUNT(*)::int,
			COUNT(*) FILTER (WHERE (embedding <=> $1::vector) < $2)::int,
			COALESCE(MIN(embedding <=> $1::vector), -1)
		FROM ads
		WHERE %s`, baseWhere)
	countArgs := append([]any{vec, threshold}, whereArgs...)

	var embeddedTotal, withinThreshold int
	var minDistance float64
	if err := db.QueryRow(countQuery, countArgs...).Scan(
		&embeddedTotal, &withinThreshold, &minDistance,
	); err != nil {
		logger.Debug("vector search diagnostics count failed", "error", err)
		return
	}

	nearestWhere := `embedding IS NOT NULL AND inactive_at IS NULL AND deleted_at IS NULL`
	if whereClause != "" {
		nearestWhere += " AND " + replacePlaceholders(whereClause, 2)
	}
	nearestQuery := fmt.Sprintf(`
		SELECT id, (embedding <=> $1::vector) AS distance
		FROM ads
		WHERE %s
		ORDER BY embedding <=> $1::vector
		LIMIT 5`, nearestWhere)
	rows, err := db.Query(nearestQuery, append([]any{vec}, whereArgs...)...)
	if err != nil {
		logger.Debug("vector search diagnostics nearest failed", "error", err)
		return
	}
	defer rows.Close()

	var nearest []searchDistanceRow
	for rows.Next() {
		var row searchDistanceRow
		if err := rows.Scan(&row.ID, &row.Distance); err != nil {
			continue
		}
		nearest = append(nearest, row)
	}

	logger.Debug("vector search diagnostics",
		"embeddedInFilter", embeddedTotal,
		"withinThreshold", withinThreshold,
		"threshold", threshold,
		"minDistance", minDistance,
		"nearest", formatDistances(nearest),
	)
}
