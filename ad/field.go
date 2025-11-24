package ad

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/db"
	"github.com/qdrant/go-client/qdrant"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// Field represents a field that can be used in forms, filters, and validation
type Field interface {
	// Name returns the name of the field
	Name() string

	// BuildFilterCondition builds a Qdrant filter condition from form values
	// Returns nil if no filter should be applied
	BuildFilterCondition(c *fiber.Ctx) *qdrant.Condition

	// RenderNewAdField renders the field for the new ad form
	// parentValues maps parent field names to their values (for filtering options)
	RenderNewAdField(parentValues map[string]string) g.Node

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

	// GetValues returns the available values for this field given parent filters
	// parentValues maps parent field names to their values
	GetValues(parentValues map[string]string) ([]string, error)

	// BuildSpecDataSubquery builds the SQL JSON key-value pair for fetching spec data values for an ad
	// Returns empty string if field is not a spec field
	BuildSpecDataSubquery() string

	// IsSpecField returns true if the field is a spec field
	IsSpecField() bool

	// GetTableNames returns all table-related names for the field
	// Returns: specTableName, specTableFieldName, adSpecTableName, tableName
	GetTableNames() (string, string, string, string)

	// IsMultiValue returns true if the field can have multiple values
	IsMultiValue() bool

	// Required returns true if the field is required
	Required() bool

	// NextField returns the name of the next field in the chain, or empty string if none
	NextField() string

	// GetQdrantType returns the Qdrant type string for this field
	// Returns: 'keyword', 'keyword_array', 'integer', 'float', or 'geo'
	GetQdrantType() string

	// EmbeddingPrompt returns a formatted prompt snippet for embedding generation
	// fieldValues is a slice of field values (single or multiple)
	// Returns empty string if no values or field should not be included
	EmbeddingPrompt(fieldValues []string) string
}

type adField struct {
	AdCategoryID       int      `json:"ad_category_id"`        // Ad category ID
	FieldName          string   `json:"name"`                  // 'make'
	DisplayName        string   `json:"display_name"`          // 'Make'
	FieldType          string   `json:"field_type"`            // 'string', 'int', 'float64'
	TableName          string   `json:"table_name"`            // 'Make'
	QdrantType         string   `json:"qdrant_type"`           // 'keyword', 'keyword_array', 'integer', 'float', 'geo'
	NextInChain        int      `json:"next_in_chain"`         // 0-based array index of next field in chain (0 = last field in chain or standalone, > 0 = index of next field)
	NextFieldName      string   `json:"next_field_name"`       // 'year'
	IsRequired         bool     `json:"is_required"`           // Is required field on new ad form
	ShowInFilters      bool     `json:"show_in_filters"`       // Show field in filter UI
	SpecTableName      string   `json:"spec_table_name"`       // 'SpecVehicle'
	SpecTableFieldName string   `json:"spec_table_field_name"` // 'make_id'
	AdSpecTableName    string   `json:"ad_spec_table_name"`    // 'AdSpecVehicle'
	IsMultiValueField  bool     `json:"is_multi_value_field"`  // True if multi-value field
	ParentField        *adField `json:"-"`                     // Backlink to parent field in chain (populated during initialization)
}

func (f adField) Name() string {
	return f.FieldName
}

func (f adField) IsSpecField() bool {
	return f.SpecTableName != ""
}

func (f adField) GetTableNames() (string, string, string, string) {
	return f.SpecTableName, f.SpecTableFieldName, f.AdSpecTableName, f.TableName
}

func (f adField) IsMultiValue() bool {
	return f.IsMultiValueField
}

func (f adField) Required() bool {
	return f.IsRequired
}

func (f adField) NextField() string {
	return f.NextFieldName
}

func (f adField) GetQdrantType() string {
	return f.QdrantType
}

func (f adField) EmbeddingPrompt(fieldValues []string) string {
	if len(fieldValues) == 0 {
		return ""
	}

	if len(fieldValues) == 1 {
		return fmt.Sprintf("%s: %s", f.DisplayName, fieldValues[0])
	}

	// Multiple values - join with comma
	return fmt.Sprintf("%s: %s", f.DisplayName, strings.Join(fieldValues, ", "))
}

