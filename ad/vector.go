package ad

import (
	"fmt"
	"strings"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// GetAdsWithoutVectors returns ad IDs for ads that don't have vector embeddings
func GetAdsWithoutVectors() ([]int, error) {
	logger.Debug("Querying for ads without vectors",
		"component", "ad")
	query := `
		SELECT a.id
		FROM Ad a
		WHERE a.has_vector = 0 AND a.deleted_at IS NULL
	`

	var adIDs []int
	err := db.Select(&adIDs, query)
	if err != nil {
		logger.Error("SQL error querying ads without vectors",
			"error", err, "component", "ad")
		return nil, err
	}

	logger.Debug("Found ads without vectors from SQL query",
		"component", "ad", "count", len(adIDs))
	return adIDs, nil
}

// MarkAdAsHavingVector marks an ad as having a vector embedding
func MarkAdAsHavingVector(adID int) error {
	return MarkAdsAsHavingVector([]int{adID})
}

// MarkAdsAsHavingVector marks multiple ads as having vector embeddings in a single SQL call
func MarkAdsAsHavingVector(adIDs []int) error {
	if len(adIDs) == 0 {
		return nil
	}

	// Build the SQL query with PostgreSQL placeholders for all ad IDs
	placeholders := make([]string, len(adIDs))
	args := make([]interface{}, len(adIDs))
	for i, adID := range adIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = adID
	}

	query := "UPDATE Ad SET has_vector = 1 WHERE id IN (" + strings.Join(placeholders, ",") + ")"

	_, err := db.Exec(query, args...)
	if err != nil {
		logger.Error("Failed to mark ads as having vector",
			"error", err, "component", "ad", "count", len(adIDs))
		return err
	}
	logger.Debug("Successfully marked ads as having vector",
		"component", "ad", "count", len(adIDs))
	return nil
}
