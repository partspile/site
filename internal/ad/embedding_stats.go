package ad

import (
	"github.com/partspile/site/internal/db"
)

type EmbeddingStats struct {
	Embedded int
	Missing  int
}

type MissingEmbedding struct {
	ID           int    `db:"id"`
	Title        string `db:"title"`
	CategoryID   int    `db:"category_id"`
	CategoryName string `db:"category_name"`
}

func GetEmbeddingStats() (EmbeddingStats, error) {
	var s EmbeddingStats
	err := db.QueryRow(`
		SELECT
			COUNT(*) FILTER (WHERE embedding IS NOT NULL),
			COUNT(*) FILTER (WHERE embedding IS NULL)
		FROM ads
		WHERE inactive_at IS NULL AND deleted_at IS NULL`,
	).Scan(&s.Embedded, &s.Missing)
	return s, err
}

func ListMissingEmbeddings(limit int) ([]MissingEmbedding, error) {
	if limit <= 0 {
		limit = 25
	}
	var rows []MissingEmbedding
	err := db.Select(&rows, `
		SELECT a.id, a.title, a.category_id, c.name AS category_name
		FROM ads a
		JOIN categories c ON c.id = a.category_id
		WHERE a.inactive_at IS NULL AND a.deleted_at IS NULL AND a.embedding IS NULL
		ORDER BY a.id
		LIMIT $1`, limit)
	return rows, err
}
