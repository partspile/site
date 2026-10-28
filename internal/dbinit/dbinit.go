package dbinit

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/partspile/site/internal/db"
)

//go:embed categories.json
var categoriesJSON []byte

type categoryJSON struct {
	Name      string   `json:"name"`
	ImageFile string   `json:"image_file"`
	Facets    []string `json:"facets"`
}

// Rebuild drops all tables, applies schema.sql, then loads categories.
// Used by backup restore to prepare an empty database before import.
func Rebuild() error {
	if err := db.ResetSchema(); err != nil {
		return fmt.Errorf("reset schema: %w", err)
	}
	if err := LoadCategories(); err != nil {
		return fmt.Errorf("load categories: %w", err)
	}
	return nil
}

// LoadCategories inserts categories from the embedded catalog.
// IDs are assigned alphabetically by name (stable across rebuilds).
func LoadCategories() error {
	var categories []categoryJSON
	if err := json.Unmarshal(categoriesJSON, &categories); err != nil {
		return err
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})
	for _, cat := range categories {
		facetKeys := cat.Facets
		if facetKeys == nil {
			facetKeys = []string{}
		}
		facetsJSON, err := json.Marshal(facetKeys)
		if err != nil {
			return fmt.Errorf("marshaling facets for %s: %w", cat.Name, err)
		}
		_, err = db.Exec(
			`INSERT INTO categories (name, image_file, facets)
			 VALUES ($1, $2, $3)`,
			cat.Name, cat.ImageFile, string(facetsJSON),
		)
		if err != nil {
			return fmt.Errorf("inserting category %s: %w", cat.Name, err)
		}
	}
	return nil
}
