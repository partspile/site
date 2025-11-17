# Spec Field Refactoring Report

This report identifies all places where specific spec fields (location, price, years, make, models, engine, part_category, part_subcategory, breed) are hardcoded instead of using the `spec/field-*.go` implementations.

## Summary

The codebase has field implementations in `spec/field-*.go` that provide:
- `BuildFilterCondition()` - builds Qdrant filter conditions
- `RenderNewAdField()` - renders fields in new ad forms
- `RenderFilterField()` - renders fields in filter dialogs
- `Validate()` - validates field values

However, there are many places where these fields are still hardcoded with direct string references or special-case logic.

## Issues Found

### ✅ RESOLVED Issues

### 1. handlers/ad-edit.go - HandleUpdateAdPrice
- **Status**: ✅ FIXED
- **Previous Issue**: Used `ValidateAndParsePrice()` which hardcoded price handling
- **Current State**: Now uses `&spec.FieldPrice{}` directly with `Validate()` method (lines 36-42)

### 2. handlers/view-tree.go - GetAdIDs()
- **Status**: ✅ FIXED
- **Previous Issue**: Hardcoded filter checks for location, make, year, price
- **Current State**: Now uses field implementations via `field.BuildFilterCondition()` (lines 25-37)

### 3. handlers/validation.go - ValidateAndParsePrice()
- **Status**: ✅ FIXED
- **Previous Issue**: Function hardcoded `c.FormValue("price")`
- **Current State**: Function no longer exists (removed)

---

### ❌ UNRESOLVED Issues

### 1. handlers/validation.go - BuildAdFromForm()

**Lines 339, 346**: Hardcodes field names:
- `priceStr := c.FormValue("price")` (line 339)
- `locationRaw := c.FormValue("location")` (line 346)

- **Issue**: Should use field implementations to get field names dynamically
- **Impact**: Price and location extraction bypasses field system
- **Note**: Fields are validated using field implementations (lines 305-336), but extraction still hardcodes names

### 2. handlers/ad-edit.go - HandleUpdateAdLocation()

**Line 157**: Hardcodes `c.FormValue("location")`

- **Issue**: Should use field implementation to get location value
- **Impact**: Location update bypasses field system
- **Note**: Comment says "using field implementation" but still hardcodes the field name

### 3. ui/ad-detail.go - PriceEditModal() and LocationEditModal()

**Lines 437-459**: `PriceEditModal()` hardcodes price field in modal
**Lines 461-483**: `LocationEditModal()` hardcodes location field in modal

- **Issue**: Should use `FieldPrice.RenderNewAdField()` and `FieldLocation.RenderNewAdField()` or similar
- **Impact**: Price and location edit modals bypass field system
- **Priority**: High - these are user-facing UI components

### 4. vector/ad.go - buildAdEmbeddingMetadata()

**Lines 147, 149**: Hardcodes field names:
- `"price": adObj.Price` (line 147)
- `"location": map[string]interface{}{"lat": lat, "lon": lon}` (line 149)

- **Issue**: Should use field config to determine metadata keys
- **Impact**: Metadata keys are hardcoded and won't work if field names change
- **Note**: Other spec fields are handled dynamically (lines 156-173)

### 5. ui/ad-detail.go - partTypePath()

**Line 768**: Special case for `fieldName == "year"` in year range formatting

- **Issue**: Year range formatting is hardcoded
- **Impact**: Year formatting logic should be in `FieldYear` implementation
- **Note**: This is formatting logic, could be moved to field implementation
- **Priority**: Low - formatting only

### 6. handlers/ad-new.go

**Status**: ✅ FIXED

