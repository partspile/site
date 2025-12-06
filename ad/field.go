package ad

import (
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/parts-pile/site/db"
	"github.com/parts-pile/site/logger"
	g "maragu.dev/gomponents"
	hx "maragu.dev/gomponents-htmx"
	. "maragu.dev/gomponents/html"
)

// Field represents a field that can be used in forms, filters, and validation
type Field interface {
	// Name returns the name of the field
	Name() string

	// BuildFilterCondition builds a SQL WHERE clause condition from form values
	// Returns empty string and nil args if no filter should be applied
	// The returned clause should be a SQL fragment (without WHERE keyword) and args for parameterized query
	BuildFilterCondition(c *fiber.Ctx) (string, []interface{})

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

	// GetCategoryID returns the ad category ID for this field
	GetCategoryID() int

	// GetParentField returns the parent field in the chain, or nil if this is the first field
	GetParentField() Field

	// SetParent sets the parent field in the chain
	SetParent(parent Field)

	// GetVectorType returns the vector type string for this field
	// Returns: 'keyword', 'keyword_array', 'integer', 'float', or 'geo'
	GetVectorType() string

	// EmbeddingPrompt returns a formatted prompt snippet for embedding generation
	// fieldValues is a slice of field values (single or multiple)
	// Returns empty string if no values or field should not be included
	EmbeddingPrompt(fieldValues []string) string
}

