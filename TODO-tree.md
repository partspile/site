# Tree View Cleanup TODO

## Overview
Clean up the tree view implementation to have clear separation between browse mode (`q==""`) and search mode (`q!=""`), with efficient SQL-based tree building.

## Key Changes Made
- ✅ Changed `GetAds()` to `GetAdIDs()` - returns `[]int` instead of `[]ad.Ad`
- ✅ Updated `performSearch()` to return ad IDs
- ✅ Separated vector search (returns IDs) from ad fetching

## TODO Items

### Phase 1: TreeView Handler Simplification
- [x] **Update TreeView.GetAdIDs()** 
  - When `q==""`: Return empty `[]int` (no vector search needed)
  - When `q!=""`: Return ad IDs from vector search
  - Remove complex tree-specific logic from this method

### Phase 2: Create New SQL Functions for Tree Building
- [x] **Create filtered tree functions in `part/part.go`:**
  - [x] `GetMakesForAdIDs(adIDs []int) []string` - Get makes filtered by ad IDs
  - [x] `GetYearsForAdIDs(adIDs []int, makeName string) []string`
  - [x] `GetModelsForAdIDs(adIDs []int, makeName, year string) []string`
  - [x] `GetEnginesForAdIDs(adIDs []int, makeName, year, model string) []string`
  - [x] `GetCategoriesForAdIDs(adIDs []int, makeName, year, model, engine string) []string`
  - [x] `GetSubCategoriesForAdIDs(adIDs []int, makeName, year, model, engine, category string) []string`
  - [x] `GetAdsForAdIDs(adIDs []int, makeName, year, model, engine, category, subcategory string) []ad.Ad`

- [x] **Create browse mode functions (when adIDs is nil/empty):**
  - [x] `GetMakesForAll() []string` - Get all makes that have ads
  - [x] `GetYearsForAll(makeName string) []string`
  - [x] `GetModelsForAll(makeName, year string) []string`
  - [x] `GetEnginesForAll(makeName, year, model string) []string`
  - [x] `GetCategoriesAll(makeName, year, model, engine string) []string`
  - [x] `GetSubCategoriesForAll(makeName, year, model, engine, category string) []string`
  - [x] `GetAdsForAll(makeName, year, model, engine, category, subcategory string) []ad.Ad`

### Phase 3: Simplify Tree Navigation Handler
- [x] **Update `HandleTreeViewNavigation` in `handlers/search-tree.go`:**
  - [x] Remove complex `filterAdsForChildPath` logic
  - [x] Remove `matchesChildPath` function
  - [x] Simplify to use new SQL functions
  - [x] Handle both browse mode (`q==""`) and search mode (`q!=""`)

- [x] **Tree navigation logic:**
  - [x] Parse tree path from URL (`/tree/{make}/{year}/{model}/{engine}/{category}/{subcategory}`)
  - [x] Determine if we're in browse mode (`q==""`) or search mode (`q!=""`)
  - [x] For search mode: Get ad IDs from vector search, then filter tree nodes
  - [x] For browse mode: Use unfiltered SQL queries
  - [x] Return appropriate tree nodes for the current level

### Phase 4: Update Tree UI Components
- [ ] **Simplify tree node components in `ui/tree.go`:**
  - [ ] Remove threshold handling (not needed for tree view)
  - [ ] Standardize expansion/collapse behavior
  - [ ] Clean up URL parameter handling

- [ ] **Update tree view rendering in `ui/view-tree.go`:**
  - [ ] Simplify `TreeViewRenderResults` - always render tree container
  - [ ] Remove complex logic for empty vs non-empty results
  - [ ] Ensure tree loads via HTMX regardless of search state

### Phase 5: Clean Up and Test
- [ ] **Remove unused functions:**
  - [ ] Remove `getTreeAdsForSearch` and `getTreeAdsForSearchWithFilter`
  - [ ] Remove `filterAdsForChildPath` and `matchesChildPath`
  - [ ] Clean up any other unused tree-related functions

- [ ] **Testing:**
  - [ ] Test browse mode (`q==""`) - should show all makes/years/models/etc
  - [ ] Test search mode (`q!=""`) - should show filtered tree based on vector search
  - [ ] Test tree expansion at each level
  - [ ] Test tree collapse functionality
  - [ ] Verify performance improvements

### Phase 6: Documentation
- [ ] **Update code comments:**
  - [ ] Document the new tree view architecture
  - [ ] Explain browse vs search mode differences
  - [ ] Document the SQL function parameters and return values

## Implementation Notes

### Tree View Architecture
- **Browse Mode (`q==""`)**: Pure SQL-based tree building, no vector search
- **Search Mode (`q!=""`)**: Vector search gets ad IDs, then SQL queries filter tree nodes by those IDs
- **Tree Expansion**: Each level uses appropriate SQL function based on current path and mode

### Route Structure (Keep Existing)
- `/view/tree` - Main tree view handler (calls TreeView.GetAdIDs())
- `/tree` - HTMX tree root expansion (shows makes)
- `/tree/*` - HTMX tree node expansion (shows children for current path)
- `/tree-collapsed/*` - HTMX tree node collapse

### Key Benefits
1. **Clear separation**: Browse mode vs Search mode logic
2. **Efficient queries**: SQL-based filtering instead of in-memory filtering
3. **Consistent behavior**: Same tree structure regardless of search mode
4. **Maintainable**: Each level has its own clear responsibility
5. **Performance**: No unnecessary vector searches for tree navigation

## Success Criteria
- [ ] Tree view works correctly in both browse and search modes
- [ ] Tree expansion is fast and efficient
- [ ] Code is clean and maintainable
- [ ] No performance regressions
- [ ] All existing functionality preserved
