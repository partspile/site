# Parts Pile Site

A Go-based auto parts marketplace built with Fiber, HTMX, and Gomponents.

## Environment Variables

### Database Configuration
- `DATABASE_URL` - Database connection string (default: `project.db` for SQLite)
  - SQLite: `project.db` or `file:project.db`
  - PostgreSQL: `postgresql://username:password@localhost:5432/dbname`

### MinIO Configuration
- `MINIO_API_URL` - MinIO API endpoint URL (e.g., `http://localhost:9000` or `https://<server>:<port>`)
- `MINIO_USERNAME` - MinIO username
- `MINIO_PASSWORD` - MinIO password
- `MINIO_BUCKET_NAME` - MinIO bucket name for image storage

### Vector Database Configuration
- Vector embeddings are stored in PostgreSQL/CockroachDB using the pgvector extension
- No additional configuration needed - vectors are stored in the `Ad` table

### AI/ML API Configuration
- `GEMINI_API_KEY` - Google Gemini API key for text embeddings
- `GROK_API_KEY` - Grok API key for AI prompts

### Server Configuration
- `PORT` - HTTP server port (default: `8000`)

## Development

```bash
# Run the application
go run ./

# Run tests
go test ./...
```