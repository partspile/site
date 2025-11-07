# Flexible Specification Fields System

## Overview
Move from hard-coded specification fields to a metadata-driven system where each ad category defines its own specification fields. This enables adding new categories (e.g., "horses" with breed/subcat) without code changes. This is a one-way migration - no backward compatibility needed.

## Database Schema Changes

### 1. Create SpecificationField table ✅ COMPLETE
Store field definitions that can be reused across categories:
```sql
CREATE TABLE SpecificationField (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE CHECK (length(name) > 0),
    display_name TEXT NOT NULL,
    field_type TEXT NOT NULL CHECK (field_type IN ('string', 'int', 'float64')),
    table_name TEXT NOT NULL,  -- Always required - table name (e.g., 'Make', 'Year', 'Model')
    table_field_name TEXT NOT NULL,  -- Column name in Specification table (e.g., 'make_id', 'year_id')
    qdrant_type TEXT NOT NULL CHECK (qdrant_type IN ('keyword', 'keyword_array', 'integer', 'float', 'geo'))
);
```

**Key Design Decision**: Unified field handling - all fields (including integers like 'year') use the same pattern with explicit `table_name` and `table_field_name`. This eliminates special casing and makes the system more consistent.

### 2. Create AdCategoryField table ✅ COMPLETE
Define which fields each category uses and their order:
```sql
CREATE TABLE AdCategoryField (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    specification_field_id INTEGER NOT NULL REFERENCES SpecificationField(id),
    display_order INTEGER NOT NULL,
    is_required INTEGER NOT NULL DEFAULT 1,
    UNIQUE (ad_category_id, specification_field_id)
);
CREATE INDEX idx_adcategoryfield_category ON AdCategoryField(ad_category_id);
CREATE INDEX idx_adcategoryfield_order ON AdCategoryField(ad_category_id, display_order);
```

### 3. Keep Specification table normalized ✅ COMPLETE
- Keep existing columns (make_id, year_id, model_id, engine_id, part_subcategory_id) for existing fields
- Add new columns dynamically via schema changes when new field types are added (e.g., breed_id)
- Use AdCategoryField to map which columns are used for each category
- Removed hard-coded composite index `idx_specification_make_year_model_engine` - relies on UNIQUE constraint index instead

### 4. Add breed_id to Specification table (example for horses category) ✅ COMPLETE
```sql
ALTER TABLE Specification ADD COLUMN breed_id INTEGER REFERENCES Breed(id);
```

### 5. Create Breed table (example for horses category) ✅ COMPLETE
```sql
CREATE TABLE Breed (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ad_category_id INTEGER NOT NULL REFERENCES AdCategory(id),
    name TEXT NOT NULL CHECK (length(name) > 0)
);
```

## Code Changes

### Phase 1: Core Infrastructure & Caching ✅ COMPLETE

#### 1.1 Create `spec/field.go` package ✅ COMPLETE
- `FieldConfig` struct with id, name, display_name, field_type, table_name, table_field_name, qdrant_type, is_required
- `GetCategoryFields(adCategoryID int) ([]FieldConfig, error)` - Returns ordered list of fields for a category (uses cache)
- `GetFieldDefinition(fieldName string) (FieldConfig, bool)` - Gets field definition from cache
- `HasField(adCategoryID int, fieldName string) bool` - Checks if category has a field
- `InitFieldCache()` - Loads SpecificationField and AdCategoryField into memory maps at startup
- In-memory cache: `categoryFieldsCache map[int][]FieldConfig` and `fieldDefinitionsCache map[string]FieldConfig`

#### 1.2 Update `ad/category.go` ✅ COMPLETE
- Removed `HasYears()`, `HasEngines()` switch statements
- Now uses `spec.HasField()` which consults dynamic field configuration

#### 1.3 Create `spec/dynamic.go` ✅ COMPLETE
- Generic query builders that work with any field configuration
- `BuildFieldQuery()` - Builds SQL query for single-value parent filters
- `BuildFieldQueryWithIntersection()` - Builds SQL query for multi-value parent filters (e.g., models across multiple years)
- `GetFieldValues()` - Generic function to get dropdown values, handles both single and multi-value cases
- `getValueColumnName()` - Helper to get value column name (handles Year table special case)
- Unified field handling - all fields (int, string) use same pattern with `table_field_name`

### Phase 2: Qdrant Integration ✅ COMPLETE

#### 2.1 Update `vector/collection.go` ✅ COMPLETE
- `GetIndexedFields()` queries SpecificationField table and returns ALL fields dynamically
- Each field's qdrant_type determines the index type (keyword, keyword_array, integer, float, geo)
- `getAllSpecFields()` helper function queries database for all field definitions
- This ensures Qdrant has indexes for all spec fields, even if individual ads only use a subset

