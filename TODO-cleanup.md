# Code Cleanup TODO List

This document contains specific recommendations for cleaning up the codebase based on a comprehensive code review.

## 🔴 Critical Issues (Fix Immediately)

### 1. Fix Inconsistent Flag Logic
- **File**: `handlers/search.go:356-366`
- **Issue**: TreeView handler still uses flaggedMap logic despite recent commit claiming to remove it
- **Action**: Update TreeView handler to use `ad.Flagged` from SQL queries like other handlers
- **Details**: Replace flaggedMap creation and usage with direct SQL-based flag checking

### 2. Implement or Remove Stub Functions
- **File**: `ad/ad.go:897-925`
- **Issue**: Multiple functions with misleading "existing code" comments that return `nil, nil`
- **Functions**:
  - `GetAdsByMakeModel`
  - `GetAdsBySubCategory` 
  - `GetAdsByMakeModelYear`
  - `GetAdsByMakeModelYearEngine`
  - `GetAdsByMakeModelYearEngineSubCategory`
- **Action**: Either implement these functions properly or remove them entirely

### 3. Remove Debug Print Statements
- **File**: `grok/grok.go:63-64, 96-97`
- **Issue**: Debug prints in production code expose sensitive API data
- **Action**: Remove `fmt.Println("REQUEST")`, `fmt.Println(string(data))`, `fmt.Println("RESPONSE")`, and `fmt.Println(grokResp.Choices[0].Message.Content)`

## 🟡 Major Code Quality Issues

### 4. Refactor Admin Handler Duplication
- **Files**: `handlers/admin.go` (lines 15-330)
- **Issue**: 9+ nearly identical admin handler functions
- **Action**: Create generic admin handler function
- **Template**:
  ```go
  func adminHandler[T any](c *fiber.Ctx, sectionName string, 
                          getData func() ([]T, error), 
                          sectionComponent func([]T) g.Node) error
  ```

### 5. Refactor Export Handler Duplication
- **Files**: `handlers/admin.go:116-173`
- **Issue**: 3 identical export handler patterns
- **Action**: Create generic export handler for all entity types

### 6. Standardize Harsh Terminology
- **Files**: Multiple files throughout codebase
- **Issue**: "Kill", "Resurrect", "Dead" terminology is harsh and unprofessional
- **Actions**:
  - Rename `HandleKillUser` → `HandleArchiveUser`
  - Rename `HandleKillAd` → `HandleArchiveAd`
  - Rename `HandleResurrectUser` → `HandleRestoreUser`
  - Rename `HandleResurrectAd` → `HandleRestoreAd`
  - Rename `StatusDead` → `StatusArchived`
  - Rename `UserDead` table → `ArchivedUser`
  - Rename `AdDead` table → `ArchivedAd`
  - Update all related functions and variables

### 7. Remove Unused Functions ✅ COMPLETED
- **File**: `ad/ad.go`
  - `CloseDB()` (lines 610-615) ✅ REMOVED
  - `GetAdsByUserID()` (lines 703-737) ✅ REMOVED
  - `CreateAd()` (lines 765-791) ✅ REMOVED
  - `GetFilteredAdsPage()` (lines 345-369) ✅ REMOVED
  - `anyStringInSlice()` (lines 598-607) ✅ REMOVED (kept duplicate in handlers/handlers.go)
- **File**: `part/part.go`
  - `GetSubCategoriesForCategory()` (lines 66-88) ✅ REMOVED
- **File**: `user/user.go`
  - `UpdateTokenBalance()` (lines 124-127) ✅ REMOVED
- **Action**: Remove all unused functions listed above ✅ COMPLETED

### 8. Remove Unused Struct Fields
- **File**: `ad/ad.go`
- **Issue**: `Title string` and `Year string` fields in Ad struct are unused
- **Action**: Remove unused fields and update related code

### 9. Fix Inconsistent Variable Naming
- **File**: `vehicle/vehicle.go:27-32`
- **Issue**: Cache variables have inconsistent naming (`makesCache` vs `allModelsCache`)
- **Action**: Standardize to either all use "all" prefix or none

## 🟢 Code Quality Improvements

### 10. Create Error Handling Utilities
- **Issue**: Repeated error handling patterns throughout handlers
- **Action**: Create utility functions:
  ```go
  func handleParamError(err error) error
  func handleDatabaseError(err error) error
  func handleValidationError(c *fiber.Ctx, message string) error
  ```

### 11. Create Generic Database Query Functions
- **Issue**: Similar GetAll patterns across all model files
- **Action**: Implement generic query functions for common patterns like GetAll, GetByID with status handling

### 12. Implement Entity Interface
- **Issue**: Duplicate IsDead() methods and status handling
- **Action**: Create common interface for entities with status management

### 13. Improve Function Parameter Names
- **File**: Various files
- **Issues**:
  - `anyStringInSlice(a, b []string)` → `stringSlicesHaveCommonElement(slice1, slice2 []string)`
  - `GetAdsForNode(..., q string)` → `GetAdsForNode(..., searchQuery string)`
- **Action**: Replace abbreviated and unclear parameter names

### 14. Use Table Name Constants Consistently
- **File**: `ad/ad.go:15-16`
- **Issue**: Constants `TableAd` and `TableAdDead` are defined but table names are hardcoded in queries
- **Action**: Either use the constants consistently in all queries or remove them

### 15. Standardize URL Route Naming
- **File**: `main.go`
- **Issue**: Some admin routes use harsh terminology
- **Action**: Update routes:
  - `/api/admin/users/kill/:id` → `/api/admin/users/archive/:id`
  - `/api/admin/users/resurrect/:id` → `/api/admin/users/restore/:id`

### 16. Create Form Validation Utilities
- **Issue**: Repeated form parsing and validation patterns
- **Action**: Create reusable form validation functions

### 17. Create Authentication Utilities
- **Issue**: Repeated user extraction and permission checking patterns
- **Action**: Centralize user context handling and permission checks

## Implementation Priority

1. **Phase 1** (Critical): Items 1-3 (flag logic, stub functions, debug prints)
2. **Phase 2** (High Impact): Items 4-7 (admin handlers, terminology, unused functions)
3. **Phase 3** (Quality): Items 8-17 (naming, utilities, consistency)

## Estimated Impact

- **Code Reduction**: ~500+ lines of duplicated code
- **Maintainability**: Significantly improved through consistent patterns
- **Readability**: Better naming and reduced complexity
- **Reliability**: Removal of stub functions prevents runtime errors