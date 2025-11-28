# Jump Server with test_db_perf and rebuild_db

This Dockerfile creates a server based on Ubuntu 22.04 that includes pre-built `test_db_perf` and `rebuild_db` binaries, along with a quote server web service.

## Building the Image

```bash
docker build -f Dockerfile.jump-server -t jump-server:latest .
```

## Running Locally

```bash
docker run -d \
  -p 10000:10000 \
  jump-server:latest
```

## CI/CD

The jump server image is automatically built and pushed to registries via GitHub Actions:

- **Docker Hub**: `your-org/your-repo-jump-server`
- **GitHub Container Registry**: `ghcr.io/your-org/your-repo-jump-server`

The image is built on:
- Pushes to `dev` branch
- Version tags (`v*`)
- Pull requests to `main` (build only, no push)

## Deploying to Koyeb

### Option 1: Use Pre-built Image from CI

1. **In Koyeb service settings:**
   - **Image**: `ghcr.io/your-org/your-repo-jump-server:dev` (or specific tag)
   - **Ports**: 
     - Port 10000 (Quote of the Day web server)

### Option 2: Build and Push Manually

```bash
docker build -f Dockerfile.jump-server -t your-registry/jump-server:latest .
docker push your-registry/jump-server:latest
```

### Option 3: Use Koyeb's Build from Git

- Point Koyeb to your repository
- Set Dockerfile path to `Dockerfile.jump-server`
- Koyeb will build and deploy automatically

## Using test_db_perf

You can run the binaries by executing commands in the container:

```bash
docker exec -it <container-name> test_db_perf
```

Or if you need to run it interactively:

```bash
# Run the performance test
test_db_perf -db "postgresql://user:pass@host:port/dbname" -duration 30s -concurrency 10

# Or with flags
test_db_perf \
  -db "$DATABASE_URL" \
  -duration 60s \
  -concurrency 20 \
  -reads=true \
  -writes=true \
  -mixed=true
```

The binary is available in `/usr/local/bin/test_db_perf` and should be in your PATH.

## Using rebuild_db

Rebuild and seed the database:

```bash
# Rebuild database (drops all tables and recreates from schema.sql)
rebuild_db -db "postgresql://user:pass@host:port/dbname"

# Or use environment variable
export DATABASE_URL="postgresql://user:pass@host:port/dbname"
rebuild_db
```

**Note**: The `rebuild_db` command will:
- Drop all existing tables
- Recreate schema from `schema.sql`
- Import seed data (categories, parts, ads, etc.)

**Warning**: This is a destructive operation! Use with caution.

The binary is available in `/usr/local/bin/rebuild_db` and should be in your PATH.

## Using gen_test_images

Generate and upload test images for all test ads (ads with `user_id = 1`) to MinIO:

```bash
# Set required environment variables
export DATABASE_URL="postgresql://user:pass@host:port/dbname"
export MINIO_API_URL="http://localhost:9000"
export MINIO_USERNAME="minioadmin"
export MINIO_PASSWORD="minioadmin"
export MINIO_BUCKET_NAME="your-bucket-name"

# Run the generator
gen_test_images
```

**What it does:**
- Generates test images for all ads with `user_id = 1`
- Creates 1-5 images per ad (based on `image_count` field)
- Generates three sizes per image: 160w, 480w, 1200w
- Uploads images to MinIO in batches of 10 concurrent uploads
- Uses WebP format for optimal compression

**Requirements:**
- Database connection (via `DATABASE_URL` environment variable)
- MinIO credentials (via `MINIO_API_URL`, `MINIO_USERNAME`, `MINIO_PASSWORD`, `MINIO_BUCKET_NAME`)

The binary is available in `/usr/local/bin/gen_test_images` and should be in your PATH.

## Using mc (MinIO Client)

The MinIO Client (`mc`) is pre-installed and available in your PATH:

```bash
# Check version
mc --version

# Configure alias for MinIO server
mc alias set myminio https://your-minio-server.com ACCESS_KEY SECRET_KEY

# List buckets
mc ls myminio

# List objects in a bucket
mc ls myminio/bucket-name

# Copy file to MinIO
mc cp local-file.txt myminio/bucket-name/

# Copy file from MinIO
mc cp myminio/bucket-name/file.txt ./

# Mirror a directory
mc mirror /local/dir myminio/bucket-name/path
```

For more information, see the [MinIO Client documentation](https://docs.min.io/docs/minio-client-quickstart-guide.html).

## Quote of the Day Web Server

The jump server includes a simple web server that serves a quote of the day page on port 10000.

### Accessing the Quote Page

Once the jump server is running, you can access the quote page at:

```
http://your-server:10000/
```

The page displays:
- A daily quote (changes once per day based on the day of the year)
- The current date
- A simple, modern web interface

### Example Usage

```bash
# Access locally (if running with port mapping)
curl http://localhost:10000/

# Or open in a browser
open http://localhost:10000/
```

The quote server starts automatically when the jump server container starts and runs in the background.

## Features

- ✅ Pre-built `test_db_perf` binary ready to use
- ✅ Pre-built `rebuild_db` binary ready to use
- ✅ Pre-built `gen_test_images` binary ready to use
- ✅ Quote of the day web server on port 10000
- ✅ MinIO Client (`mc`) pre-installed
- ✅ Non-root `jump` user with sudo access
- ✅ Command logging and auditing (`/var/log/jump-server.log`)
- ✅ Firewall configured (UFW, web server only)
- ✅ Monitoring tools installed (htop, iotop, tcpdump, strace)
- ✅ Workspace directory at `/workspace`
- ✅ Comprehensive logging via rsyslog

## Security Features

- **Firewall**: UFW configured to allow web server (port 10000) only
- **Command auditing**: All commands logged to `/var/log/jump-server.log`
- **Non-root user**: `jump` user with sudo available

## Usage

### Running Commands in the Container

You can execute commands in the running container:

```bash
# Run test_db_perf
docker exec -it <container-name> test_db_perf -db "postgresql://..." -duration 30s

# Run rebuild_db
docker exec -it <container-name> rebuild_db -db "postgresql://..."

# Access shell
docker exec -it <container-name> /bin/bash
```

### Viewing Logs

```bash
# View jump server audit log
tail -f /var/log/jump-server.log

# View system logs
journalctl -f

# View command history (logged commands)
grep "jump\|root" /var/log/jump-server.log
```

### Monitoring Tools

```bash
# System monitoring
htop          # Process monitor
iotop         # I/O monitor
netstat -tuln # Network connections
tcpdump       # Network packet capture
strace        # System call tracer
```

