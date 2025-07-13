# TODO: Switch to Vector Embedding Search

## 1. Embedding Infrastructure
- [ ] Add dependency: google.golang.org/genai for Gemini embedding API
- [ ] Add dependency: Pinecone Go client for vector DB
- [ ] Create utility for generating ad embeddings using gemini-embedding-001
- [ ] Create utility for generating query/user embeddings

## 2. Ad Embedding Management
- [ ] On ad create/edit, generate embedding and upsert to Pinecone (background job)
- [ ] Add migration to backfill embeddings for all existing ads
- [ ] Store ad ID and metadata (created_at, click_count, etc.) in Pinecone

## 3. User Embedding & Personalization
- [ ] Fetch recent search queries for user and generate embeddings
- [ ] Fetch recently clicked ads and bookmarked ads for user
- [ ] Aggregate embeddings (weighted mean: bookmarks > clicks > searches)
- [ ] Use this as the user vector for personalized search when q==""

## 4. Query Embedding
- [ ] On search (q!=""), generate embedding for query and search Pinecone

## 5. Search Handler Changes
- [ ] Replace SQL-based search with Pinecone vector search for q!="" and personalized feed
- [ ] For anonymous users with q=="", use recency/click-based or embedding-based diversity feed

## 6. Background Jobs
- [ ] Ensure all embedding generation and Pinecone upserts are done asynchronously
- [ ] Add error logging and retry for background jobs

## 7. Documentation
- [ ] Update PRD.md to describe new vector-based search and personalization
- [ ] Update TODO.md to reference vector search migration

## 8. Pagination & Infinite Scroll
- [ ] Ensure infinite scroll (pagination) works with Pinecone vector search
- [ ] Use Pinecone's cursor-based pagination if available
- [ ] If not, implement offset/ID-based pagination as fallback
- [ ] Update search handlers and UI to support paginated loading of results from Pinecone 