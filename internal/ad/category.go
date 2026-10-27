package ad

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/partspile/site/internal/config"
	"github.com/partspile/site/internal/db"
	"github.com/partspile/site/internal/facet"
)

type Category struct {
	ID        int      `json:"id"`
	Name      string   `json:"name"`
	ImageFile string   `json:"image_file"`
	FacetKeys []string `json:"facets"`
}

// Facets returns the facet definitions this category uses, in declared order.
func (c Category) Facets() []facet.Def {
	defs := make([]facet.Def, 0, len(c.FacetKeys))
	for _, key := range c.FacetKeys {
		if d, ok := facet.Get(key); ok {
			defs = append(defs, d)
		}
	}
	return defs
}

var (
	categories      = make(map[int]Category)
	defaultCategory int
)

func LoadCategories() error {
	query := `
		SELECT COALESCE(
			json_agg(
				json_build_object(
					'id', id,
					'name', name,
					'image_file', image_file,
					'facets', facets::jsonb
				)
			),
			'[]'::json
		)
		FROM categories`
	var allCategories []Category
	if err := db.QueryJSON(&allCategories, query); err != nil {
		return fmt.Errorf("loading categories: %w", err)
	}
	for _, cat := range allCategories {
		categories[cat.ID] = cat
	}

	// Precalculate default category ID
	var err error
	defaultCategory, err = GetCategoryIDByName(config.DefaultAdCategoryName)
	if err != nil {
		return fmt.Errorf("default category not found: %w", err)
	}

	return nil
}

func ParseCategory(categoryStr string) int {
	categoryID, err := strconv.Atoi(categoryStr)
	if err != nil {
		return defaultCategory
	}
	_, ok := categories[categoryID]
	if !ok {
		return defaultCategory
	}
	return categoryID
}

func GetCategoryName(categoryID int) (string, error) {
	category, ok := categories[categoryID]
	if !ok {
		return "", fmt.Errorf("category not found: %d", categoryID)
	}
	return category.Name, nil
}

func GetCategory(categoryID int) Category {
	if cat, ok := categories[categoryID]; ok {
		return cat
	}
	return categories[defaultCategory]
}

func DefaultCategoryID() int {
	return defaultCategory
}

func GetCategoryImageFile(categoryID int) (string, error) {
	category, ok := categories[categoryID]
	if !ok {
		return "", fmt.Errorf("category not found: %d", categoryID)
	}
	return category.ImageFile, nil
}

func GetCategories() ([]Category, error) {
	keys := make([]int, 0, len(categories))
	for k := range categories {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	cats := make([]Category, len(keys))
	for i, k := range keys {
		cats[i] = categories[k]
	}
	return cats, nil
}

// GetCategoryIDByName returns the ID of a category by its name
func GetCategoryIDByName(name string) (int, error) {
	for id, cat := range categories {
		if cat.Name == name {
			return id, nil
		}
	}
	return 0, fmt.Errorf("category not found: %s", name)
}