type adField struct {
	AdCategoryID       int    `json:"ad_category_id"`        // Ad category ID
	FieldName          string `json:"name"`                  // 'make'
	DisplayName        string `json:"display_name"`          // 'Make'
	FieldType          string `json:"field_type"`            // 'string', 'int', 'float64'
	TableName          string `json:"table_name"`            // 'Make'
	VectorType         string `json:"vector_type"`           // 'keyword', 'keyword_array', 'integer', 'float', 'geo'
	NextInChain        int    `json:"next_in_chain"`         // 0-based array index of next field in chain (0 = last field in chain or standalone, > 0 = index of next field)
	NextFieldName      string `json:"next_field_name"`       // 'year'
	IsRequired         bool   `json:"is_required"`           // Is required field on new ad form
	ShowInFilters      bool   `json:"show_in_filters"`       // Show field in filter UI
	SpecTableName      string `json:"spec_table_name"`       // 'SpecVehicle'
	SpecTableFieldName string `json:"spec_table_field_name"` // 'make_id'
	AdSpecTableName    string `json:"ad_spec_table_name"`    // 'AdSpecVehicle'
	IsMultiValueField  bool   `json:"is_multi_value_field"`  // True if multi-value field
	ParentField        Field  `json:"-"`                     // Backlink to parent field in chain (populated during initialization)
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

func (f adField) GetCategoryID() int {
	return f.AdCategoryID
}

func (f adField) GetParentField() Field {
	return f.ParentField
}

func (f *adField) SetParent(parent Field) {
	f.ParentField = parent
}

func (f adField) GetVectorType() string {
	return f.VectorType
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

// collectParentFields collects all fields from params that come before the current field in the category
// This handles cases where fields are in different spec tables (e.g., part_category after engine)
// Uses Field interface methods instead of concrete adField type
// allowCrossSpecTableFiltering: if true, allows filtering by fields from different spec tables (used in tree browse)
// if false, only filters by fields from the same spec table (used in new ad page)
func collectParentFields(field Field, params map[string]string, allowCrossSpecTableFiltering bool) []parentFieldInfo {
	var parentFields []parentFieldInfo

	// Get all fields for this category
	catID := field.GetCategoryID()
	allFields := GetCategoryFields(catID)
	if len(allFields) == 0 {
		return parentFields
	}

	// Find the position of the current field in the category's field list
	currentFieldIndex := -1
	for i, f := range allFields {
		if f.Name() == field.Name() {
			currentFieldIndex = i
			break
		}
	}

	if currentFieldIndex == -1 {
		// Field not found in category - fall back to direct parent chain
		// Get the current field's spec table name
		currentSpecTableName, _, _, _ := field.GetTableNames()
		currentField := field
		for {
			parentField := currentField.GetParentField()
			if parentField == nil {
				break
			}
			parentName := parentField.Name()
			if parentValue, exists := params[parentName]; exists && parentValue != "" {
				// Collect if:
				// 1. Current field has no spec table, OR
				// 2. allowCrossSpecTableFiltering is true, OR
				// 3. Parent is in the same spec table
				parentSpecTableName, _, _, _ := parentField.GetTableNames()
				if currentSpecTableName == "" || allowCrossSpecTableFiltering || parentSpecTableName == currentSpecTableName {
					parentFields = append(parentFields, parentFieldInfo{
						parent: parentField,
						value:  parentValue,
					})
				}
			}
			currentField = parentField
		}
		return parentFields
	}

	// Get the current field's spec table name
	currentSpecTableName, _, _, _ := field.GetTableNames()

	// Collect all fields that come before the current field and are present in params
	// If allowCrossSpecTableFiltering is false, only collect fields from the same spec table
	// (ensures part_category/subcategory are independent from make/year/model/engine on new ad page)
	// If allowCrossSpecTableFiltering is true, collect all earlier fields regardless of spec table
	// (allows part_category to be filtered by make/year/model/engine in tree browse view)
	for i := 0; i < currentFieldIndex; i++ {
		earlierField := allFields[i]
		fieldName := earlierField.Name()
		if fieldValue, exists := params[fieldName]; exists && fieldValue != "" {
			// Only include spec fields as parent filters (non-spec fields like location/price don't filter other fields)
			if earlierField.IsSpecField() {
				earlierSpecTableName, _, _, _ := earlierField.GetTableNames()
				// Collect if:
				// 1. Current field has no spec table (non-spec field can be filtered by any spec field), OR
				// 2. allowCrossSpecTableFiltering is true (tree browse - filter by all earlier fields), OR
				// 3. Earlier field is in the same spec table (same spec table filtering)
				if currentSpecTableName == "" || allowCrossSpecTableFiltering || earlierSpecTableName == currentSpecTableName {
					parentFields = append(parentFields, parentFieldInfo{
						parent: earlierField,
						value:  fieldValue,
					})
				}
			}
		}
	}

	return parentFields
}

// GetValues returns the available values for a field given parent filters
func GetValues(field Field, params map[string]string) ([]string, error) {

	parentFields := collectParentFields(field, params, false)

	query, args, err := buildFilteredQuery(field, parentFields)

	if err != nil {
		return []string{}, err
	}

	return executeQuery(field, query, args)
}

// GetValuesForActiveAds returns the available values for a field given parent filters,
// but only includes values that exist in active (non-deleted) ads.
// This is used for tree view browsing, where we only want to show makes/years/etc that have ads.
func GetValuesForActiveAds(field Field, params map[string]string) ([]string, error) {

	parentFields := collectParentFields(field, params, true)

	query, args, err := buildActiveAdsFilteredQuery(field, parentFields)
	if err != nil {
		return []string{}, err
	}

	return executeQuery(field, query, args)
}

// buildFilteredQuery builds a query with parent field filters but no ad filtering
// Handles both simple queries (no parent fields) and filtered queries (with parent fields)
func buildFilteredQuery(field Field, parentFields []parentFieldInfo) (string, []interface{}, error) {
	specTableName, specTableFieldName, adSpecTableName, tableName := field.GetTableNames()
	catID := field.GetCategoryID()

	query := fmt.Sprintf("SELECT DISTINCT %s.value", tableName)
	var args []interface{}
	var whereClauses []string
	var havingClauses []string

	// Build FROM clause
	needsSpecJoin := specTableName != "" && len(parentFields) > 0
	if !needsSpecJoin {
		query += fmt.Sprintf(" FROM %s", tableName)
	} else {
		query += fmt.Sprintf(" FROM %s", tableName)
		query += fmt.Sprintf(" JOIN %s ON %s.%s = %s.id",
			specTableName, specTableName, specTableFieldName, tableName)
	}

	// Add category WHERE clause
	args = append(args, catID)
	if needsSpecJoin {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.ad_category_id = $%d", specTableName, len(args)))
	} else {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.ad_category_id = $%d", tableName, len(args)))
	}

	// Handle parent fields
	multiValueParentsSameSpec, multiValueParentsDiffSpec := addParentFieldJoins(
		&query, &args, &whereClauses, &havingClauses,
		field, parentFields, specTableName, adSpecTableName, false)

	// Add final clauses
	query = addFinalClauses(query, whereClauses, havingClauses, tableName, multiValueParentsSameSpec, multiValueParentsDiffSpec)

	return query, args, nil
}

// buildActiveAdsFilteredQuery builds a query with active ads filtering (may have parent fields)
func buildActiveAdsFilteredQuery(field Field, parentFields []parentFieldInfo) (string, []interface{}, error) {
	specTableName, specTableFieldName, adSpecTableName, tableName := field.GetTableNames()
	catID := field.GetCategoryID()

	query := fmt.Sprintf("SELECT DISTINCT %s.value", tableName)
	var args []interface{}
	var whereClauses []string
	var havingClauses []string

	// Build FROM clause with Ad joins
	if specTableName == "" {
		query += fmt.Sprintf(" FROM %s", tableName)
	} else {
		query += fmt.Sprintf(" FROM %s", tableName)
		query += fmt.Sprintf(" JOIN %s ON %s.%s = %s.id",
			specTableName, specTableName, specTableFieldName, tableName)
		if adSpecTableName != "" {
			query += fmt.Sprintf(" JOIN %s ON %s.spec_id = %s.id",
				adSpecTableName, adSpecTableName, specTableName)
			query += " JOIN Ad ON Ad.id = " + adSpecTableName + ".ad_id"
		}
	}

	// Add category WHERE clause
	args = append(args, catID)
	if specTableName != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.ad_category_id = $%d", specTableName, len(args)))
	} else {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.ad_category_id = $%d", tableName, len(args)))
	}

	// Filter by active ads
	whereClauses = append(whereClauses, "Ad.deleted_at IS NULL")

	// Handle parent fields
	multiValueParentsSameSpec, multiValueParentsDiffSpec := addParentFieldJoins(
		&query, &args, &whereClauses, &havingClauses,
		field, parentFields, specTableName, adSpecTableName, true)

	// Add final clauses
	query = addFinalClauses(query, whereClauses, havingClauses, tableName, multiValueParentsSameSpec, multiValueParentsDiffSpec)

	return query, args, nil
}

// buildAdIDFilteredQuery builds a query with ad ID filtering (may have parent fields)
func buildAdIDFilteredQuery(field Field, parentFields []parentFieldInfo, adIDs []int) (string, []interface{}, error) {
	specTableName, specTableFieldName, adSpecTableName, tableName := field.GetTableNames()
	catID := field.GetCategoryID()

	query := fmt.Sprintf("SELECT DISTINCT %s.value", tableName)
	var args []interface{}
	var whereClauses []string
	var havingClauses []string

	// Build FROM clause with Ad joins
	if specTableName == "" {
		query += fmt.Sprintf(" FROM %s", tableName)
	} else {
		query += fmt.Sprintf(" FROM %s", tableName)
		query += fmt.Sprintf(" JOIN %s ON %s.%s = %s.id",
			specTableName, specTableName, specTableFieldName, tableName)
		if adSpecTableName != "" {
			query += fmt.Sprintf(" JOIN %s ON %s.spec_id = %s.id",
				adSpecTableName, adSpecTableName, specTableName)
			query += " JOIN Ad ON Ad.id = " + adSpecTableName + ".ad_id"
		}
	}

	// Add category WHERE clause
	args = append(args, catID)
	if specTableName != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.ad_category_id = $%d", specTableName, len(args)))
	} else {
		whereClauses = append(whereClauses, fmt.Sprintf("%s.ad_category_id = $%d", tableName, len(args)))
	}

	// Filter by ad IDs
	placeholders := make([]string, len(adIDs))
	for i, id := range adIDs {
		args = append(args, id)
		placeholders[i] = fmt.Sprintf("$%d", len(args))
	}
	whereClauses = append(whereClauses, fmt.Sprintf("Ad.id IN (%s)", strings.Join(placeholders, ", ")))

	// Handle parent fields
	multiValueParentsSameSpec, multiValueParentsDiffSpec := addParentFieldJoins(
		&query, &args, &whereClauses, &havingClauses,
		field, parentFields, specTableName, adSpecTableName, true)

	// Add final clauses
	query = addFinalClauses(query, whereClauses, havingClauses, tableName, multiValueParentsSameSpec, multiValueParentsDiffSpec)

	return query, args, nil
}

