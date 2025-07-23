# Parts Pile Site

A Go-based auto parts marketplace built with Fiber, HTMX, and Gomponents.

## Environment Variables

### Database Configuration
- `DATABASE_URL` - Database connection string (default: `project.db` for SQLite)
  - SQLite: `project.db` or `file:project.db`
  - PostgreSQL: `postgresql://username:password@localhost:5432/dbname`

### Backblaze B2 Configuration
- `BACKBLAZE_MASTER_KEY_ID` - Backblaze B2 master key ID
- `BACKBLAZE_KEY_ID` - Backblaze B2 key ID  
- `BACKBLAZE_APP_KEY` - Backblaze B2 application key
- `B2_BUCKET_ID` - Backblaze B2 bucket ID for image storage

### Vector Database Configuration (Pinecone)
- `PINECONE_API_KEY` - Pinecone API key for vector search
- `PINECONE_INDEX` - Pinecone index name

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