func (f adField) BuildJoinClause() string {
	if f.TableName == "" {
		return ""
	}
	return fmt.Sprintf("LEFT JOIN %s ON spec.value = %s.id",
		f.TableName, f.TableName)
}

func (f adField) BuildSelectClause() string {
	if f.TableName == "" {
		return ""
	}
	return fmt.Sprintf("COALESCE(STRING_AGG(DISTINCT CAST(%s.value AS TEXT), ','), '') as %s",
		f.TableName, f.FieldName)
}

func (f adField) GetValues(params map[string]string) ([]string, error) {
	if f.TableName == "" {
		// For fields without a table, return empty
		return []string{}, nil
	}

	// If not a spec field, use simple query without parent filtering
	if f.SpecTableName == "" {
		return f.getSimpleFieldValues()
	}

	// For spec fields, check if we have a parent in the chain
	// If we have a parent and it's not in params or is empty, return empty results
	if f.ParentField != nil {
		parentName := f.ParentField.FieldName
		parentValue, exists := params[parentName]
		if !exists || parentValue == "" {
			// Parent field is required but not provided or empty - return empty
			return []string{}, nil
		}
	}

	// If no params at all, use simple query
	if len(params) == 0 {
		return f.getSimpleFieldValues()
	}

	// Collect parent fields from params by walking backwards through the chain
	parentFields := f.collectParentFields(params)
	if len(parentFields) == 0 {
		// No parent fields in params, fall back to simple query
		return f.getSimpleFieldValues()
	}

	// Build and execute filtered query
	return f.executeFilteredQuery(parentFields)
}

// getSimpleFieldValues returns all values for a field without parent filtering
func (f adField) getSimpleFieldValues() ([]string, error) {
	var values []string
	query := fmt.Sprintf("SELECT DISTINCT value FROM %s WHERE ad_category_id = $1 ORDER BY value", f.TableName)
	err := db.Select(&values, query, f.AdCategoryID)
	return values, err
}

// GetValuesForAdIDs returns distinct field values for the given ad IDs
func GetValuesForAdIDs(field Field, adIDs []int) ([]string, error) {
	if len(adIDs) == 0 {
		return []string{}, nil
	}

	specTableName, specTableFieldName, adSpecTableName, tableName := field.GetTableNames()
	if tableName == "" || specTableName == "" || adSpecTableName == "" {
		return []string{}, nil
	}

	// Build placeholders for ad IDs
	placeholders := make([]string, len(adIDs))
	args := make([]interface{}, len(adIDs))
	for i, id := range adIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// Query: get distinct field values from ads
	query := fmt.Sprintf(`
		SELECT DISTINCT %s.value
		FROM %s
		JOIN %s ON %s.%s = %s.id
		JOIN %s ON %s.spec_id = %s.id
		WHERE %s.ad_id IN (%s)
		ORDER BY %s.value`,
		tableName, tableName,
		specTableName, specTableName, specTableFieldName, tableName,
		adSpecTableName, adSpecTableName, specTableName,
		adSpecTableName, strings.Join(placeholders, ", "),
		tableName)

	var values []string
	err := db.Select(&values, query, args...)
	return values, err
}

// collectParentFields walks backwards through parent fields and collects those present in params
func (f adField) collectParentFields(params map[string]string) []parentFieldInfo {
	var parentFields []parentFieldInfo

	parent := f.ParentField
	for parent != nil {
		parentName := parent.FieldName
		if parentValue, exists := params[parentName]; exists && parentValue != "" {
			parentFields = append(parentFields, parentFieldInfo{
				parent: parent,
				value:  parentValue,
			})
		}
		// Get next parent by following the backlink
		parent = parent.ParentField
	}

	return parentFields
}

