package spec

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
)

// FieldConfig represents a specification field configuration
type FieldConfig struct {
	ID              int
	Name            string
	DisplayName     string
	FieldType       string // 'string', 'int', 'float64'
	TableName       string // Always required - table name (e.g., 'Make', 'Year', 'Model')
	TableFieldName  string // Column name in Specification table (e.g., 'make_id', 'year_id')
	QdrantType      string // 'keyword', 'keyword_array', 'integer', 'float', 'geo'
	FieldMultiValue bool   // Whether this field can have multiple values
	IsRequired      bool
	Order           int  // Priority order (from priority_order column) - used for display and determining chains
	NextInChain     int  // Priority order of next field in chain (0 = last field in chain or standalone)
	ShowInFilters   bool // Whether this field should be shown in filter UI
}

var (
	// In-memory caches for field configurations
	categoryFieldsCache   map[int][]FieldConfig
	fieldDefinitionsCache map[string]FieldConfig
)

// InitFieldCache loads SpecificationField and AdCategoryField into memory maps at startup
func InitFieldCache() error {
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
	maps.Copy(fieldDefinitionsCache, fieldDefs)
	maps.Copy(categoryFieldsCache, categoryFields)

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
		ID             int    `db:"id"`
		Name           string `db:"name"`
		DisplayName    string `db:"display_name"`
		FieldType      string `db:"field_type"`
		TableName      string `db:"table_name"`
		TableFieldName string `db:"table_field_name"`
		QdrantType     string `db:"qdrant_type"`
	}

	var dbFields []dbField
	err := db.Select(&dbFields, query)
	if err != nil {
		return nil, err
	}

	fieldDefs := make(map[string]FieldConfig)
	for _, f := range dbFields {
		fieldDefs[f.Name] = FieldConfig{
			ID:              f.ID,
			Name:            f.Name,
			DisplayName:     f.DisplayName,
			FieldType:       f.FieldType,
			TableName:       f.TableName,
			TableFieldName:  f.TableFieldName,
			QdrantType:      f.QdrantType,
			FieldMultiValue: f.QdrantType == "keyword_array",
		}
	}

	return fieldDefs, nil
}

// loadCategoryFields loads all AdCategoryField records and builds category -> fields mapping
// Chain information is determined by order and next_in_chain fields
func loadCategoryFields(fieldDefs map[string]FieldConfig) (map[int][]FieldConfig, error) {
	query := `
		SELECT acf.ad_category_id, acf.specification_field_id, acf.priority_order, acf.next_in_chain, acf.is_required, acf.show_in_filters
		FROM AdCategoryField acf
		JOIN SpecificationField sf ON acf.specification_field_id = sf.id
		ORDER BY acf.ad_category_id, acf.priority_order
	`

	type dbCategoryField struct {
		AdCategoryID         int `db:"ad_category_id"`
		SpecificationFieldID int `db:"specification_field_id"`
		Order                int `db:"priority_order"`
		NextInChain          int `db:"next_in_chain"`
		IsRequired           int `db:"is_required"`
		ShowInFilters        int `db:"show_in_filters"`
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
		fieldConfig.Order = cf.Order
		fieldConfig.NextInChain = cf.NextInChain
		fieldConfig.ShowInFilters = cf.ShowInFilters != 0
		categoryFieldsMap[cf.AdCategoryID] = append(
			categoryFieldsMap[cf.AdCategoryID],
			fieldConfig,
		)
	}

	// Sort fields by order within each category
	for catID := range categoryFieldsMap {
		fields := categoryFieldsMap[catID]
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Order < fields[j].Order
		})
		categoryFieldsMap[catID] = fields
	}

	return categoryFieldsMap, nil
}

// GetCategoryFields returns the ordered list of fields for a category (uses cache)
func GetCategoryFields(adCategoryID int) []FieldConfig {
	fields, exists := categoryFieldsCache[adCategoryID]
	if !exists {
		logger.Warn("No fields found for ad category",
			"component", "spec-field",
			"ad_category_id", adCategoryID)
		return []FieldConfig{}
	}

	// Return a copy to prevent external modification
	result := make([]FieldConfig, len(fields))
	copy(result, fields)
	return result
}

// GetSpecTableName returns the spec table name for a given spec type
func GetSpecTableName(specType string) string {
	switch specType {
	case "vehicle":
		return "SpecVehicle"
	case "agricultural":
		return "SpecAgricultural"
	case "bicycle":
		return "SpecBicycle"
	case "part":
		return "SpecPart"
	default:
		return "SpecVehicle" // Default fallback (matches GetSpecTypeFromChain default)
	}
}

// GetSpecTypeFromChain determines the spec type based on chain fields
func GetSpecTypeFromChain(chain []FieldConfig) string {
	if len(chain) == 0 {
		return ""
	}

	// Check if it's a part chain
	if chain[0].Name == "part_category" {
		return "part"
	}

	// Determine vehicle chain type based on fields
	fieldNames := make([]string, len(chain))
	for i, field := range chain {
		fieldNames[i] = field.Name
	}

	// Check for 4-field vehicle chain (make->year->model->engine)
	if len(fieldNames) == 4 && fieldNames[0] == "make" && fieldNames[1] == "year" &&
		fieldNames[2] == "model" && fieldNames[3] == "engine" {
		return "vehicle"
	}

	// Check for 3-field vehicle chain (make->year->model)
	if len(fieldNames) == 3 && fieldNames[0] == "make" && fieldNames[1] == "year" &&
		fieldNames[2] == "model" {
		return "agricultural"
	}

	// Check for 2-field vehicle chain (make->model)
	if len(fieldNames) == 2 && fieldNames[0] == "make" && fieldNames[1] == "model" {
		return "bicycle"
	}

	// Default to vehicle for unknown patterns
	return "vehicle"
}

