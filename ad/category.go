package ad

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

var (
	adCategoryNames map[int]string    // id -> name
	adCategoryIDs   map[string]int    // name -> id
	adFields        map[int][]Field   // id -> []Field
	adFieldChains   map[int][][]Field // id -> [][]Field (array of field chains per category)
	prompts         map[int]string    // id -> precalculated query prompt template
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
		WITH category_fields AS (
			SELECT
				acf.ad_category_id,
				json_agg(
					json_build_object(
						'ad_category_id', acf.ad_category_id,
						'name', sf.name,
						'display_name', sf.display_name,
						'field_type', sf.field_type,
						'table_name', sf.table_name,
						'vector_type', sf.vector_type,
						'spec_table_name', acf.spec_table_name,
						'spec_table_field_name', acf.spec_table_field_name,
						'ad_spec_table_name', acf.ad_spec_table_name,
						'next_in_chain', acf.next_in_chain,
						'is_required', acf.is_required::boolean,
						'show_in_filters', acf.show_in_filters::boolean,
						'is_multi_value_field', (sf.vector_type = 'keyword_array')::boolean
					) ORDER BY acf.priority_order
				) AS fields
			FROM AdCategoryField acf
			JOIN SpecField sf ON acf.spec_field_id = sf.id
			GROUP BY acf.ad_category_id
		)
		SELECT COALESCE(
			json_object_agg(
				ad_category_id::text,
				COALESCE(fields, '[]'::json)
			),
			'{}'::json
		)
		FROM category_fields
	`

	var result map[string][]adField
	if err := db.QueryJSON(&result, query); err != nil {
		// Handle case where there are no categories yet (empty database)
		if err == sql.ErrNoRows {
			result = make(map[string][]adField)
		} else {
			return err
		}
	}

	// Handle empty result (no categories yet)
	if result == nil {
		result = make(map[string][]adField)
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
			if fields[i].NextInChain == 0 {
				if inChain {
					fields[i].ParentField = &fields[i-1]
					inChain = false
				}
			} else {
				if inChain {
					fields[i].ParentField = &fields[i-1]
				}
				inChain = true
			}

			// Concrete Field instance
			newField, err := NewAdField(fields[i])
			if err != nil {
				logger.Error("Failed to create field instance",
					"error", err, "category_id", id,
					"field_name", fields[i].FieldName, "component", "ad-category")
				continue
			}
			newFields = append(newFields, newField)

			// Build the chains while we're at it
			if fields[i].NextInChain == 0 {
				if len(newChain) > 0 {
					newChain = append(newChain, newField)
					newChains = append(newChains, newChain)
					newChain = nil
				}
			} else {
				newChain = append(newChain, newField)
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

	// Precalculate query prompt templates for each category
	precalculateQueryPrompts()

	return nil
}

// precalculateQueryPrompts builds and caches query prompt templates for all categories
func precalculateQueryPrompts() {
	prompts = make(map[int]string)
	// Iterate over all category names to ensure we generate prompts for all categories
	for categoryID, categoryName := range adCategoryNames {
		// Get fields for this category (may be empty)
		fields := adFields[categoryID]
		var specFieldNames []string
		for _, field := range fields {
			if field.IsSpecField() {
				// Format field name nicely (e.g., "part_category" -> "Part category")
				fieldName := field.Name()
				formatted := strings.ReplaceAll(fieldName, "_", " ")
				// Capitalize first letter
				if len(formatted) > 0 {
					formatted = strings.ToUpper(formatted[:1]) + formatted[1:]
				}
				specFieldNames = append(specFieldNames, formatted)
			}
		}
		specFieldNamesStr := strings.Join(specFieldNames, ", ")

		// Build query prompt template
		var p []string
		p = append(p, fmt.Sprintf("Search query for ads in the %s category.", categoryName))
		p = append(p, "Find ads that match the user's search intent. Focus on matching:")
		p = append(p, "- Title and description content")
		if specFieldNamesStr != "" {
			p = append(p, fmt.Sprintf("- Specifications (%s)", specFieldNamesStr))
		}
		p = append(p, "- Location")
		p = append(p, "- Price range")
		p = append(p, "- Quality indicators")
		p = append(p, "")
		p = append(p, "User search query: %s")

		prompts[categoryID] = strings.Join(p, "\n")
	}
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

// GetFieldChains returns the field chains for a category
func GetFieldChains(adCat int) [][]Field {
	return adFieldChains[adCat]
}

// GetCategoryQueryPromptTemplate returns the precalculated query prompt template for a category (without user query)
func GetCategoryQueryPromptTemplate(adCat int) string {
	if template, exists := prompts[adCat]; exists {
		return template
	}
	// Fallback to generic template if category not found
	return `Search query for ads.
Find ads that match the user's search intent. Focus on matching:
- Title and description content
- Location
- Price range
- Quality indicators

User search query: %s`
}

// FieldsInSameChain checks if two fields are in the same chain
func FieldsInSameChain(field1, field2 Field, allFields []Field) bool {
	// Get chains for the category (we need to find which category these fields belong to)
	// Since we don't have category ID here, we'll check by iterating through chains
	// For now, let's use a simpler approach: check if field2 is reachable from field1 via NextFieldName

	// If field1 has no next field, it's not in a chain with field2 (unless field2 is field1)
	if field1.NextField() == "" {
		return field1.Name() == field2.Name()
	}

	// Check if field2 is in the chain starting from field1
	currentName := field1.Name()
	for {
		if currentName == field2.Name() {
			return true
		}

		// Find the next field
		found := false
		for _, f := range allFields {
			if f.Name() == currentName {
				currentName = f.NextField()
				found = true
				break
			}
		}

		if !found || currentName == "" {
			break
		}
	}

	// Also check reverse: if field2 is before field1 in the chain
	currentName = field2.Name()
	for {
		if currentName == field1.Name() {
			return true
		}

		found := false
		for _, f := range allFields {
			if f.Name() == currentName {
				currentName = f.NextField()
				found = true
				break
			}
		}

		if !found || currentName == "" {
			break
		}
	}

	return false
}