// addParentFieldJoins adds JOINs and WHERE/HAVING clauses for parent fields
// Returns counts of multi-value parents for GROUP BY/HAVING logic
func addParentFieldJoins(query *string, args *[]interface{}, whereClauses *[]string, havingClauses *[]string,
	field Field, parentFields []parentFieldInfo, specTableName, adSpecTableName string, hasAdJoin bool) ([]parentFieldInfo, []parentFieldInfo) {

	if len(parentFields) == 0 {
		return nil, nil
	}

	var singleValueParentsSameSpec []parentFieldInfo
	var singleValueParentsDiffSpec []parentFieldInfo
	var multiValueParentsSameSpec []parentFieldInfo
	var multiValueParentsDiffSpec []parentFieldInfo

	// Separate parent fields by type and spec table
	for _, pf := range parentFields {
		pfSpecTableName, _, _, _ := pf.parent.GetTableNames()
		if pf.parent.IsMultiValue() {
			if pfSpecTableName == specTableName {
				multiValueParentsSameSpec = append(multiValueParentsSameSpec, pf)
			} else {
				multiValueParentsDiffSpec = append(multiValueParentsDiffSpec, pf)
			}
		} else if pfSpecTableName == specTableName {
			singleValueParentsSameSpec = append(singleValueParentsSameSpec, pf)
		} else {
			singleValueParentsDiffSpec = append(singleValueParentsDiffSpec, pf)
		}
	}

	// Ensure Ad table is joined if we have parents in different spec tables
	if (len(singleValueParentsDiffSpec) > 0 || len(multiValueParentsDiffSpec) > 0) &&
		specTableName != "" && adSpecTableName != "" && !hasAdJoin {
		*query += fmt.Sprintf(" JOIN %s ON %s.spec_id = %s.id",
			adSpecTableName, adSpecTableName, specTableName)
		*query += " JOIN Ad ON Ad.id = " + adSpecTableName + ".ad_id"
		hasAdJoin = true
	}

	// Add JOINs for single-value parent fields in the same spec table
	for _, pf := range singleValueParentsSameSpec {
		_, pfSpecTableFieldName, _, pfTableName := pf.parent.GetTableNames()
		*query += fmt.Sprintf(" JOIN %s ON %s.%s = %s.id",
			pfTableName, specTableName, pfSpecTableFieldName, pfTableName)
		*args = append(*args, pf.value)
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s.value = $%d", pfTableName, len(*args)))
	}

	// Add JOINs for single-value parent fields in different spec tables
	addDiffSpecSingleValueJoins(query, args, whereClauses, singleValueParentsDiffSpec, hasAdJoin)

	// Handle multi-value parent fields in the same spec table
	for _, pf := range multiValueParentsSameSpec {
		_, pfSpecTableFieldName, _, pfTableName := pf.parent.GetTableNames()
		values := strings.Split(pf.value, ",")
		if len(values) == 0 {
			continue
		}

		*query += fmt.Sprintf(" JOIN %s ON %s.%s = %s.id",
			pfTableName, specTableName, pfSpecTableFieldName, pfTableName)

		placeholders := make([]string, len(values))
		for i, val := range values {
			*args = append(*args, val)
			placeholders[i] = fmt.Sprintf("$%d", len(*args))
		}
		*whereClauses = append(*whereClauses, fmt.Sprintf("%s.value IN (%s)",
			pfTableName, strings.Join(placeholders, ", ")))
		*havingClauses = append(*havingClauses, fmt.Sprintf("COUNT(DISTINCT %s.value) = %d",
			pfTableName, len(values)))
	}

	// Handle multi-value parent fields in different spec tables
	addDiffSpecMultiValueJoins(query, args, whereClauses, havingClauses, multiValueParentsDiffSpec, hasAdJoin)

	// Add combination HAVING clauses for multiple multi-value parents
	addMultiValueCombinationHavings(havingClauses, multiValueParentsSameSpec, multiValueParentsDiffSpec)

	return multiValueParentsSameSpec, multiValueParentsDiffSpec
}

