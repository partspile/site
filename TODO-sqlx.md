# SQLX Migration TODO List

## Overview
This document outlines the step-by-step migration from `database/sql` to `sqlx` for the PartsPile codebase. The migration will significantly reduce boilerplate code and improve maintainability.

## Current State Analysis
- **Current**: Using `database/sql` with manual `rows.Next()` and `rows.Scan()` patterns
- **Target**: Using `sqlx` with `Select()`, `Get()`, and struct scanning
- **Impact**: 37+ locations with manual row iteration identified

## Phase 1: Foundation Setup

### 1.1 Add SQLX Dependency
- [X] Add `github.com/jmoiron/sqlx` to `go.mod`
- [X] Run `go mod tidy` to update dependencies
- [X] Verify sqlx is properly imported

### 1.2 Update Database Package (`db/db.go`)
- [X] Change `var db *sql.DB` to `var db *sqlx.DB`
- [X] Update `Init()` function to use `sqlx.Open()` instead of `sql.Open()`
- [X] Update `Get()` function to return `*sqlx.DB`
- [X] Update `SetForTesting()` to accept `*sqlx.DB`
- [X] Add sqlx convenience methods:
  - [X] `Select(dest interface{}, query string, args ...interface{}) error`
  - [X] `Get(dest interface{}, query string, args ...interface{}) error`
  - [X] `NamedExec(query string, arg interface{}) (sql.Result, error)`
  - [X] `NamedQuery(query string, arg interface{}) (*sqlx.Rows, error)`

### 1.3 Update Imports Across Codebase
- [X] Update all files using `db` package to use new sqlx methods

## Phase 2: High-Impact Migrations (Priority Order)

### 2.1 Simple Array Queries (Low Risk, High Impact)
**Files**: `ad/bookmark.go`, `ad/click.go`, `search/search.go`

- [X] **ad/bookmark.go**: `GetBookmarkedAdIDs()` - Convert to `db.Select(&adIDs, query)`
- [X] **ad/click.go**: Similar pattern queries
- [X] **search/search.go**: 
  - [X] `GetUserSearches()` - Convert to `db.Select()`
  - [X] `GetPopularSearches()` - Convert to `db.Select()`

### 2.2 Vehicle Data Queries (Medium Risk, High Impact)
**File**: `vehicle/vehicle.go`

- [X] **Make queries** (lines 53-69):
  - [X] `GetMakes()` - Convert to `db.Select()`
  - [X] `GetMakeByName()` - Convert to `db.Get()`
- [X] **Year queries** (lines 106-125):
  - [X] `GetYears()` - Convert to `db.Select()`
  - [X] `GetYearByValue()` - Convert to `db.Get()`
- [X] **Model queries** (lines 160-222):
  - [X] `GetModels()` - Convert to `db.Select()`
  - [X] `GetModelByName()` - Convert to `db.Get()`
- [X] **Engine queries** (lines 268-320):
  - [X] `GetEngines()` - Convert to `db.Select()`
  - [X] `GetEngineByName()` - Convert to `db.Get()`

### 2.3 Part Category Queries (Low Risk, Medium Impact)
**File**: `part/part.go`

- [ ] **Category queries** (lines 33-57):
  - [ ] `GetCategories()` - Convert to `db.Select()`
  - [ ] `GetSubCategories()` - Convert to `db.Select()`
- [ ] **Simple lookup queries** (lines 87-425):
  - [ ] Various `Get*ByParent()` functions - Convert to `db.Select()`

## Phase 3: Complex Struct Migrations (Higher Risk)

### 3.1 Ad Queries (High Risk, High Impact)
**Files**: `ad/ad.go`, `part/part.go`

- [ ] **ad/ad.go**: `scanAdRows()` function (lines 164-220):
  - [ ] Convert complex struct scanning to sqlx struct tags
  - [ ] Add struct tags to `Ad` struct for automatic scanning
  - [ ] Test with complex queries involving joins

- [ ] **part/part.go**: Complex ad queries (lines 506-539, 818-870):
  - [ ] `GetAdsForNode()` - Convert to `db.Select()` with struct tags
  - [ ] `GetAdsForNodeStructured()` - Convert to `db.Select()` with struct tags
  - [ ] Handle complex joins and nullable fields

### 3.2 Messaging System (Medium Risk, Medium Impact)
**File**: `messaging/messaging.go`

- [ ] **Conversation queries** (lines 192-210):
  - [ ] `GetConversationsForUser()` - Convert to `db.Select()`
  - [ ] Add struct tags to `Conversation` struct
- [ ] **Message queries** (lines 356-370):
  - [ ] `GetMessagesForConversation()` - Convert to `db.Select()`
  - [ ] Add struct tags to `Message` struct

### 3.3 Rock System (Medium Risk, Low Impact)
**File**: `rock/rock.go`

- [ ] **Rock queries** (lines 147-160, 191-204):
  - [ ] `GetRocksForUser()` - Convert to `db.Select()`
  - [ ] `GetRocksForAd()` - Convert to `db.Select()`
  - [ ] Add struct tags to `Rock` struct

## Phase 4: Testing and Validation

### 4.1 Unit Testing
- [ ] Run existing tests to ensure no regressions
- [ ] Add specific tests for sqlx conversions
- [ ] Test edge cases (empty results, null values, etc.)

### 4.2 Integration Testing
- [ ] Test all migrated queries in development environment
- [ ] Verify performance characteristics
- [ ] Test with real data scenarios

### 4.3 Performance Validation
- [ ] Benchmark critical queries before/after migration
- [ ] Ensure no performance degradation
- [ ] Optimize any queries that show performance issues

## Phase 5: Cleanup and Documentation

### 5.1 Code Cleanup
- [ ] Remove unused `database/sql` imports where possible
- [ ] Update comments and documentation
- [ ] Ensure consistent error handling patterns

### 5.2 Documentation Updates
- [ ] Update README with sqlx usage examples
- [ ] Document new database patterns and best practices
- [ ] Create migration guide for future developers

## Migration Strategy Notes

### Risk Mitigation
- **Gradual Migration**: Migrate one file/function at a time
- **Backward Compatibility**: Keep old methods during transition
- **Testing**: Comprehensive testing at each phase
- **Rollback Plan**: Keep git commits granular for easy rollback

### Struct Tag Examples
```go
type Ad struct {
    ID            int       `db:"id"`
    Title         string    `db:"title"`
    Description   string    `db:"description"`
    Price         float64   `db:"price"`
    CreatedAt     time.Time `db:"created_at"`
    // ... other fields
}
```

### Query Conversion Examples
```go
// Before (database/sql)
rows, err := db.Query("SELECT id, name FROM Make")
if err != nil {
    return nil, err
}
defer rows.Close()
var makes []Make
for rows.Next() {
    var make Make
    if err := rows.Scan(&make.ID, &make.Name); err != nil {
        continue
    }
    makes = append(makes, make)
}

// After (sqlx)
var makes []Make
err := db.Select(&makes, "SELECT id, name FROM Make")
return makes, err
```

## Success Criteria
- [ ] All manual `rows.Next()` loops eliminated
- [ ] Code reduction of ~30-50% in database query functions
- [ ] No performance regression
- [ ] All tests passing
- [ ] Improved code readability and maintainability

## Estimated Timeline
- **Phase 1**: 1-2 days (Foundation setup)
- **Phase 2**: 3-5 days (Simple migrations)
- **Phase 3**: 5-7 days (Complex migrations)
- **Phase 4**: 2-3 days (Testing)
- **Phase 5**: 1-2 days (Cleanup)

**Total Estimated Time**: 2-3 weeks
