# Parts Pile Web Site

[Parts Pile](https://parts-pile.com) is classified ads for the Internet.

<p align="center">
  <img src="static/images/classifieds.jpg" alt="Newspaper classifieds" height="500">
</p>

Remember classified ads? Easy, simple. Post an ad with your phone number and folks contact you. Parts Pile works the same way, except your phone number stays hidden.

&emsp;🚫 No email  
&emsp;🚫 No Facebook friends  
&emsp;🚫 No posting fees  
&emsp;🚫 No credit cards

All you need is your phone number to [get started](https://parts-pile.com/register).

## Development

```mermaid
flowchart LR
  Browser -->|HTTP / HTMX| Site["Site<br/>Go / Fiber"]
  Browser -->|presigned PUT/GET| MinIO
  Browser -->|Turnstile| Cloudflare["Cloudflare<br/>Turnstile"]
  Site --> Postgres[(Postgres)]
  Site --> Redis[(Redis)]
  Site --> MinIO
  Site --> Ollama
  Site --> Twilio
  Site --> Cloudflare
  Site --> Grok["Grok API<br/>x.ai"]
  Site --> Geoapify
  Twilio -->|SMS webhooks| Site
```

### Prerequisites

- Go 1.27.3+ — https://go.dev/dl
- Node.js (includes npm) — https://nodejs.org, or e.g. `brew install node` on macOS

### CSS (Tailwind)

The app uses Tailwind CSS v4 (listed in `package.json`; no need to install it separately). Install deps once, then build the stylesheet before or while running the server:

1. `npm install`
2. One-off build: `npm run build-css-dev`
   - Or watch mode (rebuild on changes): `npm run build-css`
   - Or production (minified): `npm run build-css-prod`

Input: `input.css` → output: `static/css/output.css`.

### Object storage (MinIO)

Ad images are stored in MinIO (`cgr.dev/chainguard/minio:latest`). The server requires `MINIO_API_URL` (and related credentials) to start.

- `MINIO_API_URL` — S3 API endpoint used by the server (private-network host in production).
- `MINIO_PUBLIC_URL` — browser-facing host used when minting presigned PUT/GET URLs (e.g. `https://minio.parts-pile.com` or `http://127.0.0.1:9000` locally). Defaults to `MINIO_API_URL` if unset.

Browsers upload JPEG derivatives via short-lived **presigned PUT** URLs. Listing/detail pages embed **reused long-lived presigned GET** URLs so image bytes come from MinIO and remain browser-cacheable.

### SMS (Twilio)

Phone verification (register / change-phone) uses **Twilio Verify**. Account
recovery inbound SMS and unread-message alerts use **Programmable Messaging**.
See [doc/DOC_SMS_OTP_AND_PUMPING_DEFENSES.md](doc/DOC_SMS_OTP_AND_PUMPING_DEFENSES.md).

Always required at server start:

- `PUBLIC_SITE_URL` — public base URL for the site (non-localhost for real SMS). Used for Twilio webhooks (`…/api/sms/webhook`) and SMS deep links (e.g. `…/auth/user/messages`). Local: your ngrok (or similar) HTTPS URL.

Required at server start unless `ALLOW_TEST_REGISTRATION=true`:

- `TWILIO_ACCOUNT_SID` — Twilio account SID (must start with `AC`).
- `TWILIO_AUTH_TOKEN` — Twilio auth token.
- `TWILIO_FROM_NUMBER` — Messaging sender in E.164 (e.g. `+12025550123`).
- `TWILIO_VERIFY_SERVICE_SID` — Verify Service SID (must start with `VA`).

### Turnstile (Cloudflare)

Bot gate before OTP start (register / change-phone). Required at server start unless `ALLOW_TEST_REGISTRATION=true`:

- `TURNSTILE_SITE_KEY` — public site key (browser widget).
- `TURNSTILE_SECRET_KEY` — secret key (server-side verification).

### DB (Postgres)

- `DATABASE_URL` — PostgreSQL DSN. Default: `postgres://localhost:5432/partspile?sslmode=disable`. With docker-compose Postgres: `postgres://postgres:postgres@localhost:5432/partspile?sslmode=disable`.
- `DB_ENCRYPTION_KEY` — required. Base64-encoded 32-byte (256-bit) key for encrypting user name/phone.
- `DB_HASH_PEPPER` — required. Base64-encoded 32-byte (256-bit) HMAC pepper for `name_hash` / `phone_hash` lookups. Must differ from `DB_ENCRYPTION_KEY`.
- `SHARE_SECRET` — required. Base64-encoded 32-byte (256-bit) key for opaque `/u/…` profile share links. Must differ from `DB_ENCRYPTION_KEY` and `DB_HASH_PEPPER`.

### Redis

- `REDIS_URL` — required. Redis URL for shared rate limits (registration/recovery/login IP + per-username login failures + per-phone OTP starts). Local: `redis://localhost:6379` with `docker compose up -d redis`. Render: Key Value internal URL (e.g. `redis://red-xxx:6379`).

Inspect Redis:

```bash
# local compose (Redis container)
docker compose exec redis redis-cli

# jump server (rebuild image for redis-tools)
docker compose exec -it jump-server sh -c 'redis-cli -u "$REDIS_URL"'
# on Render SSH / shell (REDIS_URL from BASE):
redis-cli -u "$REDIS_URL"
```

### LLM / geocoding APIs

- `GROK_API_KEY` — xAI Grok API key (chat completions for rock opinions / AI features).
- `GEOAPIFY_API_KEY` — Geoapify API key (forward geocoding for locations table cache misses).
- `OLLAMA_URL` — Ollama HTTP endpoint for embeddings (`nomic-embed-text`). Default: `http://localhost:11434`.
- `FAL_API_KEY` — fal.ai key for `go run ./cmd/gen_images` only (not required by the server).

### JWT

- `JWT_SECRET` — required. Secret for signing auth cookies; at least 32 characters.

### Other settings

- `PORT` — HTTP listen port. Default: `10000`.
- `PORT_TEST` — port used by integration tests. Default: `10001`.
- `APP_NAME` — required. Display name shown in the UI.
- `CONTACT_EMAIL` — required. Contact address shown in the UI.
- `LOCAL_DEVELOPMENT` — set to `true` so auth cookies work over plain HTTP (otherwise cookies are Secure-only).
- `ALLOW_TEST_REGISTRATION` — set to `true` to skip SMS verification for `+1555010xxxx` phones and relax Twilio startup checks (dev/test only).
- `LOG_LEVEL` — `debug` / `info` / … Default: `info`.
- `LOG_FORMAT` — `json` or text. Default: `json`.
- `LOG_FILE` — optional log file path; empty logs to stderr.

### Local dev with docker-compose

```bash
docker compose up -d   # postgres, redis, minio, jump-server, ollama, rockyads
```

Set in `.env`:

```
DATABASE_URL=postgres://postgres:postgres@localhost:5432/partspile?sslmode=disable
REDIS_URL=redis://localhost:6379
DB_ENCRYPTION_KEY=AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=
DB_HASH_PEPPER=AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=
SHARE_SECRET=AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=
JWT_SECRET=local-dev-jwt-secret-key-min-32-chars
PUBLIC_SITE_URL=http://localhost:10000
APP_NAME=Parts Pile
CONTACT_EMAIL=contact@parts-pile.com
LOCAL_DEVELOPMENT=true
ALLOW_TEST_REGISTRATION=true
MINIO_API_URL=http://127.0.0.1:9000
MINIO_PUBLIC_URL=http://127.0.0.1:9000
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_BUCKET_NAME=partspile
OLLAMA_URL=http://localhost:11434
```

`REDIS_URL` is required; start Redis via Compose before running the server.

Add Twilio (`TWILIO_*`), `GROK_API_KEY`, `GEOAPIFY_API_KEY`, and `FAL_API_KEY` when you need real SMS, Grok, location geocoding, or image generation. For real SMS locally, set `PUBLIC_SITE_URL` to an ngrok (or similar) HTTPS URL so Twilio can reach `/api/sms/webhook`.

Restore a database backup via the jump-server admin TUI:

```bash
docker compose exec -it jump-server admin
```

Main menu: **Database** · **Users** · **Embeddings**. Under Database, choose **Restore**. User PII and promote/demote/delete live under Users (not the web admin dashboard).

Run the site (loads `.env` into the environment):

```bash
set -a && source .env && set +a
go run ./cmd/server
```

Open http://localhost:10000.

The `rockyads` container is the rockyads.com stand-in. It listens on
http://localhost:8010 and serves a rename landing page (link to
`PUBLIC_SITE_URL`, no auto-redirect). `/health` is for probes.