// addDiffSpecSingleValueJoins adds JOINs for single-value parent fields in different spec tables
func addDiffSpecSingleValueJoins(query *string, args *[]interface{}, whereClauses *[]string,
	singleValueParentsDiffSpec []parentFieldInfo, hasAdJoin bool) {

	if len(singleValueParentsDiffSpec) == 0 {
		return
	}

	// Group by spec table
	specTableGroups := make(map[string][]parentFieldInfo)
	for _, pf := range singleValueParentsDiffSpec {
		pfSpecTableName, _, _, _ := pf.parent.GetTableNames()
		specTableGroups[pfSpecTableName] = append(specTableGroups[pfSpecTableName], pf)
	}

	// For each spec table group, join once through Ad, then join all fields
	for parentSpecTableName, groupFields := range specTableGroups {
		if len(groupFields) == 0 {
			continue
		}

		firstField := groupFields[0]
		_, _, parentAdSpecTableName, _ := firstField.parent.GetTableNames()
		parentAdSpecAlias := parentAdSpecTableName + "_parent"
		parentSpecAlias := parentSpecTableName + "_parent"

		*query += fmt.Sprintf(" JOIN %s %s ON %s.ad_id = Ad.id",
			parentAdSpecTableName, parentAdSpecAlias, parentAdSpecAlias)
		*query += fmt.Sprintf(" JOIN %s %s ON %s.spec_id = %s.id",
			parentSpecTableName, parentSpecAlias, parentAdSpecAlias, parentSpecAlias)

		for _, pf := range groupFields {
			_, pfSpecTableFieldName, _, pfTableName := pf.parent.GetTableNames()
			parentFieldAlias := pfTableName + "_parent"
			*query += fmt.Sprintf(" JOIN %s %s ON %s.%s = %s.id",
				pfTableName, parentFieldAlias, parentSpecAlias, pfSpecTableFieldName, parentFieldAlias)
			*args = append(*args, pf.value)
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s.value = $%d", parentFieldAlias, len(*args)))
		}
	}
}