// executeFilteredQuery builds and executes a SQL query with parent field filters
func (f adField) executeFilteredQuery(parentFields []parentFieldInfo) ([]string, error) {
	query, args := f.buildFilteredQuery(parentFields)

	var values []string
	err := db.Select(&values, query, args...)
	return values, err
}

// buildFilteredQuery constructs the SQL query with JOINs and WHERE clauses for parent filters
// For multi-value parent fields, uses intersection (AND) logic to ensure values match ALL selected parent values
func (f adField) buildFilteredQuery(parentFields []parentFieldInfo) (string, []interface{}) {
	var whereClauses []string
	var args []interface{}

	// Base query: select distinct values from current field table
	query := fmt.Sprintf("SELECT DISTINCT %s.value", f.TableName)

	// FROM and JOIN spec table
	query += fmt.Sprintf(" FROM %s", f.TableName)
	query += fmt.Sprintf(" JOIN %s ON %s.%s = %s.id",
		f.SpecTableName, f.SpecTableName, f.SpecTableFieldName, f.TableName)

	// Add WHERE clause for category
	args = append(args, f.AdCategoryID)
	whereClauses = append(whereClauses, fmt.Sprintf("%s.ad_category_id = $%d", f.SpecTableName, len(args)))

	// Track multi-value parent fields that need intersection logic
	var multiValueParents []parentFieldInfo
	var singleValueParents []parentFieldInfo

	// Separate multi-value and single-value parents
	for _, pf := range parentFields {
		if pf.parent.IsMultiValueField {
			multiValueParents = append(multiValueParents, pf)
		} else {
			singleValueParents = append(singleValueParents, pf)
		}
	}

	// Add JOINs and WHERE clauses for single-value parent fields (simple case)
	for _, pf := range singleValueParents {
		// JOIN parent field table through the same spec table
		query += fmt.Sprintf(" JOIN %s ON %s.%s = %s.id",
			pf.parent.TableName, f.SpecTableName, pf.parent.SpecTableFieldName, pf.parent.TableName)

		// Add WHERE clause for parent field value
		args = append(args, pf.value)
		whereClauses = append(whereClauses, fmt.Sprintf("%s.value = $%d", pf.parent.TableName, len(args)))
	}

	// For multi-value parent fields, use intersection logic
	// We need to ensure that for each selected value, there exists a DIFFERENT spec with that value
	// Each spec row has only one value for the multi-value parent field (e.g., one year per spec)
	// So we need to check: for model X, does there exist a spec with make=AC AND year=1947 AND model=X?
	// AND does there exist a spec with make=AC AND year=1948 AND model=X?
	for _, pf := range multiValueParents {
		values := strings.Split(pf.value, ",")
		if len(values) == 0 {
			continue
		}

		// For intersection, we need to ensure the current field value appears in DIFFERENT specs
		// that each have one of the selected parent values
		// Each EXISTS checks: there exists a spec with the current model AND the specific year
		subqueryParts := []string{}
		for i, val := range values {
			args = append(args, val)
			argNum := len(args)
			aliasS := fmt.Sprintf("s%d", i)
			aliasP := fmt.Sprintf("p%d", i)

			// Build correlation conditions: the spec must have the same model as outer query
			// The outer query has: Model JOIN SpecVehicle ON SpecVehicle.model_id = Model.id
			// So we correlate: s0.model_id = SpecVehicle.model_id (which equals Model.id)
			modelCorrelation := fmt.Sprintf("%s.%s = %s.%s", aliasS, f.SpecTableFieldName, f.SpecTableName, f.SpecTableFieldName)

			// Also need to correlate with single-value parents (like make)
			// The outer query already JOINs Make, so we correlate: s0.make_id = SpecVehicle.make_id
			var parentCorrelations []string
			for _, svp := range singleValueParents {
				parentCorrelations = append(parentCorrelations,
					fmt.Sprintf("%s.%s = %s.%s", aliasS, svp.parent.SpecTableFieldName, f.SpecTableName, svp.parent.SpecTableFieldName))
			}

			correlationClause := modelCorrelation
			if len(parentCorrelations) > 0 {
				correlationClause += " AND " + strings.Join(parentCorrelations, " AND ")
			}

			subqueryParts = append(subqueryParts, fmt.Sprintf(
				`EXISTS (SELECT 1 FROM %s %s JOIN %s %s ON %s.%s = %s.id WHERE %s.ad_category_id = $1 AND %s AND %s.value = $%d)`,
				f.SpecTableName, aliasS,
				pf.parent.TableName, aliasP,
				aliasS, pf.parent.SpecTableFieldName, aliasP,
				aliasS,
				correlationClause,
				aliasP, argNum))
		}
		whereClauses = append(whereClauses, strings.Join(subqueryParts, " AND "))
	}

	// Add WHERE clause
	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	// Add ORDER BY
	query += fmt.Sprintf(" ORDER BY %s.value", f.TableName)

	return query, args
}

