# Tech Stack

This document outlines the technologies, frameworks, and tools used in the Parts Pile web application.

## Backend

### Programming Language
- **Go 1.27.3** - Primary programming language

### Web Framework
- **Fiber v2.52.5** - High-performance HTTP web framework built on top of fasthttp
  - Fast HTTP server with low memory footprint
  - Middleware support for security, logging, rate limiting

### Database
- **PostgreSQL** - Relational database (managed or local)
  - Connection via `DATABASE_URL` environment variable
  - JSON stored in `TEXT` columns; queried with Postgres `json_*` functions
  - Foreign key constraints enabled
  - Indexes optimized for search performance

### Cache / shared rate limits
- **Redis** - Shared counters for registration/recovery IP limits and per-phone
  OTP starts (`REDIS_URL`, required). Docker Compose service `redis` on
  port 6379.

### Database Libraries
- **sqlx v1.4.0** - SQL toolkit providing extensions to database/sql
  - Named parameter support
  - Struct scanning capabilities
- **jackc/pgx/v5** - PostgreSQL driver for Go (via `database/sql`)

### Authentication & Security
- **golang-jwt/jwt/v5 v5.3.0** - JSON Web Token implementation
  - HS256 signing method
  - 24-hour token expiration
  - Secure cookie-based storage
- **Argon2** (via `golang.org/x/crypto`) - Password hashing
  - Memory: 64 KB
- **AES-GCM** (via `golang.org/x/crypto`) - Encryption for sensitive data
  - 256-bit keys
  - HKDF-SHA256 for key derivation
  - Used for message and user data encryption
- **HMAC-SHA256** - Peppered `name_hash` / `phone_hash` lookups (`DB_HASH_PEPPER`)

### Security Middleware
- **Helmet** (Fiber middleware) - Security headers
  - Content Security Policy (CSP)
  - X-Frame-Options: DENY
  - HSTS (when HTTPS enabled)
  - Referrer-Policy: strict-origin-when-cross-origin
- **CSRF Protection** (Fiber middleware) - Double-submit cookie pattern
- **Rate Limiting** (Fiber middleware)
  - General: 600 requests per minute
  - Registration: 3 attempts per 15 minutes

### Logging
- **slog** (Go standard library) - Structured logging
  - JSON and text formats supported
  - Color output for terminal (text format)
  - Configurable log levels (debug, info, warn, error)
  - File or stdout output

### HTTP Client & Compression
- **fasthttp v1.51.0** - Fast HTTP implementation (used by Fiber)
- **brotli v1.0.5** - Brotli compression support
- **klauspost/compress v1.17.0** - Compression algorithms

## Frontend

### JavaScript
- **HTMX v2.0.10** — HTML-over-the-wire (self-hosted in `static/js/`)
- **First-party scripts** for ad image upload: client-side JPEG resize/encode and MinIO presigned PUT with progress (`image-preview.js`, `image-upload.js`)
- SSE extension for live updates
- **HATEOAS** — server-driven hypermedia for most flows; image upload uses authenticated JSON + direct-to-MinIO PUTs

### CSS Framework
- **Tailwind CSS** - Utility-first CSS framework
  - Inline styles (via `unsafe-inline` CSP directive)

### HTML Generation
- **gomponents** - Type-safe HTML generation library
  - `maragu.dev/gomponents` - Core HTML components
  - `maragu.dev/gomponents-htmx` - HTMX-specific attributes
  - `maragu.dev/gomponents/components` - Pre-built component library
  - Used for server-side HTML rendering in MPA (Multi-Page Application) architecture

### Static Assets
- Static files served directly by Fiber
- Located in `./static` directory

## External Services & APIs

### Object Storage
- **MinIO** - S3-compatible object storage for ad images
  (`cgr.dev/chainguard/minio:latest`; community `minio/minio` images
  are no longer published)
  - `MINIO_API_URL` — server-side S3 API (private network in production)
  - `MINIO_PUBLIC_URL` — host embedded in browser-facing presigned URLs
  - **PUT:** short-lived (~15m) presigned uploads from the browser after create/edit; client encodes 160/480/1200 JPEG derivatives
  - **GET:** long-lived (~24h) presigned URLs reused in-process so `img src` stays stable and browsers can cache; app does not proxy image bodies
  - Bucket CORS must allow PUT (and typically GET) from the site origin

### Communication Services
- **Twilio Verify** — Registration and change-phone OTP (Fraud Guard + geo permissions)
- **Twilio Programmable Messaging** — Unread alerts, inbound STOP/RECOVER webhooks
  - Account SID and Auth Token authentication
  - See `doc/DOC_SMS_OTP_AND_PUMPING_DEFENSES.md`
- **Cloudflare Turnstile** — Bot challenge before OTP start

### AI/ML Services
- **Ollama embeddings** - Local `nomic-embed-text` (768-dim) for ad and query vector search
  - Endpoint: `http://localhost:11434` (configurable via `OLLAMA_URL`)
  - Runs via the `ollama` Docker Compose service; no external API key required
  - Uses `search_query:` / `search_document:` prefixes for asymmetric retrieval
- **Grok API (x.ai)** - Chat completions
  - Model: `grok-3-mini`
  - Endpoint: `https://api.x.ai/v1/chat/completions`
