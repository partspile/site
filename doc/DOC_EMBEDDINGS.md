# Embeddings

How Parts Pile turns ads and activity into vectors, and how empty-box
and typed search use them.

**Code:** `internal/vector/`, `internal/search/`  
**Admin:** Embeddings tab on `/admin/dashboard`  
**Stack note:** [DOC_TECH_STACK.md](DOC_TECH_STACK.md)

---

## Role in search

Listing and search are nearest-neighbor lookup in Postgres (pgvector
cosine distance `<=>`). Every request needs a **query vector**:

| Search box | Who | Query vector |
|---|---|---|
| Non-empty | anyone | Embedding of the typed text |
| Empty | logged-in, has activity in scope | User embedding |
| Empty | anonymous, or user with no activity | Site embedding |

Scope follows the optional category search filter: a selected category
builds a per-category centroid; no category filter uses a **global**
centroid (`categoryID = 0`).

Hard filters (optional category, flag count, optional facets, geo)
apply after the vector is chosen. Ads without an embedding, paused,
deleted, or over the flag limit are excluded.

If no ad embeddings exist yet (backfill still running), search returns
no results instead of HTTP 500.

---

## Model and storage

- **Ollama** `nomic-embed-text`, 768 dimensions (`OLLAMA_URL`, default
  `http://localhost:11434`)
- Ads are stored as `ads.embedding vector(768)` with an HNSW cosine
  index
- `ads.vector_metadata` JSONB holds category and facet values used as
  SQL filters (not part of the vector itself)
- **Asymmetric prefixes:** documents use `search_document:`, typed
  queries use `search_query:`
- User and site vectors are **averages of ad (document) embeddings**,
  then compared to other document embeddings. They are not re-embedded
  as queries.

---

## Ad embeddings

Each active ad is encoded from title, description, tags, facet
snippets, location, price, and a short flag-count sentence.

Rebuilt when an ad is created, edited, reactivated, or gets images.
Paused ads drop their embedding. A background queue (and startup
backfill) fills `embedding IS NULL`. Queue depth and missing ads are
on the admin Embeddings tab.

Changing an ad’s text does **not** immediately refresh user/site
caches. Those recompute on miss or TTL (1 hour), or when an admin
clears them.

---

## User embedding

**Intent:** what this person is looking at (in the selected category,
or across all categories when unfiltered).

Take that user’s bookmarks, ad clicks, and image clicks on active
embedded ads, optionally scoped by category filter. Posting an ad does
**not** count. Your own listing counts only if you bookmarked or
clicked it.

Event weight (then age decay):

| Event | Base | Half-life |
|---|---|---|
| Bookmark | 1.0 | 45 days |
| Ad click | 0.7 | 30 days |
| Image click | 0.4 | 20 days |

`weight = base × exp(-ln(2) / half_life_days × age_days)`

Keep the **top 8 events** (`VectorUserEmbeddingLimit`). The same ad
can appear twice (bookmark + click). Average:

```
user_vec = Σ (weight_i × embedding_i) / Σ weight_i
```

No events → search uses the site embedding. Cache key
`user_{id}_cat_{category}` (`category` is `0` when unfiltered), TTL 1
hour.

---

## Site embedding

**Intent:** what people are interested in (in the selected category,
or site-wide when unfiltered), without letting one front-page ad run
away on repeat clicks.

Same event types and decay, **all users**. Then:

1. **Sum** decayed weights per ad (many users on one ad → one score)
2. Keep the **top 8 ads** (`VectorSiteEmbeddingLimit`)
3. Damp each score: `log(1 + sum)` so extra clicks help less and less
4. Same weighted average of those 8 ad embeddings

No usable activity → average of up to 100 recent ads in scope
(`VectorSystemEmbeddingLimit`). For a specific category that is empty,
fall back to site-wide recent ads.

Cache key `site_cat_{category}_default` (`category` is `0` when
unfiltered), TTL 1 hour. `campaignKey` is accepted but unused.

Top 8 **ads** is not top 8 **events**. With many users, an ad 40
people clicked outranks an ad one person bookmarked yesterday.

---

## Ranking

Postgres orders by `embedding <=> query` (cosine distance). Results
with distance ≥ `SearchThreshold` (0.6) are dropped.

Always filtered by `flag_count <= 2`. Category is applied only when
the category filter is set. Expanded search adds facet predicates on
`vector_metadata`. Location + within still ranks by the same vector;
in-area ads are listed before out-of-area.

Empty-box ranking is **only** this distance. Bookmarks are not pinned
to the front of the list. They influence order only through the
average.

Nearest neighbor to an average is often a **central** ad in the
cluster, not the single highest-weight event. That is expected for a
centroid, not a bug in the weights.

---

## In-memory caches

Three Ristretto caches, all 1 hour TTL:

| Cache | Key | Stores |
|---|---|---|
| Query | trimmed search text | query embedding |
| User | user + category | user embedding |
| Site | category + `default` | site embedding |

New clicks and bookmarks do not invalidate these. Clear them on the
admin Embeddings tab (or wait out the TTL) after changing the
algorithm or when inspecting a fresh mix.

---

## Admin Embeddings tab

- Counts: embedded ads, missing, queue depth
- Cache hit/miss stats
- User inputs: pick user + category; table is the top events that
  feed that user vector
- Site inputs: pick category; table is the top **ads** after
  per-ad sum and `log(1 + sum)` damping
- Dropdowns HTMX-refresh only those tables

---

## Constants

Defined in `internal/config/config.go`:

| Name | Value | Meaning |
|---|---|---|
| `OllamaEmbeddingModel` | `nomic-embed-text` | Model |
| `OllamaEmbeddingDimensions` | 768 | Vector size |
| `SearchThreshold` | 0.6 | Max cosine distance |
| `VectorUserEmbeddingLimit` | 8 | User events in the average |
| `VectorSiteEmbeddingLimit` | 8 | Site ads in the average |
| `VectorSystemEmbeddingLimit` | 100 | Recent-ad fallback |
| `Vector*EmbeddingCacheTTL` | 1 hour | Query / user / site TTL |
| `MaxFlagCount` | 2 | Hidden from search at 3 flags |

---

## Files

| Path | What |
|---|---|
| `internal/vector/ollama.go` | Ollama client, query/document prefixes |
| `internal/vector/ad.go` | Ad prompt, queue, backfill |
| `internal/vector/postgres.go` | Upsert and nearest-neighbor SQL |
| `internal/vector/user_embedding.go` | User events and weights |
| `internal/vector/site_embedding.go` | Site per-ad sum and damping |
| `internal/vector/embedding.go` | Weighted average, recent-ad fallback |
| `internal/vector/cache.go` | Caches and `ResolveSearchEmbedding` |
| `internal/search/search.go` | Ranking and geo |
| `internal/search/metadata.go` | Category / facet / flag filters |