// buildParentFieldWhereClause builds a WHERE clause for a parent field filter
func (f adField) buildParentFieldWhereClause(pf parentFieldInfo, args *[]interface{}) string {
	if pf.parent.IsMultiValueField {
		// Multi-value: split comma-separated values and use IN clause
		values := strings.Split(pf.value, ",")
		var placeholders []string
		for _, v := range values {
			*args = append(*args, v)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(*args)))
		}
		return fmt.Sprintf("%s.value IN (%s)", pf.parent.TableName, strings.Join(placeholders, ", "))
	}

	// Single value
	*args = append(*args, pf.value)
	return fmt.Sprintf("%s.value = $%d", pf.parent.TableName, len(*args))
}

func (f adField) RenderFilterField(params map[string]string) g.Node {
	return g.Group{}
}

func (f adField) RenderNewAdField(parentValues map[string]string) g.Node {
	return g.Group{}
}

func (f adField) BuildSpecDataSubquery() string {
	subquery := fmt.Sprintf(`(
		SELECT json_agg(value ORDER BY value)
		FROM (
			SELECT DISTINCT %s.value
			FROM %s
			JOIN %s sv ON %s.id = sv.%s
			JOIN %s asv ON asv.spec_id = sv.id
			WHERE asv.ad_id = $1
		) AS distinct_values
	)`, f.TableName, f.TableName, f.SpecTableName, f.TableName, f.SpecTableFieldName, f.AdSpecTableName)
	return fmt.Sprintf("'%s', COALESCE(%s, '[]'::json)", f.FieldName, subquery)
}

// NewAdField creates a Field instance based on the field name
func NewAdField(adField adField) (Field, error) {
	switch adField.FieldName {
	case "location":
		return &FieldLocation{adField}, nil
	case "price":
		return &FieldPrice{adField}, nil
	case "year":
		return &FieldYear{adField}, nil
	case "make":
		return &FieldMake{adField}, nil
	case "model":
		return &FieldModel{adField}, nil
	case "engine":
		return &FieldEngine{adField}, nil
	case "part_category":
		return &FieldPartCategory{adField}, nil
	case "part_subcategory":
		return &FieldPartSubcategory{adField}, nil
	case "breed":
		return &FieldBreed{adField}, nil
	default:
		return nil, fmt.Errorf("unknown field name: %s", adField.FieldName)
	}
}

// parentFieldInfo holds information about a parent field used in filtering
type parentFieldInfo struct {
	parent *adField
	value  string
}

// TODO move these to ui pkg once ad pkg is removed from ui pkg

