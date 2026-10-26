# Jump Server

The jump server image includes database utilities (`psql`), Redis CLI (`redis-cli`), image migration, JPEG conversion, description-addition fold, and the MinIO client (`mc`).

## Building

```bash
docker build -f dockerfiles/Dockerfile.jump-server -t rockyads-jump-server:dev .
```

With docker-compose (includes Postgres and MinIO):

```bash
docker compose up -d
```

## Admin tool

Interactive TUI (requires a TTY):

```bash
# local compose
docker compose exec -it jump-server admin

# Render / SSH — allocate a pseudo-TTY
ssh -t <jump-host> admin
```

Without a TTY, `admin` exits immediately (stdin EOF). Use `ssh -t` or an equivalent that requests a PTY.

Menu:

- **Backup** — writes `backups/backup-YYYYMMDD-HHMMSS.tar.gz`
- **Restore** — pick a file from `backups/`, optionally enter source `BACKUP_DB_ENCRYPTION_KEY`, confirm wipe, restore. Leave the key blank and put `users.csv` in `backups/` to fill names and phones from that file instead.
- **Init database** — wipe and reload schema + categories (or with seed users/ads)
- **Promote / Demote admin** — by username

`./backups` on the host is mounted at `/workspace/backups`.

### Encryption keys

| Env var | Use |
|---------|-----|
| `DB_ENCRYPTION_KEY` | Live DB crypto (name/phone; journals sealed on write/restore) |
| `DB_HASH_PEPPER` | HMAC pepper for `name_hash` / `phone_hash` (must differ from encryption key) |
| `BACKUP_DB_ENCRYPTION_KEY` | Decrypt archive user fields when restoring from another env (prompted in TUI; empty uses `backups/users.csv` if present, otherwise `DB_ENCRYPTION_KEY`) |

Backup verify-decrypts users with `DB_ENCRYPTION_KEY` and stores ciphertext in the archive. Restore re-keys users to the target `DB_ENCRYPTION_KEY`, recomputes peppered lookup hashes with `DB_HASH_PEPPER`, and seals conversation journals. Restore resets the DB like init (schema + categories) before import. Ad embeddings and `vector_metadata` are included in the archive (base64 pgvector binary on each ad) so search works without recomputing after restore. Older archives without embeddings still restore; those ads get vectors from the server backfill on startup. Account pictures (`users/{id}/account.jpg`) are stored under `user_accounts/{name_hash}/` and re-uploaded to the restored user IDs. Older archives without that directory still restore; users keep `has_account_picture` and restore logs a warning for each missing file.

## Ad images

Images are stored in MinIO (required). After seeding the database, populate images from your dev machine with `go run ./cmd/gen_images` (requires `FAL_API_KEY`) or upload existing local files with `migrate_images` on the jump server (see below).

### Migrate existing local files (one-time)

If you have files under `static/images/ad/` (or another directory):

```bash
migrate_images -dir static/images/ad
```

Use `-dry-run` to preview without uploading. MinIO connection uses `MINIO_API_URL`, `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD`, and `MINIO_BUCKET_NAME` from the service environment.

On the jump server, you can also mirror files with `mc` (see below).

### Convert legacy WebP/PNG objects to JPEG

Stop the site first. Then on the jump server:

```bash
convert_images -dry-run
convert_images
```

Rewrites MinIO objects in place to `.jpg` (`image/jpeg`, quality 80). Idempotent. Uses `MINIO_API_URL`, `MINIO_ROOT_USER`, `MINIO_ROOT_PASSWORD`, and `MINIO_BUCKET_NAME`. Restart the JPEG-only site after it finishes.

### Fold legacy description additions (one-time)

Append-only description edits used to store extra paragraphs as history
blocks. Fold them into the listing body, then drop those blocks:

```bash
migrate_desc_additions -dry-run
migrate_desc_additions
```

Idempotent. Uses `DATABASE_URL`. Run after restore if the archive predates this fold.

## Using redis-cli

`redis-tools` is pre-installed. With `REDIS_URL` in the environment (BASE env group on Render; compose sets `redis://redis:6379`):

```bash
redis-cli -u "$REDIS_URL"
redis-cli -u "$REDIS_URL" PING
redis-cli -u "$REDIS_URL" --scan --pattern 'otp:*'
```

Prefer `--scan` / `SCAN` over `KEYS *`. Rate-limit keys look like `otp:cd:<e164>`, `otp:hr:<e164>`, `reg:<ip>`, `rec:<ip>`, `srv:<ip>`.

Local alternative without jump: `docker compose exec redis redis-cli`.

## Using mc

The MinIO client is pre-installed:

```bash
mc alias set local http://minio:9000 minioadmin minioadmin
mc mb local/rockyads --ignore-existing
mc mirror static/images/ad/ local/rockyads/
mc --version
mc ls local/rockyads
mc cp local/rockyads/23/1-480w.jpg ./
```

See the [MinIO Client documentation](https://docs.min.io/docs/minio-client-quickstart-guide.html).

## Binaries

| Binary | Purpose |
|--------|---------|
| `admin` | Interactive TUI: backup/restore, init DB, promote/demote |
| `dumpusers` | CSV of decrypted names, phones, and password hashes |
| `migrate_images` | One-time upload of local ad image files to MinIO |
| `convert_images` | One-time MinIO WebP/PNG → JPEG rewrite |
| `migrate_desc_additions` | One-time fold of appended description history into the listing body |
| `quote_server` | Quote-of-the-day page on port 10000 |
| `mc` | MinIO command-line client |
| `redis-cli` | Redis / Render Key Value CLI (`redis-tools`) |
| `psql` | PostgreSQL client |
