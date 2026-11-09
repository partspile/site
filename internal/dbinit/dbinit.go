package dbinit

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/partspile/site/internal/config"
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

// LoadCategories upserts categories from the embedded catalog and removes
// obsolete rows (e.g. Garage/Estate Sale). Leftover categories with no
// shared facets empty the "All categories" search filter intersection.
func LoadCategories() error {
	var categories []categoryJSON
	if err := json.Unmarshal(categoriesJSON, &categories); err != nil {
		return err
	}
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Name < categories[j].Name
	})
	keep := make(map[string]bool, len(categories))
	for _, cat := range categories {
		keep[cat.Name] = true
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
			 VALUES ($1, $2, $3)
			 ON CONFLICT (name) DO UPDATE SET
			   image_file = EXCLUDED.image_file,
			   facets = EXCLUDED.facets`,
			cat.Name, cat.ImageFile, string(facetsJSON),
		)
		if err != nil {
			return fmt.Errorf("inserting category %s: %w", cat.Name, err)
		}
	}
	if err := pruneObsoleteCategories(keep); err != nil {
		return err
	}
	return nil
}

func pruneObsoleteCategories(keep map[string]bool) error {
	rows, err := db.Query(`SELECT id, name FROM categories`)
	if err != nil {
		return fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	type obsolete struct {
		id   int
		name string
	}
	var gone []obsolete
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return fmt.Errorf("scan category: %w", err)
		}
		if !keep[name] {
			gone = append(gone, obsolete{id: id, name: name})
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("list categories: %w", err)
	}
	if len(gone) == 0 {
		return nil
	}

	var defaultID int
	if err := db.QueryRow(
		`SELECT id FROM categories WHERE name = $1`,
		config.DefaultAdCategoryName,
	).Scan(&defaultID); err != nil {
		return fmt.Errorf("default category %q: %w",
			config.DefaultAdCategoryName, err)
	}

	for _, cat := range gone {
		if _, err := db.Exec(
			`UPDATE ads SET category_id = $1 WHERE category_id = $2`,
			defaultID, cat.id,
		); err != nil {
			return fmt.Errorf("reassign ads from %s: %w", cat.name, err)
		}
		if _, err := db.Exec(
			`DELETE FROM categories WHERE id = $1`, cat.id,
		); err != nil {
			return fmt.Errorf("delete obsolete category %s: %w", cat.name, err)
		}
	}
	return nil
}
