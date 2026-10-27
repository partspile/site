package dbinit

import (
	"fmt"

	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/seed"
)

// Rebuild drops all tables, applies schema.sql, then loads categories
// (and optionally seed users/ads).
func Rebuild(loadSeed bool) error {
	if err := db.ResetSchema(); err != nil {
		return fmt.Errorf("reset schema: %w", err)
	}
	if loadSeed {
		if err := seed.LoadAll(); err != nil {
			return fmt.Errorf("load seed: %w", err)
		}
		return nil
	}
	if err := seed.LoadCategories(); err != nil {
		return fmt.Errorf("load categories: %w", err)
	}
	return nil
}
