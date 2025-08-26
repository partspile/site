# **TODO: Embedding Cache Consolidation Migration**

## **Phase 1: Add Three Cache Instances (Parallel Implementation)**

### **1.1 Update vector/embedding.go**
- [x] **Add new cache variables** alongside existing `embeddingCache`:
  ```go
  var (
      // Keep existing for backward compatibility during migration
      embeddingCache *cache.Cache[[]float32]
      
      // New specialized caches
      queryEmbeddingCache *cache.Cache[[]float32]  // String keys, 1 hour TTL
      userEmbeddingCache  *cache.Cache[[]float32]  // User ID keys, 24 hour TTL  
      siteEmbeddingCache  *cache.Cache[[]float32]  // Campaign keys, 6 hour TTL
  )
  ```

- [x] **Update InitEmbeddingCache()** to initialize all three caches:
  ```go
  func InitEmbeddingCaches() error {
      // Initialize query cache (1 hour TTL, smaller size)
      // Initialize user cache (24 hour TTL, larger size)  
      // Initialize site cache (6 hour TTL, medium size)
      // Keep existing embeddingCache initialization for backward compatibility
  }
  ```

- [x] **Add new helper functions**:
  - [x] `GetQueryEmbedding(text string) ([]float32, error)`
  - [x] `GetUserEmbedding(userID int) ([]float32, error)`
  - [x] `GetSiteEmbedding(campaignKey string) ([]float32)`

- [x] **Update existing functions** to use new caches:
  - [x] `GetEmbeddingCacheStats()` - return unified stats from all three caches
  - [x] `ClearEmbeddingCache()` - clear all three caches

### **1.2 Update main.go**
- [x] **Rename function call** from `InitEmbeddingCache()` to `InitEmbeddingCaches()`
- [x] **Verify admin routes** still work with updated function names

### **1.3 Update tests**
- [x] **Update vector/embedding_test.go** to test new cache initialization
- [x] **Add tests** for new helper functions
- [x] **Verify existing tests** still pass

---

## **Phase 2: Migrate Query Embeddings (EmbedTextCached)**

### **2.1 Update vector/embedding.go**
- [x] **Modify EmbedTextCached()** to use `GetQueryEmbedding()` internally
- [x] **Keep function signature** identical for backward compatibility
- [x] **Update TTL** from default 1 hour to explicit 1 hour for query cache

### **2.2 Update all callers of EmbedTextCached**
- [x] **handlers/search.go** - `queryEmbedding()` function
- [x] **handlers/search-tree.go** - tree search functions
- [x] **vector/user_embedding.go** - rock preference embedding generation
- [x] **vector/embedding.go** - site level vector enhancement

### **2.3 Verify query caching behavior**
- [x] **Test cache hits** for identical queries
- [x] **Test TTL expiration** after 1 hour
- [x] **Verify admin stats** show query cache activity

---

## **Phase 3: Migrate User Embeddings (Database → Cache)**

### **3.1 Update vector/user_embedding.go**
- [ ] **Modify GetUserPersonalizedEmbedding()** to use cache first:
  ```go
  func GetUserPersonalizedEmbedding(userID int, forceRecompute bool) ([]float32, error) {
      if !forceRecompute {
          // Try cache first
          if cached, found := GetUserEmbedding(userID); found {
              return cached, nil
          }
      }
      // Fall back to existing DB logic for now
  }
  ```

- [ ] **Add cache storage** after successful embedding generation:
  ```go
  // After successful generation, cache the result
  if err := SetUserEmbedding(userID, embedding); err != nil {
      log.Printf("[warn] failed to cache user embedding: %v", err)
  }
  ```

### **3.2 Update all callers**
- [ ] **handlers/search.go** - `userEmbedding()` function
- [ ] **vector/user_processor.go** - background processing

### **3.3 Test user embedding caching**
- [ ] **Verify cache hits** for same user ID
- [ ] **Test TTL expiration** after 24 hours
- [ ] **Verify fallback** to DB when cache misses

---

## **Phase 4: Migrate Site Embeddings (In-Memory → Cache)**

### **4.1 Update vector/embedding.go**
- [ ] **Modify GetSiteLevelVector()** to use cache:
  ```go
  func GetSiteLevelVector() ([]float32, error) {
      // Use default campaign key for now
      return GetSiteEmbedding("default")
  }
  ```

