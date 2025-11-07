package spec

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// FieldConfig represents a specification field configuration
type FieldConfig struct {
	ID            int
	Name          string
	DisplayName   string
	FieldType     string // 'string', 'int', 'float64'
	TableName     string // Always required - table name (e.g., 'Make', 'Year', 'Model')
	TableFieldName string // Column name in Specification table (e.g., 'make_id', 'year_id')
	QdrantType    string // 'keyword', 'keyword_array', 'integer', 'float', 'geo'
	IsRequired    bool
	// Chain information for cascading dropdowns
	ChainGroup    int  // Fields with same chain_group are in the same chain (0 = not in a chain)
	ChainPosition int  // Position within the chain (0 = first, 1 = second, etc.)
	NextInChain   string // Name of next field in chain (empty if last or not in chain)
}

var (
	// In-memory caches for field configurations
	categoryFieldsCache   map[int][]FieldConfig
	fieldDefinitionsCache map[string]FieldConfig
	fieldCacheMutex       sync.RWMutex
)

// InitFieldCache loads SpecificationField and AdCategoryField into memory maps at startup
func InitFieldCache() error {
	fieldCacheMutex.Lock()
	defer fieldCacheMutex.Unlock()

	// Initialize maps
	categoryFieldsCache = make(map[int][]FieldConfig)
	fieldDefinitionsCache = make(map[string]FieldConfig)

	// Load all SpecificationField records
	fieldDefs, err := loadFieldDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load field definitions: %w", err)
	}

	// Load all AdCategoryField records and build category -> fields mapping
	categoryFields, err := loadCategoryFields(fieldDefs)
	if err != nil {
		return fmt.Errorf("failed to load category fields: %w", err)
	}

	// Populate caches
	for name, def := range fieldDefs {
		fieldDefinitionsCache[name] = def
	}
	for catID, fields := range categoryFields {
		categoryFieldsCache[catID] = fields
	}

	logger.Info("Field cache initialized",
		"component", "spec-field", "fieldCount", len(fieldDefinitionsCache),
		"categoryCount", len(categoryFieldsCache))

	return nil
}

// loadFieldDefinitions loads all SpecificationField records from database
func loadFieldDefinitions() (map[string]FieldConfig, error) {
	query := `
		SELECT id, name, display_name, field_type, table_name, table_field_name, qdrant_type
		FROM SpecificationField
		ORDER BY id
	`

	type dbField struct {
		ID            int    `db:"id"`
		Name          string `db:"name"`
		DisplayName   string `db:"display_name"`
		FieldType     string `db:"field_type"`
		TableName     string `db:"table_name"`
		TableFieldName string `db:"table_field_name"`
		QdrantType    string `db:"qdrant_type"`
	}

	var dbFields []dbField
	err := db.Select(&dbFields, query)
	if err != nil {
		return nil, err
	}

	fieldDefs := make(map[string]FieldConfig)
	for _, f := range dbFields {
		fieldDefs[f.Name] = FieldConfig{
			ID:            f.ID,
			Name:          f.Name,
			DisplayName:   f.DisplayName,
			FieldType:     f.FieldType,
			TableName:     f.TableName,
			TableFieldName: f.TableFieldName,
			QdrantType:    f.QdrantType,
		}
	}

	return fieldDefs, nil
}

// loadCategoryFields loads all AdCategoryField records and builds category -> fields mapping
// Chain information is computed based on field order - consecutive fields with same chain_group
// are considered chained. For now, we'll compute chains based on display_order.
func loadCategoryFields(fieldDefs map[string]FieldConfig) (map[int][]FieldConfig, error) {
	query := `
		SELECT acf.ad_category_id, acf.specification_field_id, acf.display_order, acf.is_required
		FROM AdCategoryField acf
		JOIN SpecificationField sf ON acf.specification_field_id = sf.id
		ORDER BY acf.ad_category_id, acf.display_order
	`

	type dbCategoryField struct {
		AdCategoryID       int `db:"ad_category_id"`
		SpecificationFieldID int `db:"specification_field_id"`
		DisplayOrder       int `db:"display_order"`
		IsRequired         int `db:"is_required"`
	}

	var dbCategoryFields []dbCategoryField
	err := db.Select(&dbCategoryFields, query)
	if err != nil {
		return nil, err
	}

	// Build map of field ID -> FieldConfig
	fieldIDMap := make(map[int]FieldConfig)
	for _, def := range fieldDefs {
		fieldIDMap[def.ID] = def
	}

	// Group by category and sort by display_order
	categoryFieldsMap := make(map[int][]FieldConfig)
	for _, cf := range dbCategoryFields {
		fieldConfig, exists := fieldIDMap[cf.SpecificationFieldID]
		if !exists {
			logger.Warn("Field definition not found for category field",
				"component", "spec-field",
				"specification_field_id", cf.SpecificationFieldID,
				"ad_category_id", cf.AdCategoryID)
			continue
		}

		fieldConfig.IsRequired = cf.IsRequired != 0
		categoryFieldsMap[cf.AdCategoryID] = append(
			categoryFieldsMap[cf.AdCategoryID],
			fieldConfig,
		)
	}

	// Sort fields by display_order within each category and compute chain information
	for catID := range categoryFieldsMap {
		fields := categoryFieldsMap[catID]
		sort.Slice(fields, func(i, j int) bool {
			// Find display_order for each field
			var orderI, orderJ int
			for _, cf := range dbCategoryFields {
				if cf.AdCategoryID == catID {
					if cf.SpecificationFieldID == fields[i].ID {
						orderI = cf.DisplayOrder
					}
					if cf.SpecificationFieldID == fields[j].ID {
						orderJ = cf.DisplayOrder
					}
				}
			}
			return orderI < orderJ
		})
		
		// Compute chain information
		// For now, we'll need to load chain info from JSON file
		// This is a temporary solution - ideally we'd store chain info in DB
		fields = computeChainInfo(fields, catID)
		categoryFieldsMap[catID] = fields
	}

	return categoryFieldsMap, nil
}

