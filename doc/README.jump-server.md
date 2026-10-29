# Jump Server

The jump server image includes database utilities (`psql`), Redis CLI (`redis-cli`), and the MinIO client (`mc`).

## Building

```bash
docker build -f dockerfiles/Dockerfile.jump-server -t partspile-jump-server:dev .
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
- **Restore** — pick a file from `backups/`, optionally enter source `BACKUP_DB_ENCRYPTION_KEY`, confirm wipe, restore. Leave the key blank to decrypt with `DB_ENCRYPTION_KEY`.
- **Promote / Demote admin** — by username

`./backups` on the host is mounted at `/workspace/backups`.

### Encryption keys

| Env var | Use |
|---------|-----|
| `DB_ENCRYPTION_KEY` | Live DB crypto (name/phone; journals sealed on write/restore) |
| `DB_HASH_PEPPER` | HMAC pepper for `name_hash` / `phone_hash` (must differ from encryption key) |
| `BACKUP_DB_ENCRYPTION_KEY` | Decrypt archive user fields when restoring from another env (prompted in TUI; empty uses `DB_ENCRYPTION_KEY`) |

Backup verify-decrypts users with `DB_ENCRYPTION_KEY` and stores ciphertext in the archive. Restore re-keys users to the target `DB_ENCRYPTION_KEY`, recomputes peppered lookup hashes with `DB_HASH_PEPPER`, and seals conversation journals. Restore resets the DB (schema + categories) before import. Ad embeddings and `vector_metadata` are included in the archive (base64 pgvector binary on each ad) so search works without recomputing after restore. Older archives without embeddings still restore; those ads get vectors from an admin embeddings backfill. Account pictures (`users/{id}/account.jpg`) are stored under `user_accounts/{name_hash}/` and re-uploaded to the restored user IDs. Older archives without that directory still restore; users keep `has_account_picture` and restore logs a warning for each missing file.

## Ad images

Images are stored in MinIO (required). Populate images from your dev machine with `go run ./cmd/gen_images` (requires `FAL_API_KEY`). On the jump server, mirror existing objects with `mc` (see below).

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
mc mb local/partspile --ignore-existing
mc mirror static/images/ad/ local/partspile/
mc --version
mc ls local/partspile
mc cp local/partspile/23/1-480w.jpg ./
```

See the [MinIO Client documentation](https://docs.min.io/docs/minio-client-quickstart-guide.html).

## Binaries

| Binary | Purpose |
|--------|---------|
| `admin` | Interactive TUI: backup/restore, init DB, promote/demote |
| `dumpusers` | CSV of decrypted names, phones, and password hashes |
| `quote_server` | Quote-of-the-day page on port 10000 |
| `mc` | MinIO command-line client |
| `redis-cli` | Redis / Render Key Value CLI (`redis-tools`) |
| `psql` | PostgreSQL client |
