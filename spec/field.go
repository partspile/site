package spec

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// FieldConfig represents a specification field configuration
type FieldConfig struct {
	ID              int
	Name            string
	DisplayName     string
	FieldType       string // 'string', 'int', 'float64'
	TableName       string // Table name (e.g., 'Make', 'Year', 'Model'). Empty for fields that don't use database tables (e.g., 'location', 'price')
	TableFieldName  string // Column name in Specification table (e.g., 'make_id', 'year_id')
	ValueColumnName string // Column name in the foreign table for the value (e.g., 'name', 'year')
	QdrantType      string // 'keyword', 'keyword_array', 'integer', 'float', 'geo'
	FieldMultiValue bool   // Whether this field can have multiple values
	IsRequired      bool
	Order           int    // Priority order (from priority_order column) - used for display and determining chains
	NextInChain     int    // Priority order of next field in chain (0 = last field in chain or standalone)
	NextFieldName   string // Name of next field in chain (populated during load) - empty if last in chain
	IsFirstInChain  bool   // Whether this field is the first in its chain (populated during load)
	ShowInFilters   bool   // Whether this field should be shown in filter UI
}

// LocationResolver is a function type for resolving location text to coordinates
type LocationResolver func(locationText string) (latitude, longitude float64, err error)

// Field represents a field that can be used in forms, filters, and validation
type Field interface {
	// Name returns the field name
	Name() string

	// Config returns the underlying FieldConfig
	// TODO: Consider removing this and adding specific getter methods as needed
	Config() FieldConfig

	// BuildFilterCondition builds a Qdrant filter condition from form values
	// Returns nil if no filter should be applied
	// locationResolver is optional and only needed for geo fields
	BuildFilterCondition(c *fiber.Ctx, locationResolver LocationResolver) *qdrant.Condition

	// RenderNewAdField renders the field for the new ad form
	// selectedValues is a map of selected values (for efficient lookup)
	// fieldValues are the available values to choose from
	// defaultValue is the default value for single-select fields
	RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node

	// RenderFilterField renders the field for the filter dialog
	RenderFilterField(params map[string]string) g.Node

	// Validate validates a field value
	Validate(value string) error

	// BuildJoinClause builds the SQL JOIN clause for this field
	// Returns empty string if field doesn't use a database table
	BuildJoinClause() string

	// BuildSelectClause builds the SQL SELECT clause for this field
	// Returns empty string if field doesn't use a database table
	BuildSelectClause() string

	// IsFirstInChain returns true if the field is the first field in its chain
	IsFirstInChain() bool

	// GetValues returns the available values for this field given parent filters
	// parentValues maps parent field names to their values
	GetValues(parentValues map[string]string) ([]string, error)
}

// BaseField provides common functionality for all field types
type BaseField struct {
	config     FieldConfig
	categoryID int // The ad category ID this field belongs to
}

func (b *BaseField) Name() string {
	return b.config.Name
}

func (b *BaseField) Config() FieldConfig {
	return b.config
}

func (b *BaseField) BuildJoinClause() string {
	config := b.config
	if config.TableName == "" {
		return ""
	}
	return fmt.Sprintf("LEFT JOIN %s ON spec.%s = %s.id",
		config.TableName, config.TableFieldName, config.TableName)
}

func (b *BaseField) BuildSelectClause() string {
	config := b.config
	if config.TableName == "" {
		return ""
	}
	return fmt.Sprintf("COALESCE(GROUP_CONCAT(DISTINCT CAST(%s.%s AS TEXT)), '') as %s",
		config.TableName, config.ValueColumnName, config.Name)
}

func (b *BaseField) IsFirstInChain() bool {
	return b.config.IsFirstInChain
}

func (b *BaseField) GetValues(parentValues map[string]string) ([]string, error) {
	// Get the field from the category to ensure we have a full Field implementation
	field, err := GetCategoryField(b.categoryID, b.config.Name)
	if err != nil {
		return nil, fmt.Errorf("field not found: %s", b.config.Name)
	}

	query, args, err := BuildFieldQueryWithIntersection(b.categoryID, field, parentValues)
	if err != nil {
		return nil, err
	}

	var values []string
	err = db.Select(&values, query, args...)
	return values, err
}