// computeChainInfo computes chain information for fields by loading from ad-category.json
// This is a temporary solution until we store chain info in the database
func computeChainInfo(fields []FieldConfig, categoryID int) []FieldConfig {
	// Load ad-category.json to get chain syntax
	// This is a workaround - ideally chain info would be in the database
	categoryFile := "cmd/rebuild_db/ad-category.json"
	data, err := os.ReadFile(categoryFile)
	if err != nil {
		logger.Debug("Failed to read ad-category.json for chain info", "error", err)
		return fields
	}

	type AdCategoryJSON struct {
		Name       string   `json:"name"`
		FieldNames []string `json:"field_names"`
	}
	var categories []AdCategoryJSON
	if err := json.Unmarshal(data, &categories); err != nil {
		logger.Debug("Failed to parse ad-category.json", "error", err)
		return fields
	}

	// Find category name by ID
	categoryName := ""
	rows, err := db.Query("SELECT name FROM AdCategory WHERE id = ?", categoryID)
	if err == nil {
		defer rows.Close()
		if rows.Next() {
			rows.Scan(&categoryName)
		}
	}

	if categoryName == "" {
		return fields
	}

	// Find matching category in JSON
	var fieldNamesWithChain []string
	for _, cat := range categories {
		if cat.Name == categoryName {
			fieldNamesWithChain = cat.FieldNames
			break
		}
	}

	if len(fieldNamesWithChain) == 0 {
		return fields
	}

	// Build map of field name -> field config
	fieldMap := make(map[string]*FieldConfig)
	for i := range fields {
		fieldMap[fields[i].Name] = &fields[i]
	}

	// Parse chain information
	chainGroup := 1
	currentChainPosition := 0
	inChain := false
	
	for i, fieldNameWithChain := range fieldNamesWithChain {
		fieldName := strings.TrimSuffix(fieldNameWithChain, "->")
		isChained := strings.HasSuffix(fieldNameWithChain, "->")
		
		if fieldConfig, exists := fieldMap[fieldName]; exists {
			if isChained {
				// This field is chained to the next field
				if !inChain {
					// Start of a new chain
					inChain = true
					currentChainPosition = 0
				}
				fieldConfig.ChainGroup = chainGroup
				fieldConfig.ChainPosition = currentChainPosition
				// Find next field in chain
				if i+1 < len(fieldNamesWithChain) {
					nextFieldName := strings.TrimSuffix(fieldNamesWithChain[i+1], "->")
					if nextConfig, exists := fieldMap[nextFieldName]; exists {
						fieldConfig.NextInChain = nextConfig.Name
					}
				}
				currentChainPosition++
			} else {
				// No arrow - check if we're continuing a chain
				if inChain {
					// This is the last field in the current chain
					fieldConfig.ChainGroup = chainGroup
					fieldConfig.ChainPosition = currentChainPosition
					fieldConfig.NextInChain = "" // Last in chain
					// End of chain
					chainGroup++
					inChain = false
					currentChainPosition = 0
				} else {
					// Not part of any chain
					fieldConfig.ChainGroup = 0
					fieldConfig.ChainPosition = 0
					fieldConfig.NextInChain = ""
				}
			}
		}
	}

	return fields
}

// GetCategoryFields returns the ordered list of fields for a category (uses cache)
func GetCategoryFields(adCategoryID int) ([]FieldConfig, error) {
	fieldCacheMutex.RLock()
	defer fieldCacheMutex.RUnlock()

	fields, exists := categoryFieldsCache[adCategoryID]
	if !exists {
		return []FieldConfig{}, nil
	}

	// Return a copy to prevent external modification
	result := make([]FieldConfig, len(fields))
	copy(result, fields)
	return result, nil
}

// GetFieldDefinition returns the field definition for a field name (uses cache)
func GetFieldDefinition(fieldName string) (FieldConfig, bool) {
	fieldCacheMutex.RLock()
	defer fieldCacheMutex.RUnlock()

	field, exists := fieldDefinitionsCache[fieldName]
	return field, exists
}

// HasField returns true if the category has the specified field
func HasField(adCategoryID int, fieldName string) bool {
	fields, err := GetCategoryFields(adCategoryID)
	if err != nil {
		return false
	}

	for _, field := range fields {
		if field.Name == fieldName {
			return true
		}
	}
	return false
}