// fieldSelect creates a reusable select dropdown field for single-select fields
// fieldID: ID for the container div (e.g., "makeDiv")
// fieldName: Name attribute for select (e.g., "make")
// displayName: Label text (e.g., "Make")
// fieldValues: Available values to display as options
// isRequired: Whether the field is required
// nextFieldName: Optional next field name for HTMX chain (empty string if none)
func fieldSelect(fieldID, fieldName, displayName string, fieldValues []string,
	isRequired bool, nextFieldName string) g.Node {

	// If no values available, show a message
	if len(fieldValues) == 0 {
		return Div(
			ID(fieldID),
			Class("space-y-6"),
			Div(
				Class("space-y-2"),
				Label(For(fieldName), Class("block"), g.Text(displayName)),
				Div(
					Class("w-full p-2 border border-gray-300 rounded bg-gray-50 text-gray-500 text-sm"),
					g.Text("No choices available"),
				),
			),
		)
	}

	options := []g.Node{
		Option(Value(""), g.Text(fmt.Sprintf("Select a %s", displayName))),
	}

	for _, value := range fieldValues {
		options = append(options, Option(Value(value), g.Text(value)))
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
	if nextFieldName != "" {
		endpoint := fmt.Sprintf("/api/field/%s", nextFieldName)
		attrs = append(attrs,
			hx.Trigger("change"),
			hx.Get(endpoint),
			hx.Target(fmt.Sprintf("#%s", nextFieldName+"Div")),
			hx.Include("form"),
			hx.Swap("innerHTML"),
		)
	}

	formGroup := Div(
		Class("space-y-2"),
		Label(For(fieldName), Class("block"), g.Text(displayName)),
		Select(append(attrs, g.Group(options)...)...),
	)

	containerChildren := []g.Node{formGroup}
	if nextFieldName != "" {
		// Add empty placeholder div for next field in chain
		containerChildren = append(containerChildren,
			Div(ID(nextFieldName+"Div")))
	}

	return Div(
		ID(fieldID),
		Class("space-y-6"),
		g.Group(containerChildren),
	)
}

// fieldInput creates a reusable input field for text/number inputs
// fieldID: ID for the container div (e.g., "priceDiv")
// fieldName: Name attribute for input (e.g., "price")
// displayName: Label text (e.g., "Price")
// inputType: Input type (e.g., "text", "number")
// isRequired: Whether the field is required
// placeholder: Placeholder text (empty string if none)
// additionalAttrs: Additional attributes to add to the input
func fieldInput(fieldID, fieldName, displayName, inputType string, isRequired bool,
	placeholder string, additionalAttrs ...g.Node) g.Node {

	attrs := []g.Node{
		Type(inputType),
		ID(fieldName),
		Name(fieldName),
		Class("w-full p-2 border rounded invalid:border-red-500 valid:border-emerald-500"),
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
		Class("space-y-6"),
		formGroup,
	)
}

// fieldCheckboxGrid creates a reusable checkbox grid field for multi-select fields
// This is a helper function that can be used by field implementations
// fieldID: ID for the container div (e.g., "yearDiv")
// fieldName: Name attribute for checkboxes (e.g., "year")
// displayName: Label text (e.g., "Year")
// fieldValues: Available values to display as checkboxes
// nextFieldName: Optional next field name for HTMX chain (empty string if none)
func fieldCheckboxGrid(fieldID, fieldName, displayName string, fieldValues []string, nextFieldName string) g.Node {

	// If no values available, show a message
	if len(fieldValues) == 0 {
		return Div(
			ID(fieldID),
			Class("space-y-6"),
			Div(
				Class("space-y-2"),
				Label(For(fieldName), Class("block"), g.Text(displayName)),
				Div(
					Class("p-4 bg-gray-50 border border-gray-200 rounded-lg text-gray-500 text-sm"),
					g.Text("No choices available"),
				),
			),
		)
	}

	// Create checkbox grid
	checkboxes := []g.Node{}
	for _, value := range fieldValues {
		attrs := []g.Node{
			Type("checkbox"),
			Name(fieldName),
			Value(value),
			ID(fieldName + "-" + value),
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
		endpoint := fmt.Sprintf("/api/field/%s", nextFieldName)
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

	containerChildren := []g.Node{formGroup}
	if nextFieldName != "" {
		// Add empty placeholder div for next field in chain
		// This allows HTMX to replace it and enables proper deletion of downstream elements
		containerChildren = append(containerChildren,
			Div(ID(nextFieldName+"Div")))
	}

	return Div(
		ID(fieldID),
		Class("space-y-6"),
		g.Group(containerChildren),
	)
}
