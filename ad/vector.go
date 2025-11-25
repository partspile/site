package ad

import (
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
		WHERE a.embedding IS NULL AND a.deleted_at IS NULL
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