// addDiffSpecMultiValueJoins adds JOINs for multi-value parent fields in different spec tables
func addDiffSpecMultiValueJoins(query *string, args *[]interface{}, whereClauses *[]string, havingClauses *[]string,
	multiValueParentsDiffSpec []parentFieldInfo, hasAdJoin bool) {

	if len(multiValueParentsDiffSpec) == 0 {
		return
	}

	// Group by spec table
	multiValueSpecTableGroups := make(map[string][]parentFieldInfo)
	for _, pf := range multiValueParentsDiffSpec {
		pfSpecTableName, _, _, _ := pf.parent.GetTableNames()
		multiValueSpecTableGroups[pfSpecTableName] = append(multiValueSpecTableGroups[pfSpecTableName], pf)
	}

	// For each spec table group, join once through Ad, then join all fields
	for parentSpecTableName, groupFields := range multiValueSpecTableGroups {
		if len(groupFields) == 0 {
			continue
		}

		firstField := groupFields[0]
		_, _, parentAdSpecTableName, _ := firstField.parent.GetTableNames()
		parentAdSpecAlias := parentAdSpecTableName + "_mv_parent"
		parentSpecAlias := parentSpecTableName + "_mv_parent"

		*query += fmt.Sprintf(" JOIN %s %s ON %s.ad_id = Ad.id",
			parentAdSpecTableName, parentAdSpecAlias, parentAdSpecAlias)
		*query += fmt.Sprintf(" JOIN %s %s ON %s.spec_id = %s.id",
			parentSpecTableName, parentSpecAlias, parentAdSpecAlias, parentSpecAlias)

		for _, pf := range groupFields {
			_, pfSpecTableFieldName, _, pfTableName := pf.parent.GetTableNames()
			parentFieldAlias := pfTableName + "_mv_parent"
			*query += fmt.Sprintf(" JOIN %s %s ON %s.%s = %s.id",
				pfTableName, parentFieldAlias, parentSpecAlias, pfSpecTableFieldName, parentFieldAlias)

			values := strings.Split(pf.value, ",")
			if len(values) == 0 {
				continue
			}

			placeholders := make([]string, len(values))
			for i, val := range values {
				*args = append(*args, val)
				placeholders[i] = fmt.Sprintf("$%d", len(*args))
			}
			*whereClauses = append(*whereClauses, fmt.Sprintf("%s.value IN (%s)",
				parentFieldAlias, strings.Join(placeholders, ", ")))
			*havingClauses = append(*havingClauses, fmt.Sprintf("COUNT(DISTINCT %s.value) = %d",
				parentFieldAlias, len(values)))
		}
	}
}

// addMultiValueCombinationHavings adds HAVING clauses for multiple multi-value parent combinations
func addMultiValueCombinationHavings(havingClauses *[]string,
	multiValueParentsSameSpec, multiValueParentsDiffSpec []parentFieldInfo) {

	// Multiple multi-value parents in same spec table
	if len(multiValueParentsSameSpec) > 1 {
		var comboParts []string
		for _, pf := range multiValueParentsSameSpec {
			_, _, _, pfTableName := pf.parent.GetTableNames()
			comboParts = append(comboParts, pfTableName+".value")
		}
		expectedCombos := 1
		for _, pf := range multiValueParentsSameSpec {
			values := strings.Split(pf.value, ",")
			expectedCombos *= len(values)
		}
		comboExpr := strings.Join(comboParts, " || '|' || ")
		*havingClauses = append(*havingClauses, fmt.Sprintf("COUNT(DISTINCT (%s)) = %d",
			comboExpr, expectedCombos))
	}

	// Multiple multi-value parents in different spec tables
	if len(multiValueParentsDiffSpec) > 1 {
		var comboParts []string
		for _, pf := range multiValueParentsDiffSpec {
			_, _, _, pfTableName := pf.parent.GetTableNames()
			comboParts = append(comboParts, pfTableName+"_mv_parent.value")
		}
		expectedCombos := 1
		for _, pf := range multiValueParentsDiffSpec {
			values := strings.Split(pf.value, ",")
			expectedCombos *= len(values)
		}
		comboExpr := strings.Join(comboParts, " || '|' || ")
		*havingClauses = append(*havingClauses, fmt.Sprintf("COUNT(DISTINCT (%s)) = %d",
			comboExpr, expectedCombos))
	}
}