- **Previous Issue**: Handler functions for specific fields (`HandleYears`, `HandleModels`, `HandleEngines`, `HandlePartCategory`, `HandlePartSubcategory`) were wrapper functions with hardcoded field names
- **Current State**: Removed all specific handler functions. Now uses a single generic endpoint `/api/field/:fieldName` with `HandleFieldValuesFromRoute()` handler
- **Changes Made**:
  - Removed `HandleYears`, `HandleModels`, `HandleEngines`, `HandlePartCategory`, `HandlePartSubcategory` functions
  - Created `HandleFieldValuesFromRoute()` that extracts field name from route parameter
  - Updated `spec/field.go` and `handlers/ad-new.go` to use `/api/field/:fieldName` endpoint
  - Removed specific routes from `main.go`
  - Updated `PRD.md` to reflect new API endpoint

### 7. spec/spec.go

**Line 45**: `GetMakes()` hardcodes `GetFieldValues(adCat, "make", ...)`
**Lines 64-65**: `GetYears()` hardcodes field names
**Lines 90-91, 93**: `GetModels()` hardcodes field names
**Lines 132-134, 136**: `GetEngines()` hardcodes field names

- **Issue**: Field names are hardcoded in convenience functions
- **Impact**: Functions assume specific field chains
- **Note**: These functions are convenience wrappers for common chains. They could be deprecated in favor of generic field access, but they provide backward compatibility.
- **Priority**: Low - backward compatibility wrappers

### 8. part/part.go

**Lines 159, 226**: `buildAdCategoriesQuery()` and `buildAdSubCategoriesQuery()` hardcode:
- `if config.Name == "part_category" || config.Name == "part_subcategory"`

- **Issue**: Field names are hardcoded
- **Impact**: Logic assumes these specific field names exist
- **Note**: This is reasonable since these functions are specifically for part categories, but could use field config to identify category fields
- **Priority**: Low - reasonable for part-specific functions

### 9. vector/ad.go - buildAdEmbeddingPrompt()

**Lines 132, 135**: Hardcodes:
- `if catValues, exists := adObj.SpecData["part_category"]`
- `if subcatValues, exists := adObj.SpecData["part_subcategory"]`

- **Issue**: Field names are hardcoded
- **Impact**: Part category/subcategory handling bypasses field system
- **Note**: The comment says "not in fieldConfigs but always present for parts" - this suggests these should be in the field system
- **Priority**: Medium - affects embedding quality

### 10. ui/ad-new.go

**Note**: `priceInputField()` and `locationInputField()` functions mentioned in original report - these no longer exist in the codebase (removed or never existed)

### 11. ui/filter.go

**Note**: Old hardcoded filter functions (`locationFilter()`, `yearFilter()`, `priceFilter()`) mentioned in original report - these no longer exist. Filter rendering is now dynamic via `filterControls()` and field implementations.

## Recommendations

### High Priority

1. **handlers/validation.go**: Refactor `BuildAdFromForm()` to use field implementations for price and location extraction (lines 339, 346)
2. **handlers/ad-edit.go**: Refactor `HandleUpdateAdLocation()` to use field implementation for location field name (line 157)
3. **ui/ad-detail.go**: Refactor `PriceEditModal()` and `LocationEditModal()` to use field implementations (lines 437-483)

### Medium Priority

4. **vector/ad.go**: Use field configs instead of hardcoded field names for price and location metadata (lines 147, 149)
5. **vector/ad.go**: Use field configs for part_category and part_subcategory in embedding prompt (lines 132, 135)

### Low Priority

6. **ui/ad-detail.go**: Move year formatting logic to `FieldYear` implementation (line 768)
7. **spec/spec.go**: Consider deprecating convenience functions in favor of generic field access (but maintain backward compatibility)
8. **part/part.go**: Consider using field config to identify category fields instead of hardcoded names

## Notes

- The `handlers/filter.go` file correctly uses field implementations via `field.BuildFilterCondition()` - this is the correct pattern
- Many of the hardcoded references are in convenience/legacy functions that could be deprecated
- Some hardcoding is acceptable for backward compatibility, but should be documented
- The field system is well-designed; the issue is that not all code has been migrated to use it
- **Progress**: 3 issues resolved, 9 issues remaining (6 high/medium priority, 3 low priority)