- [ ] **Add campaign key support** for future extensibility:
  ```go
  func GetSiteEmbedding(campaignKey string) ([]float32, error) {
      // Cache lookup first, then calculation
  }
  ```

### **4.2 Update all callers**
- [ ] **handlers/search.go** - `siteEmbedding()` function
- [ ] **handlers/search-tree.go** - tree search fallback

### **4.3 Test site embedding caching**
- [ ] **Verify cache hits** for same campaign key
- [ ] **Test TTL expiration** after 6 hours
- [ ] **Verify recalculation** when cache expires

---

## **Phase 5: Remove Database Storage for User Embeddings**

### **5.1 Update vector/user_embedding.go**
- [ ] **Remove LoadUserEmbeddingFromDB()** function
- [ ] **Remove SaveUserEmbeddingToDB()** function
- [ ] **Remove math32frombytes()** and `math32tobytes()` helper functions
- [ ] **Update GetUserPersonalizedEmbedding()** to only use cache

### **5.2 Update database schema**
- [ ] **Remove UserEmbedding table** from schema.sql
- [ ] **Add migration script** to drop existing table
- [ ] **Update PRD.md** to reflect schema changes

### **5.3 Test user embedding persistence**
- [ ] **Verify cache persistence** across application restarts
- [ ] **Test user embedding regeneration** when cache is cleared
- [ ] **Verify background processing** still works

---

## **Phase 6: Remove Old In-Memory Site Vector Code**

### **6.1 Update vector/embedding.go**
- [ ] **Remove site-level vector variables**:
  ```go
  // Remove these variables
  var (
      siteLevelVector         []float32
      siteLevelVectorLastCalc time.Time
      siteLevelVectorTTL      = config.QdrantTTL
      siteLevelVectorMutex    sync.RWMutex
  )
  ```

- [ ] **Remove old functions**:
  - [ ] `GetSiteLevelVector()` (replaced by `GetSiteEmbedding()`)
  - [ ] `CalculateSiteLevelVector()` (move logic to new function)

- [ ] **Clean up imports** - remove unused `sync` import

### **6.2 Update config**
- [ ] **Remove QdrantTTL** if no longer used elsewhere
- [ ] **Add new config options** for cache TTLs if needed

---

## **Phase 7: Final Cleanup and Testing**

### **7.1 Remove old embeddingCache**
- [ ] **Remove embeddingCache variable** and all references
- [ ] **Update function names** to remove "Cache" suffix where appropriate
- [ ] **Clean up old cache-related functions**

### **7.2 Update admin interface**
- [ ] **Update ui/admin.go** to display stats from all three caches
- [ ] **Add individual cache clear buttons** if desired
- [ ] **Update cache type labels** to be more descriptive

### **7.3 Comprehensive testing**
- [ ] **Test all search flows**:
  - [ ] Query-based search
  - [ ] User-based search  
  - [ ] Site-based search
  - [ ] Tree-based search
- [ ] **Test cache behavior**:
  - [ ] Hit rates
  - [ ] TTL expiration
  - [ ] Memory usage
  - [ ] Eviction policies
- [ ] **Test admin functions**:
  - [ ] Cache statistics display
  - [ ] Cache clearing
  - [ ] Performance monitoring

### **7.4 Update documentation**
- [ ] **Update PRD.md** to reflect new caching architecture
- [ ] **Add comments** explaining the three-cache design
- [ ] **Document TTL policies** and cache sizing decisions

---

## **Migration Notes**

### **Rollback Strategy**
- Each phase can be rolled back independently
- Keep old code alongside new code during migration
- Use feature flags if needed for gradual rollout

### **Testing Strategy**
- Test each phase thoroughly before proceeding
- Monitor cache performance metrics during migration
- Verify no regression in search quality or performance

### **Performance Considerations**
- Monitor memory usage of three caches vs. old approach
- Adjust cache sizes based on actual usage patterns
- Consider adding cache warming strategies if needed

### **Future Enhancements**
- Add cache persistence across restarts if needed
- Implement cache warming for popular queries/users
- Add cache analytics and alerting

---

## **Progress Tracking**

**Phase 1:** 8/8 tasks completed ✅
**Phase 2:** 7/7 tasks completed ✅  
**Phase 3:** 0/6 tasks completed
**Phase 4:** 0/6 tasks completed
**Phase 5:** 0/6 tasks completed
**Phase 6:** 0/6 tasks completed
**Phase 7:** 0/12 tasks completed

**Overall Progress:** 15/55 tasks completed (27%)