// addFinalClauses adds WHERE, GROUP BY, HAVING, and ORDER BY clauses to the query
func addFinalClauses(query string, whereClauses []string, havingClauses []string,
	tableName string, multiValueParentsSameSpec, multiValueParentsDiffSpec []parentFieldInfo) string {

	if len(whereClauses) > 0 {
		query += " WHERE " + strings.Join(whereClauses, " AND ")
	}

	if len(multiValueParentsSameSpec) > 0 || len(multiValueParentsDiffSpec) > 0 {
		query += fmt.Sprintf(" GROUP BY %s.value", tableName)
	}

	if len(havingClauses) > 0 {
		query += " HAVING " + strings.Join(havingClauses, " AND ")
	}

	query += fmt.Sprintf(" ORDER BY %s.value", tableName)

	return query
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

// executeQuery executes a SQL query and logs the results
func executeQuery(field Field, query string, args []interface{}) ([]string, error) {

	var values []string
	err := db.Select(&values, query, args...)
	if err != nil {
		logger.Error("executeQuery failed",
			"component", "ad-field",
			"field", field.Name(),
			"error", err,
			"query", query,
		)
		return []string{}, err
	}

	return values, err
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
	parent Field
	value  string
}

// BuildParentValues builds a parentValues map from fiber context query parameters
// It validates against known fields from the category and handles multi-value fields correctly
// excludeFields: field names to exclude from the result (e.g., the current field being queried)
func BuildParentValues(c *fiber.Ctx, adCat int, excludeFields []string) map[string]string {
	parentValues := make(map[string]string)
	fields := GetCategoryFields(adCat)

	// Create a map of excluded fields for quick lookup
	excludedMap := make(map[string]bool)
	for _, name := range excludeFields {
		excludedMap[name] = true
	}

	// Get all query parameters
	allQueries := c.Queries()

	// Debug: Log raw query parameters
	logger.Debug("BuildParentValues",
		"component", "ad-field",
		"adCat", adCat,
		"excludeFields", excludeFields,
		"allQueries", allQueries,
		"queryString", c.OriginalURL(),
	)

	for _, field := range fields {
		if excludedMap[field.Name()] {
			continue
		}

		// Get values from query parameters
		if queryValue, exists := allQueries[field.Name()]; exists && len(queryValue) > 0 {
			// For multi-value fields, we need to get all values (not just the last one)
			// c.Queries() only returns the last value for duplicate keys, so we need to parse manually
			if field.IsMultiValue() {
				// Get all values for this field from query args
				queryArgs := c.Context().QueryArgs()
				var values []string
				queryArgs.VisitAll(func(key, value []byte) {
					if string(key) == field.Name() && len(value) > 0 {
						values = append(values, string(value))
					}
				})
				if len(values) > 0 {
					parentValues[field.Name()] = strings.Join(values, ",")
					logger.Debug("BuildParentValues added multi-value field",
						"component", "ad-field",
						"fieldName", field.Name(),
						"values", values,
						"joined", parentValues[field.Name()],
					)
				}
			} else {
				// Single-value field - convert entire byte slice to string
				parentValues[field.Name()] = string(queryValue)
				logger.Debug("BuildParentValues added field",
					"component", "ad-field",
					"fieldName", field.Name(),
					"value", parentValues[field.Name()],
				)
			}
		}
	}

	logger.Debug("BuildParentValues result",
		"component", "ad-field",
		"parentValues", parentValues,
	)

	return parentValues
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
			hx.Include("closest form"),
			hx.Swap("outerHTML"),
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
			hx.Trigger("change"),
			hx.Get(endpoint),
			hx.Target(fmt.Sprintf("#%s", nextFieldName+"Div")),
			hx.Include("closest form"),
			hx.Swap("outerHTML"),
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