- **Geoapify** - Forward geocoding for locations table cache misses
  - Endpoint: `https://api.geoapify.com/v1/geocode/search`
  - Env: `GEOAPIFY_API_KEY`

## Infrastructure & Deployment

### CI/CD
- **GitHub Actions** - Continuous integration
  - Automated testing on push/PR to main/dev branches
  - Go 1.27.3 setup
  - Module caching
  - Database rebuild with test data
  - Test execution with required environment variables

### Hosting
- **Render** - Cloud hosting platform (referenced in configuration)
  - Environment variable detection for Render deployments
  - External URL configuration support

## Development Tools

### Testing
- Go standard testing framework (`go test`)
- Test database rebuild via package `TestMain` / fixtures
- Test data generation support

### Build Tools
- Go modules (`go.mod` / `go.sum`)
- Standard Go build toolchain

## Architecture Patterns

### Frontend Architecture
- **Server-Side Rendering (SSR)** - All HTML is generated server-side
- **HTML-over-the-Wire** - Dynamic updates via HTMX without custom JavaScript
- **Zero JavaScript Framework** - No React, Vue, Angular, or other JS frameworks
- **Declarative Interactivity** - All UI interactions defined via HTML attributes (hx-get, hx-post, etc.)
- **HATEOAS (Hypermedia As The Engine Of Application State)** - Core architectural principle where application state and available actions are embedded in the HTML responses themselves. The server controls application flow by returning hypermedia (links, forms) that represent valid state transitions, eliminating the need for client-side routing or API endpoint knowledge.

### Application Structure
- **Layered Architecture**
  - Handlers (HTTP request handling, map domain data to UI inputs)
  - Domain packages (business logic, data access)
  - UI (`ui/`, server-side HTML rendering via gomponents)
  - Database (PostgreSQL via sqlx + pgx)
- **Middleware Chain**
  - Security headers (Helmet)
  - Rate limiting
  - JWT authentication
  - CSRF protection
  - Request logging

### UI Layer Boundary
Request flow: **handler → domain → handler maps to UI inputs → `ui/` renders**.

- **`ui/` must not import domain packages** (`ad`, `user`, `message`, etc.)
- **Exceptions:** `ui/` may import `facet/` (shared field/filter definitions for forms and search UI) and `config/` (app constants such as max field lengths and server name)
- **Domain packages must not import `ui/`**
- **Exported UI functions** accept only:
  - Go primitives and stdlib types (`int`, `string`, `bool`, `time.Time`, …)
  - Handler→UI boundary structs (e.g. `UserProfileData`, `AdCard` in [`ui/types.go`](ui/types.go); `SearchFilters` in [`ui/ads/types.go`](ui/ads/types.go))
- **Boundary structs** — defined in `types.go` at the package that owns the render entry point. Handlers map domain data to these before calling `ui/`. Keep them separate from **internal UI structs** (e.g. `SMSQueueEntry` in `admin.go`, derived inside `ui/` from `SMSQueueEntryInput` via `SMSQueueEntriesFrom`).
- **Do not pass `*time.Location` to exported UI functions** — it is request context, not presentation data
- **Time presentation** (default: handler maps; UI renders):
  - **Fixed-format display strings** (e.g. member since date): pre-format in the handler and set a `string` field on the view struct (`UserProfileData.MemberSince`)
  - **Relative or multi-format times** (e.g. `"5m ago"` plus a tooltip): handler converts to the viewer's timezone (`t.In(loc)`) on the view struct's `time.Time` field; UI applies shared relative formatting at render time
- **Handlers** fetch domain objects, then map them to UI view structs (or primitives) before calling render functions
- **Presentation formatting** (dates, currency, derived display strings) belongs in the handler mapper or `ui/`, not in domain types passed across the boundary

### Data Patterns
- **Normalized Database** - Users, categories, ads, locations, bookmarks, conversations
- **Hard-filter search** - Category, price range, location + radius (bounding box on lat/lon); interim text match on title/description (`q`)
- **Semantic search** - Postgres pgvector cosine similarity on ad embeddings; metadata filters via `vector_metadata` JSONB; query/user/site embedding modes for empty search box. See [DOC_EMBEDDINGS.md](DOC_EMBEDDINGS.md).

### Security Patterns
- **Encryption at Rest** - AES-GCM for sensitive user data
- **Key Derivation** - HKDF for per-user encryption keys
- **Secure Cookies** - HTTPOnly, Secure, SameSite=Strict
- **Token-based Auth** - JWT stored in secure cookies
- **Double-Submit Cookie** - CSRF protection pattern

## Configuration

Configuration is managed through environment variables:
- **Database connection** — set `DATABASE_URL` to a PostgreSQL DSN (e.g. `postgres://localhost:5432/partspile?sslmode=disable` for local dev). For local Postgres via Docker: `docker compose up -d postgres`, then use `postgres://postgres:postgres@localhost:5432/partspile?sslmode=disable`. Restore production/dev data via the Admin TUI. Integration tests live in `cmd/server` (`integration_*.go`) and share one fixture database via `TestMain`.
- MinIO credentials
- API keys (Grok, Geoapify, Twilio)
- Ollama server URL (`OLLAMA_URL`, default `http://localhost:11434`)
- JWT secrets
- Encryption keys (base64-encoded)
- Server port and logging settings

See `config/config.go` for complete configuration options.

