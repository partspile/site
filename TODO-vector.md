# TODO: Switch to Vector Embedding Search

## 1. Embedding Infrastructure
- [x] Add dependency: google.golang.org/genai for Gemini embedding API
- [x] Add dependency: Pinecone Go client for vector DB
- [x] Create utility for generating ad embeddings using gemini-embedding-001 (now using embedding-001)
- [x] Create utility for generating query/user embeddings (utility exists, user/query logic next)

## 2. Ad Embedding Management
- [x] On ad create/edit, generate embedding and upsert to Pinecone (background job, upsert logic implemented)
- [x] Add migration to backfill embeddings for all existing ads (backfill job implemented)
- [x] Store ad ID and metadata (created_at, click_count, etc.) in Pinecone (metadata upserted, type-safe)

## 3. User Embedding & Personalization
- [x] Fetch recent search queries for user and generate embeddings
- [x] Fetch recently clicked ads and bookmarked ads for user
- [x] Aggregate embeddings (weighted mean: bookmarks > clicks > searches)
- [x] Use this as the user vector for personalized search when q==""
- [x] Store user embeddings in the database (UserEmbedding table) and update after every search

## 4. Query Embedding
- [x] On search (q!=""), generate embedding for query and search Pinecone

## 5. Search Handler Changes
- [x] Replace SQL-based search with Pinecone vector search for q!="" and personalized feed
- [x] For anonymous users with q=="", use recency/click-based or embedding-based diversity feed

## 6. Background Jobs
- [x] Ensure all embedding generation and Pinecone upserts are done asynchronously
- [x] Add error logging and retry for background jobs

## 7. Documentation
- [x] Update PRD.md to describe new vector-based search and personalization
- [x] Update TODO.md to reference vector search migration

## 8. Pagination & Infinite Scroll
- [ ] Ensure infinite scroll (pagination) works with Pinecone vector search
- [ ] Use Pinecone's cursor-based pagination if available
- [ ] If not, implement offset/ID-based pagination as fallback
- [ ] Update search handlers and UI to support paginated loading of results from Pinecone 

---

**Next up:**
- Focus on UI/UX improvements, infinite scroll, and advanced features for vector search (e.g., diversity feed for anonymous users, trending topics, etc.)
- All core vector embedding and personalization logic is complete and live. 