#### 2.2 Update `vector/ad.go` ✅ COMPLETE
- `buildAdEmbeddingMetadata()` dynamically builds metadata based on category's field configuration
- Only includes fields that exist for the ad's category
- Uses field configuration to determine if field is array (years, models, engines) or single value (make)
- TODO: Make AdDetail more flexible to support arbitrary fields (currently uses temporary compatibility layer)

#### 2.3 Update `vector/qdrant.go` ✅ COMPLETE
- `convertMetadataToQdrant()` handles dynamic fields based on field configuration
- Uses field's qdrant_type to determine conversion logic
- Added default case to infer Qdrant type for unknown fields, allowing future extensibility

### Phase 3: Ad Creation & Association ✅ COMPLETE

#### 3.1 Refactor `ad/ad-new.go` ✅ COMPLETE
- Replaced `addAdSpecAssociations()` switch with dynamic field processing
- `addDynamicAssociations()` handles spec associations dynamically based on category field configuration
- `findSpec()` works with dynamic fields using `table_field_name` and unified query building
- Removed `addCarAssociations()`, `addBicycleAssociations()`, etc. - replaced with single dynamic function
- `buildFieldValueMap()` extracts field values from AdDetail (temporary compatibility layer)
- `generateFieldCombinations()` handles multi-value fields (years, models, engines)

#### 3.2 Update `handlers/ad-new.go` ✅ COMPLETE
- Replaced `HandleYears()`, `HandleModels()`, `HandleEngines()` with generic `HandleFieldValues(c *fiber.Ctx, fieldName string)`
- `HandleFieldValues()` dynamically retrieves parent field values from query parameters
- Renders appropriate UI component (single-select dropdown or multi-select checkboxes) based on field's `QdrantType`
- `HandleNewAd()` and `HandleDuplicateAd()` dynamically render fields based on category configuration
- Helper functions: `renderEmptyFieldSelector()`, `renderMultiSelectField()`, `renderSingleSelectField()`

#### 3.3 Update `handlers/validation.go` ✅ COMPLETE
- `BuildAdFromForm()` dynamically validates and extracts form fields based on category's field configuration
- Iterates through `fieldConfigs` to validate required fields and extract values for single-value and multi-value fields
- `validateAdFormMultipart()` helper updated to use `FieldConfig` for validating multi-value fields

### Phase 4: Search & Filtering ✅ COMPLETE

#### 4.1 Update `handlers/search.go` ✅ COMPLETE
- `buildSearchFilter()` dynamically constructs Qdrant filters based on category's field configuration
- Fetches `FieldConfig` for current ad category
- Iterates through `fieldConfigs` and conditionally adds Qdrant filter conditions for fields present in category and search parameters
- Supports make, year (range), and breed filters
- TODO: model and engine filters not yet implemented (explicitly noted in code)

#### 4.2 Update `spec/spec.go` ✅ COMPLETE
- `GetMakes()`, `GetYears()`, `GetModels()`, `GetEngines()` refactored to internally use dynamic query builders
- Maintains original function signatures for backward compatibility
- Internally calls `GetFieldValues()` and `BuildFieldQueryWithIntersection()` from `spec/dynamic.go`
- Search-mode functions (`GetAdMakesForAdIDs`, etc.) still use hard-coded logic for now

### Phase 5: Tree View ✅ COMPLETE

#### 5.1 Update `handlers/tree.go` ✅ COMPLETE
- Replaced hard-coded level-based switch with dynamic field-based navigation
- `HandleTreeExpandBrowse()` and `HandleTreeExpandSearch()` use field configuration to determine hierarchy
- Tree depth determined by number of fields in category configuration
- `buildParentValuesFromPath()` helper extracts parent field values from URL path
- Logic for determining children (makes, years, models, engines, categories, subcategories, ads) based on iterating through `fieldConfigs` and current `level`
- TODO: `HandleTreeExpandSearch()` still uses hard-coded `spec.GetAd*ForAdIDs` functions for now, with TODO for breed

#### 5.2 Update `part/part.go` ✅ COMPLETE
- `buildAdCategoriesQuery()` and `buildAdSubCategoriesQuery()` refactored to use dynamic field configuration
- Uses `FieldConfig` and its `TableFieldName` and `FieldType` to construct dynamic SQL queries
- Removed special casing for `year` - all fields use unified logic
- `getValueColumnName()` helper handles Year table special case

### Phase 6: UI Components ⚠️ PARTIAL

#### 6.1 Update `ui/ad-new.go` ✅ COMPLETE
- Replaced hard-coded field rendering with dynamic field renderer
- `renderFieldSelector()` dynamically creates HTML for a given field
- Includes HTMX attributes for dynamic loading of child fields
- Handles clearing subsequent fields when parent field changes
- `NewAdPage()` and `DuplicateAdPage()` fetch `FieldConfig`s and use `renderFieldSelector()` helper
- `makeSelector()` updated to accept parameters for `nextFieldName`, `nextFieldTarget`, and `clearTargetsScript`