// RenderNewAdField provides a generic fallback implementation
// Concrete field types should override this for field-specific rendering
func (b *BaseField) RenderNewAdField(selectedValues map[string]bool, fieldValues []string, defaultValue string) g.Node {
	config := b.config
	fieldID := config.Name + "Div"
	fieldName := config.Name
	nextFieldName := config.NextFieldName
	nextFieldTarget := ""
	if nextFieldName != "" {
		nextFieldTarget = nextFieldName + "Div"
	}

	if config.FieldMultiValue {
		// Multi-select (checkboxes)
		selected := selectedValues
		if selected == nil {
			selected = make(map[string]bool)
		}

		checkboxes := []g.Node{}
		for _, value := range fieldValues {
			attrs := []g.Node{
				Type("checkbox"),
				Name(fieldName),
				Value(value),
				ID(fieldName + "-" + value),
			}
			if selected[value] {
				attrs = append(attrs, Checked())
			}

			// Add HTMX trigger if there's a next field
			if nextFieldTarget != "" && nextFieldName != "" {
				endpoint := fmt.Sprintf("/api/%s", nextFieldName)
				attrs = append(attrs,
					hx.Trigger("change"),
					hx.Get(endpoint),
					hx.Target(fmt.Sprintf("#%s", nextFieldTarget)),
					hx.Include("form"),
					hx.Swap("innerHTML"),
				)
			}

			checkboxNode := Div(
				Class("flex items-center space-x-2"),
				Input(attrs...),
				Label(
					For(fieldName+"-"+value),
					g.Text(value),
				),
			)
			checkboxes = append(checkboxes, checkboxNode)
		}

		gridContainer := Div(
			Class("grid grid-cols-4 gap-2 p-4 bg-gray-50 border border-gray-200 rounded-lg"),
			g.Group(checkboxes),
		)

		formGroup := Div(
			Class("space-y-2"),
			Label(For(fieldName), Class("block"), g.Text(config.DisplayName)),
			gridContainer,
		)

		return Div(
			ID(fieldID),
			Class("space-y-2"),
			formGroup,
		)
	} else {
		// Single-select (dropdown)
		options := []g.Node{
			Option(Value(""), g.Text(fmt.Sprintf("Select a %s", config.DisplayName))),
		}

		for _, value := range fieldValues {
			optionAttrs := []g.Node{Value(value), g.Text(value)}
			if value == defaultValue {
				optionAttrs = append(optionAttrs, Selected())
			}
			options = append(options, Option(optionAttrs...))
		}

		attrs := []g.Node{
			ID(fieldName),
			Name(fieldName),
			Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
		}

		if config.IsRequired {
			attrs = append(attrs, Required())
		}

		// Add HTMX trigger if there's a next field in chain
		if nextFieldTarget != "" && nextFieldName != "" {
			endpoint := fmt.Sprintf("/api/%s", nextFieldName)
			attrs = append(attrs,
				hx.Trigger("change"),
				hx.Get(endpoint),
				hx.Target(fmt.Sprintf("#%s", nextFieldTarget)),
				hx.Include("form"),
				hx.Swap("innerHTML"),
			)
		}

		formGroup := Div(
			Class("space-y-2"),
			Label(For(fieldName), Class("block"), g.Text(config.DisplayName)),
			Select(append(attrs, g.Group(options)...)...),
		)

		return Div(
			ID(fieldID),
			formGroup,
		)
	}
}

// NewField creates a Field instance based on the field name
func NewField(config FieldConfig, categoryID int) (Field, error) {
	base := BaseField{config: config, categoryID: categoryID}

	switch config.Name {
	case "year":
		return &FieldYear{BaseField: base}, nil
	case "make":
		return &FieldMake{BaseField: base}, nil
	case "model":
		return &FieldModel{BaseField: base}, nil
	case "engine":
		return &FieldEngine{BaseField: base}, nil
	case "part_category":
		return &FieldPartCategory{BaseField: base}, nil
	case "part_subcategory":
		return &FieldPartSubcategory{BaseField: base}, nil
	case "breed":
		return &FieldBreed{BaseField: base}, nil
	case "location":
		return &FieldLocation{BaseField: base}, nil
	case "price":
		return &FieldPrice{BaseField: base}, nil
	default:
		return nil, fmt.Errorf("unknown field name: %s", config.Name)
	}
}

var (
	// In-memory caches for field configurations
	categoryFieldsCache   map[int][]Field
	dbFieldsCache         map[int][]Field
	fieldDefinitionsCache map[string]FieldConfig
)

