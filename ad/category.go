package ad

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

var (
	adCategoryNames map[int]string    // id -> name
	adCategoryIDs   map[string]int    // name -> id
	adFields        map[int][]Field   // id -> []Field
	adFieldChains   map[int][][]Field // id -> [][]Field (array of field chains per category)
)

// loadCategoriesFromDB loads category data from the database and populates the maps
func loadCategoriesFromDB() error {
	adCategoryNames = make(map[int]string)
	adCategoryIDs = make(map[string]int)

	rows, err := db.Query("SELECT id, name FROM AdCategory ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			logger.Error("Failed to scan AdCategory row",
				"error", err, "component", "ad-category")
			continue
		}
		adCategoryNames[id] = name
		adCategoryIDs[name] = id
	}
	return nil
}

func loadCategoryFieldsFromDB() error {
	adFields = make(map[int][]Field)
	adFieldChains = make(map[int][][]Field)

	query := `
		SELECT json_object_agg(
			acf.ad_category_id::text,
			COALESCE(
				json_agg(
					json_build_object(
						'ad_category_id', acf.ad_category_id,
						'name', sf.name,
						'display_name', sf.display_name,
						'field_type', sf.field_type,
						'table_name', sf.table_name,
						'qdrant_type', sf.qdrant_type,
						'spec_table_name', acf.spec_table_name,
						'spec_table_field_name', acf.spec_table_field_name,
						'ad_spec_table_name', acf.ad_spec_table_name,
						'next_in_chain', acf.next_in_chain,
						'is_required', acf.is_required::boolean,
						'show_in_filters', acf.show_in_filters::boolean,
						'is_multi_value_field', (sf.qdrant_type = 'keyword_array')::boolean
					) ORDER BY acf.priority_order
				),
				'[]'::json
			)
		)
		FROM AdCategoryField acf
		JOIN SpecField sf ON acf.spec_field_id = sf.id
		GROUP BY acf.ad_category_id
	`

	var result map[string][]adField
	if err := db.QueryJSON(&result, query); err != nil {
		return err
	}

	// Populate NextFieldName for each field in all categories
	for _, fields := range result {
		for i, field := range fields {
			if field.NextInChain != 0 && field.NextInChain < len(fields) {
				fields[i].NextFieldName = fields[field.NextInChain].FieldName
			}
		}
	}

	// Create concrete Field instances from adField
	for idStr, fields := range result {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Error("Failed to convert category ID",
				"error", err, "id", idStr, "component", "ad-category")
			continue
		}

		var newFields []Field
		var newChains [][]Field
		var newChain []Field
		var inChain bool
		for i := range fields {

			// Concrete Field instance
			newField, err := NewAdField(fields[i])
			if err != nil {
				logger.Error("Failed to create field instance",
					"error", err, "category_id", id,
					"field_name", fields[i].FieldName, "component", "ad-category")
				continue
			}
			newFields = append(newFields, newField)

			// Build the chains wile we're at it
			if fields[i].NextInChain == 0 {
				if inChain {
					newChain = append(newChain, newField)
					newChains = append(newChains, newChain)
					inChain = false
				}
				newChain = nil
			} else {
				newChain = append(newChain, newField)
				inChain = true
			}
		}
		adFields[id] = newFields
		adFieldChains[id] = newChains
	}

	return nil
}

func InitAdCategories() error {

	// Load AdCategory table
	if err := loadCategoriesFromDB(); err != nil {
		logger.Error("Failed to query AdCategory",
			"error", err, "component", "ad-category")
		return err
	}

	// Load AdCategoryField table
	if err := loadCategoryFieldsFromDB(); err != nil {
		logger.Error("Failed to query AdCategoryField",
			"error", err, "component", "ad-category")
		return err
	}

	return nil
}

// GetCategoryIDs returns all category IDs in order
func GetCategoryIDs() []int {
	ids := make([]int, 0, len(adCategoryNames))
	for id := range adCategoryNames {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

// GetCategoryName returns the name for a category ID
func GetCategoryName(adCat int) string {
	if name, exists := adCategoryNames[adCat]; exists {
		return name
	}
	return "Unknown Category"
}

// IsValidCategory returns true if the category ID is valid
func IsValidCategory(adCat int) bool {
	_, exists := adCategoryNames[adCat]
	return exists
}

// GetCategoryIDByName returns the category ID for a given category name, or 0 if not found
func GetCategoryIDByName(name string) (id int) {
	id = adCategoryIDs[name]
	return
}

func GetCategoryFields(adCat int) []Field {
	return adFields[adCat]
}

func GetCategoryField(adCat int, fieldName string) (Field, error) {
	fields := adFields[adCat]
	for _, field := range fields {
		if field.Name() == fieldName {
			return field, nil
		}
	}
	return nil, fmt.Errorf("field %s not found in category %d", fieldName, adCat)
}

func GetSpecFields(adCat int) []Field {
	fields := adFields[adCat]
	specFields := make([]Field, 0)
	for _, field := range fields {
		if field.IsSpecField() {
			specFields = append(specFields, field)
		}
	}
	return specFields
}