#### 6.2 Update `ui/search.go` and `ui/filter.go` ⚠️ TODO
- Make filter UI dynamic based on category field configuration
- Only show filters for fields that exist in category

#### 6.3 Update `ui/view-tree.go` ⚠️ TODO
- Make tree view depth dynamic based on field count

### Phase 7: Rebuild DB Integration ✅ COMPLETE

#### 7.1 Update `cmd/rebuild_db/main.go` ✅ COMPLETE
- Populates SpecificationField table from `specification-fields.json`:
  - make (string, Make table, make_id, keyword)
  - year (int, Year table, year_id, keyword_array)
  - model (string, Model table, model_id, keyword_array)
  - engine (string, Engine table, engine_id, keyword_array)
  - part_subcategory (string, PartSubCategory table, part_subcategory_id, keyword)
  - breed (string, Breed table, breed_id, keyword) - example for horses
- Populates AdCategoryField table from `ad-category.json` (field_names array):
  - Cars & Trucks/Car & Truck Parts: make(1), year(2), model(3), engine(4), part_subcategory(5)
  - Motorcycles/Motorcycle Parts: make(1), year(2), model(3), engine(4), part_subcategory(5)
  - Agricultural Equipment/Agricultural Equipment Parts: make(1), year(2), model(3), part_subcategory(4)
  - Bicycles/Bicycle Parts: make(1), model(2), part_subcategory(3)
- Updated `importSpecs()` and `executeSpecBatch()` to handle `breed_id` column
- All `fmt.Printf`/`fmt.Println` calls converted to `logger.Info()` for proper slog integration
- Field mappings merged into `ad-category.json` - single source of truth for category configuration

## Implementation Order

1. **Database schema changes** (SpecificationField, AdCategoryField tables, add breed_id example)
2. **Core infrastructure & caching** (spec/field.go, cache initialization, ad/category.go updates)
3. **Qdrant integration** (dynamic indexing, dynamic metadata building)
4. **Ad creation** (refactor ad-new.go, handlers)
5. **Search & filtering** (dynamic filter building)
6. **Tree view** (dynamic hierarchy)
7. **UI components** (dynamic rendering)
8. **Rebuild DB** (populate new tables with seed data)
9. **Testing** (verify all categories work)
10. **Remove old code** (switch statements, hard-coded functions)

## Testing Considerations

- Verify existing categories (cars, motorcycles, bicycles, ag) still work
- Test new category creation (e.g., horses with breed/subcat)
- Verify tree view works with different field counts
- Test search/filtering with different field combinations
- Verify Qdrant indexes are created for all spec fields
- Verify rebuild_db populates all tables correctly

## Key Design Decisions

1. **Specification table structure**: Normalized - keep existing columns, add new columns as needed
2. **Field ordering**: Configurable via display_order in AdCategoryField
3. **Field dependencies**: Yes - explicit parent-child relationships via parentFilters
4. **Migration strategy**: No migration scripts - rebuild_db handles everything
5. **Caching**: SpecificationField and AdCategoryField cached in memory at startup for efficient lookups
6. **Qdrant indexing**: ALL spec fields indexed in Qdrant, even if individual ads only use subset
7. **Backward compatibility**: None - this is a one-way migration

## Progress Tracking

- [x] Database schema changes ✅
- [x] Core infrastructure & caching ✅
- [x] Qdrant integration ✅
- [x] Ad creation ✅
- [x] Search & filtering ✅ (partial - model/engine filters TODO)
- [x] Tree view ✅ (partial - search mode functions still hard-coded)
- [x] UI components ⚠️ (ad-new.go complete, search/filter/tree UI TODO)
- [x] Rebuild DB integration ✅
- [ ] Testing ⚠️ (needs verification)
- [x] Remove old code ✅ (most hard-coded logic removed)

## Key Implementation Notes

1. **Unified Field Handling**: All fields (including integers like 'year') use the same pattern with explicit `table_name` and `table_field_name`, eliminating special casing.

2. **Field Type System**: Changed from `('text', 'integer', 'foreign_key')` to Go types `('string', 'int', 'float64')` for consistency.

3. **JSON-Based Configuration**: Field definitions in `specification-fields.json`, category field mappings merged into `ad-category.json` for single source of truth.

4. **Index Optimization**: Removed hard-coded composite index `idx_specification_make_year_model_engine` - relies on UNIQUE constraint index instead for flexibility.

5. **Backward Compatibility**: `spec/spec.go` functions (`GetMakes()`, `GetYears()`, etc.) maintain original signatures but internally use dynamic query builders.

6. **TODOs Remaining**:
   - Make `AdDetail` more flexible to support arbitrary fields (currently uses temporary compatibility layer)
   - Implement model/engine filters in search
   - Update search-mode functions in `spec/spec.go` to be fully generic
   - Update `ui/search.go`, `ui/filter.go`, and `ui/view-tree.go` for dynamic UI rendering