// InitFieldCache loads SpecificationField and AdCategoryField into memory maps at startup
func InitFieldCache() error {
	// Initialize maps
	categoryFieldsCache = make(map[int][]Field)
	dbFieldsCache = make(map[int][]Field)
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
		SELECT id, name, display_name, field_type, table_name, table_field_name, value_column_name, qdrant_type
		FROM SpecificationField
		ORDER BY id
	`

	type dbField struct {
		ID              int    `db:"id"`
		Name            string `db:"name"`
		DisplayName     string `db:"display_name"`
		FieldType       string `db:"field_type"`
		TableName       string `db:"table_name"`
		TableFieldName  string `db:"table_field_name"`
		ValueColumnName string `db:"value_column_name"`
		QdrantType      string `db:"qdrant_type"`
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
			ValueColumnName: f.ValueColumnName,
			QdrantType:      f.QdrantType,
			FieldMultiValue: f.QdrantType == "keyword_array",
		}
	}

	return fieldDefs, nil
}

// loadCategoryFields loads all AdCategoryField records and builds category -> fields mapping
// Chain information is determined by order and next_in_chain fields
func loadCategoryFields(fieldDefs map[string]FieldConfig) (map[int][]Field, error) {
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
	categoryFieldsMap := make(map[int][]Field)
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

		// Create Field instance from FieldConfig
		field, err := NewField(fieldConfig, cf.AdCategoryID)
		if err != nil {
			logger.Warn("Failed to create field instance",
				"component", "spec-field",
				"field_name", fieldConfig.Name,
				"ad_category_id", cf.AdCategoryID,
				"error", err)
			continue
		}
		categoryFieldsMap[cf.AdCategoryID] = append(
			categoryFieldsMap[cf.AdCategoryID],
			field,
		)
	}

	// Sort fields by order within each category
	// Also build DB fields cache (fields with database tables)
	// And populate NextFieldName for each field
	dbFieldsMap := make(map[int][]Field)
	for catID := range categoryFieldsMap {
		fields := categoryFieldsMap[catID]
		sort.Slice(fields, func(i, j int) bool {
			return fields[i].Config().Order < fields[j].Config().Order
		})

		// Build order -> field name map for looking up next field names
		orderToName := make(map[int]string)
		for _, field := range fields {
			orderToName[field.Config().Order] = field.Name()
		}

		// Populate NextFieldName and IsFirstInChain for each field
		for i := range fields {
			config := fields[i].Config()

			// Set NextFieldName if there's a next field in chain
			if config.NextInChain != 0 {
				if nextName, exists := orderToName[config.NextInChain]; exists {
					config.NextFieldName = nextName
				}
			}

			// Determine if this field is first in chain
			// A field is first if no other field's NextInChain points to its Order
			isFirst := true
			if config.NextInChain == 0 {
				// Standalone field (chain of length 1) - it's first
				isFirst = true
			} else {
				// Check if any other field points to this field's order
				for _, otherField := range fields {
					if otherField.Config().NextInChain == config.Order {
						isFirst = false
						break
					}
				}
			}
			config.IsFirstInChain = isFirst

			// Recreate the field with updated config
			field, err := NewField(config, catID)
			if err == nil {
				fields[i] = field
			}
		}

		categoryFieldsMap[catID] = fields

		// Build DB fields cache for this category
		var dbFields []Field
		for _, field := range fields {
			config := field.Config()
			if config.TableName != "" {
				dbFields = append(dbFields, field)
			}
		}
		dbFieldsMap[catID] = dbFields
	}

	// Populate DB fields cache
	maps.Copy(dbFieldsCache, dbFieldsMap)

	return categoryFieldsMap, nil
}

// GetCategoryFields returns the ordered list of fields for a category (uses cache)
func GetCategoryFields(adCategoryID int) []Field {
	fields, exists := categoryFieldsCache[adCategoryID]
	if !exists {
		logger.Warn("No fields found for ad category",
			"component", "spec-field",
			"ad_category_id", adCategoryID)
		return []Field{}
	}

	return fields
}

// GetDBFields returns only fields that have database tables (uses cache)
// This is useful for building SQL queries that need JOINs
func GetDBFields(adCategoryID int) []Field {
	fields, exists := dbFieldsCache[adCategoryID]
	if !exists {
		logger.Warn("No DB fields found for ad category",
			"component", "spec-field",
			"ad_category_id", adCategoryID)
		return []Field{}
	}

	return fields
}

// IsFirstInChain returns true if the field is the first field in its chain
// This is a convenience wrapper around the Field method
func IsFirstInChain(field Field) bool {
	return field.IsFirstInChain()
}

// chainContains checks if a field's chain contains a target order
func chainContains(field Field, targetOrder int, orderMap map[int]Field) bool {
	current := field.Config()
	for current.NextInChain != 0 {
		if current.NextInChain == targetOrder {
			return true
		}
		nextField, exists := orderMap[current.NextInChain]
		if !exists {
			break
		}
		current = nextField.Config()
	}
	return false
}

// FieldsInSameChain returns true if two fields are in the same chain
func FieldsInSameChain(field1, field2 Field, allFields []Field) bool {
	if field1.Config().Order == field2.Config().Order {
		return true
	}

	// Build order -> field map for quick lookup
	orderMap := make(map[int]Field)
	for _, f := range allFields {
		orderMap[f.Config().Order] = f
	}

	// Check if field1's chain includes field2 or field2's chain includes field1
	return chainContains(field1, field2.Config().Order, orderMap) ||
		chainContains(field2, field1.Config().Order, orderMap)
}

// GetNextFieldName returns the name of the next field in chain, or empty string if last
func GetNextFieldName(field Field, allFields []Field) string {
	config := field.Config()
	if config.NextInChain == 0 {
		return ""
	}
	for _, f := range allFields {
		if f.Config().Order == config.NextInChain {
			return f.Config().Name
		}
	}
	return ""
}

// GetDownstreamFields returns all fields that come after this field in the same chain
func GetDownstreamFields(field Field, allFields []Field) []string {
	config := field.Config()
	if config.NextInChain == 0 {
		return []string{}
	}

	orderMap := make(map[int]Field)
	for _, f := range allFields {
		orderMap[f.Config().Order] = f
	}

	var downstream []string
	currentOrder := config.NextInChain
	for currentOrder != 0 {
		nextField, exists := orderMap[currentOrder]
		if !exists {
			break
		}
		downstream = append(downstream, nextField.Config().Name)
		currentOrder = nextField.Config().NextInChain
	}

	return downstream
}

// fieldSelect creates a reusable select dropdown field for single-select fields
// fieldID: ID for the container div (e.g., "makeDiv")
// fieldName: Name attribute for select (e.g., "make")
// displayName: Label text (e.g., "Make")
// fieldValues: Available values to display as options
// defaultValue: Default selected value (empty string if none)
// isRequired: Whether the field is required
// nextFieldName: Optional next field name for HTMX chain (empty string if none)
func fieldSelect(fieldID, fieldName, displayName string, fieldValues []string, defaultValue string, isRequired bool, nextFieldName string) g.Node {
	options := []g.Node{
		Option(Value(""), g.Text(fmt.Sprintf("Select a %s", displayName))),
	}

	for _, value := range fieldValues {
		optionAttrs := []g.Node{Value(value), g.Text(value)}
		if value == defaultValue {
			optionAttrs = append(optionAttrs, Selected())
		}
		options = append(options, Option(optionAttrs...))
	}

	attrs := []g.Node{
		ID(fieldName),
		Name(fieldName),
		Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
	}

	if isRequired {
		attrs = append(attrs, Required())
	}

	// Add HTMX trigger if there's a next field in chain
	nextFieldTarget := ""
	if nextFieldName != "" {
		nextFieldTarget = nextFieldName + "Div"
		endpoint := fmt.Sprintf("/api/%s", nextFieldName)
		attrs = append(attrs,
			hx.Trigger("change"),
			hx.Get(endpoint),
			hx.Target(fmt.Sprintf("#%s", nextFieldTarget)),
			hx.Include("form"),
			hx.Swap("innerHTML"),
		)
	}

	formGroup := Div(
		Class("space-y-2"),
		Label(For(fieldName), Class("block"), g.Text(displayName)),
		Select(append(attrs, g.Group(options)...)...),
	)

	return Div(
		ID(fieldID),
		formGroup,
	)
}

// fieldInput creates a reusable input field for text/number inputs
// fieldID: ID for the container div (e.g., "priceDiv")
// fieldName: Name attribute for input (e.g., "price")
// displayName: Label text (e.g., "Price")
// inputType: Input type (e.g., "text", "number")
// defaultValue: Default value (empty string if none)
// isRequired: Whether the field is required
// placeholder: Placeholder text (empty string if none)
// additionalAttrs: Additional attributes to add to the input
func fieldInput(fieldID, fieldName, displayName, inputType, defaultValue string, isRequired bool, placeholder string, additionalAttrs ...g.Node) g.Node {
	attrs := []g.Node{
		Type(inputType),
		ID(fieldName),
		Name(fieldName),
		Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
	}

	if defaultValue != "" {
		attrs = append(attrs, Value(defaultValue))
	}

	if isRequired {
		attrs = append(attrs, Required())
	}

	if placeholder != "" {
		attrs = append(attrs, Placeholder(placeholder))
	}

	attrs = append(attrs, additionalAttrs...)

	formGroup := Div(
		Class("space-y-2"),
		Label(For(fieldName), Class("block"), g.Text(displayName)),
		Input(attrs...),
	)

	return Div(
		ID(fieldID),
		formGroup,
	)
}

// fieldCheckboxGrid creates a reusable checkbox grid field for multi-select fields
// This is a helper function that can be used by field implementations
// fieldID: ID for the container div (e.g., "yearDiv")
// fieldName: Name attribute for checkboxes (e.g., "year")
// displayName: Label text (e.g., "Year")
// fieldValues: Available values to display as checkboxes
// selectedValues: Map of selected values (for pre-selecting)
// nextFieldName: Optional next field name for HTMX chain (empty string if none)
func fieldCheckboxGrid(fieldID, fieldName, displayName string, fieldValues []string, selectedValues map[string]bool, nextFieldName string) g.Node {
	// Create checkbox grid
	checkboxes := []g.Node{}
	for _, value := range fieldValues {
		attrs := []g.Node{
			Type("checkbox"),
			Name(fieldName),
			Value(value),
			ID(fieldName + "-" + value),
		}
		if selectedValues != nil {
			if _, ok := selectedValues[value]; ok {
				attrs = append(attrs, Checked())
			}
		}

		checkboxNode := Div(
			Class("flex items-center space-x-2"),
			Input(attrs...),
			Label(
				For(fieldName+"-"+value),
				g.Text(value),
			),
		)
		checkboxes = append(checkboxes, checkboxNode)
	}

	// Grid container for checkboxes
	// Add HTMX trigger on parent to listen for changes from any checkbox
	gridAttrs := []g.Node{
		Class("grid grid-cols-4 gap-2 p-4 bg-gray-50 border border-gray-200 rounded-lg"),
	}
	if nextFieldName != "" {
		endpoint := fmt.Sprintf("/api/%s", nextFieldName)
		gridAttrs = append(gridAttrs,
			hx.Trigger("change from:input"),
			hx.Get(endpoint),
			hx.Target(fmt.Sprintf("#%s", nextFieldName+"Div")),
			hx.Include("form"),
			hx.Swap("innerHTML"),
		)
	}

	gridContainer := Div(
		append(gridAttrs, g.Group(checkboxes)...)...,
	)

	// Form group wrapper
	formGroup := Div(
		Class("space-y-2"),
		Label(For(fieldName), Class("block"), g.Text(displayName)),
		gridContainer,
	)

	return Div(
		ID(fieldID),
		Class("space-y-2"),
		formGroup,
	)
}

// GetFieldDefinition returns the field definition for a field name
func GetFieldDefinition(fieldName string) (FieldConfig, bool) {
	field, exists := fieldDefinitionsCache[fieldName]
	return field, exists
}

// GetCategoryField returns the field for a specific field in a category
func GetCategoryField(adCategoryID int, fieldName string) (Field, error) {
	fields := GetCategoryFields(adCategoryID)

	for _, field := range fields {
		if field.Config().Name == fieldName {
			return field, nil
		}
	}

	return nil, fmt.Errorf("field %s not found in category %d", fieldName, adCategoryID)
}

// BuildFieldPromptSnippet generates a prompt snippet for a field given its values
// Returns empty string if field has no values
func BuildFieldPromptSnippet(field Field, fieldValues []string) string {
	if len(fieldValues) == 0 {
		return ""
	}

	fieldConfig := field.Config()

	// Format field values
	valueStr := strings.Join(fieldValues, ", ")
	if valueStr == "" {
		return ""
	}

	// Special handling for "make" field - include parent company info
	if fieldConfig.Name == "make" {
		make := fieldValues[0]
		snippet := fmt.Sprintf("%s: %s\n", fieldConfig.DisplayName, make)

		// Get parent company information
		if pcInfo, err := GetParentCompanyInfoForMake(make); err == nil && pcInfo != nil {
			if pcInfo.Name != "" {
				snippet += fmt.Sprintf("Parent Company: %s\n", pcInfo.Name)
			}
			if pcInfo.Country != "" {
				snippet += fmt.Sprintf("Parent Company Country: %s\n", pcInfo.Country)
			}
		}

		return snippet
	}

	// Default: simple field name and value
	return fmt.Sprintf("%s: %s\n", fieldConfig.DisplayName, valueStr)
}