// IsFirstInChain returns true if the field is the first field in its chain
// A field is first if no other field's NextInChain points to its Order
func IsFirstInChain(field FieldConfig, allFields []FieldConfig) bool {
	if field.NextInChain == 0 {
		// Standalone field (chain of length 1) - it's first
		return true
	}
	// Check if any other field points to this field's order
	for _, f := range allFields {
		if f.NextInChain == field.Order {
			return false
		}
	}
	return true
}

// chainContains checks if a field's chain contains a target order
func chainContains(field FieldConfig, targetOrder int, orderMap map[int]FieldConfig) bool {
	current := field
	for current.NextInChain != 0 {
		if current.NextInChain == targetOrder {
			return true
		}
		nextField, exists := orderMap[current.NextInChain]
		if !exists {
			break
		}
		current = nextField
	}
	return false
}

// FieldsInSameChain returns true if two fields are in the same chain
func FieldsInSameChain(field1, field2 FieldConfig, allFields []FieldConfig) bool {
	if field1.Order == field2.Order {
		return true
	}

	// Build order -> field map for quick lookup
	orderMap := make(map[int]FieldConfig)
	for _, f := range allFields {
		orderMap[f.Order] = f
	}

	// Check if field1's chain includes field2 or field2's chain includes field1
	return chainContains(field1, field2.Order, orderMap) ||
		chainContains(field2, field1.Order, orderMap)
}

// GetNextFieldName returns the name of the next field in chain, or empty string if last
func GetNextFieldName(field FieldConfig, allFields []FieldConfig) string {
	if field.NextInChain == 0 {
		return ""
	}
	for _, f := range allFields {
		if f.Order == field.NextInChain {
			return f.Name
		}
	}
	return ""
}

// GetDownstreamFields returns all fields that come after this field in the same chain
func GetDownstreamFields(field FieldConfig, allFields []FieldConfig) []string {
	if field.NextInChain == 0 {
		return []string{}
	}

	orderMap := make(map[int]FieldConfig)
	for _, f := range allFields {
		orderMap[f.Order] = f
	}

	var downstream []string
	currentOrder := field.NextInChain
	for currentOrder != 0 {
		nextField, exists := orderMap[currentOrder]
		if !exists {
			break
		}
		downstream = append(downstream, nextField.Name)
		currentOrder = nextField.NextInChain
	}

	return downstream
}

// GetFieldDefinition returns the field definition for a field name
func GetFieldDefinition(fieldName string) (FieldConfig, bool) {
	field, exists := fieldDefinitionsCache[fieldName]
	return field, exists
}

// GetCategoryField returns the field configuration for a specific field in a category
func GetCategoryField(adCategoryID int, fieldName string) (FieldConfig, error) {
	fieldConfigs := GetCategoryFields(adCategoryID)

	for _, fieldConfig := range fieldConfigs {
		if fieldConfig.Name == fieldName {
			return fieldConfig, nil
		}
	}

	return FieldConfig{}, fmt.Errorf("field %s not found in category %d", fieldName, adCategoryID)
}

// BuildFieldPromptSnippet generates a prompt snippet for a field given its values
// Returns empty string if field has no values
func BuildFieldPromptSnippet(fieldConfig FieldConfig, fieldValues []string) string {
	if len(fieldValues) == 0 {
		return ""
	}

	// Format field values
	valueStr := strings.Join(fieldValues, ", ")
	if valueStr == "" {
		return ""
	}

	// Default: simple field name and value
	return fmt.Sprintf("%s: %s\n", fieldConfig.DisplayName, valueStr)
}

// GetAllChains returns all field chains for a given category
// Returns a map: chain start order -> []FieldConfig
// Uses the cached field configuration
func GetAllChains(adCategoryID int) (map[int][]FieldConfig, error) {
	allFields := GetCategoryFields(adCategoryID)
	if len(allFields) == 0 {
		return nil, fmt.Errorf("no fields found for category %d", adCategoryID)
	}

	// Build order -> field map
	orderMap := make(map[int]FieldConfig)
	for _, f := range allFields {
		orderMap[f.Order] = f
	}

	// Find all chain starts (fields that are not pointed to by any other field's next_in_chain)
	chainStarts := make(map[int]bool)
	for _, f := range allFields {
		chainStarts[f.Order] = true
	}
	for _, f := range allFields {
		if f.NextInChain != 0 {
			delete(chainStarts, f.NextInChain)
		}
	}

	// Build all chains
	chains := make(map[int][]FieldConfig)
	for startOrder := range chainStarts {
		var chainFields []FieldConfig
		currentOrder := startOrder
		for {
			field, exists := orderMap[currentOrder]
			if !exists {
				break
			}
			chainFields = append(chainFields, field)
			if field.NextInChain == 0 {
				break // End of chain
			}
			currentOrder = field.NextInChain
		}
		if len(chainFields) > 0 {
			chains[startOrder] = chainFields
		}
	}

	return chains, nil
}

// GetChainStartingWith returns the chain that starts with the given field name
// Returns nil if not found
// Uses the cached field configuration
func GetChainStartingWith(adCategoryID int, fieldName string) ([]FieldConfig, error) {
	chains, err := GetAllChains(adCategoryID)
	if err != nil {
		return nil, err
	}

	for _, chain := range chains {
		if len(chain) > 0 && chain[0].Name == fieldName {
			return chain, nil
		}
	}

	return nil, nil
}
