# Jump Server with test_db_perf and rebuild_db

This Dockerfile creates a jump server based on Ubuntu 22.04 with SSH server configured, and includes pre-built `test_db_perf` and `rebuild_db` binaries ready to use.

## Building the Image

```bash
docker build -f Dockerfile.jump-server -t jump-server:latest .
```

## Running Locally

```bash
docker run -d \
  -p 2222:22 \
  -e SSH_PUBLIC_KEY="$(cat ~/.ssh/id_ed25519_koyeb.pub)" \
  jump-server:latest
```

Then SSH in:
```bash
ssh -i ~/.ssh/id_ed25519_koyeb -p 2222 root@localhost
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
   - **SSH Public Key**: Your public key
   - **Port**: 22 (default SSH port)

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

Once SSH'd into the server:

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

## Features

- ✅ Pre-built `test_db_perf` binary ready to use
- ✅ Pre-built `rebuild_db` binary ready to use
- ✅ MinIO Client (`mc`) pre-installed
- ✅ Key-only SSH authentication (no passwords)
- ✅ Non-root `jump` user with sudo access (recommended)
- ✅ Command logging and auditing (`/var/log/jump-server.log`)
- ✅ Firewall configured (UFW, SSH only)
- ✅ Monitoring tools installed (htop, iotop, tcpdump, strace)
- ✅ Workspace directory at `/workspace`
- ✅ Comprehensive logging via rsyslog

## Security Features

- **Key-only authentication**: No password login allowed
- **Firewall**: UFW configured to allow SSH (port 22) only
- **Command auditing**: All commands logged to `/var/log/jump-server.log`
- **Non-root user**: `jump` user with sudo (use instead of root when possible)
- **Access control**: Only `root` and `jump` users can SSH in

## Usage

### SSH Access

You can SSH as either `root` or `jump` user:

```bash
# As root
ssh -i ~/.ssh/id_ed25519_koyeb root@your-server

# As jump user (recommended)
ssh -i ~/.ssh/id_ed25519_koyeb jump@your-server
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

