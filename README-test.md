# SteadyPhoto Testing Guide

## Prerequisites
- Go 1.26+ installed
- Docker and Docker Compose available
- PostgreSQL client tools (optional, for manual verification)

## Environment Variables

Before running tests or the application, set up environment variables:

```bash
# Database connection string
export DATABASE_URL="postgres://steadyphoto:password@localhost:5432/steadyphoto?sslmode=disable"

# Storage directory for photos (absolute path - must exist)
export STORAGE_DIR="/path/to/your/photo/storage"

# Migrations directory (optional, defaults to ./migrations)
export MIGRATIONS_DIR="./migrations"

# Optional: For debugging verbose output
export LOG_LEVEL=debug

# API port (if different from default)
export API_PORT=8081
```

## Docker Compose Commands

### Start Database Service
```bash
docker compose up -d
```

This starts the PostgreSQL database with pgvector support in detached mode.

### Stop Database Service (Keep Volumes)
```bash
docker compose down
```
**Note:** This stops containers but preserves all data in volumes (`postgres_data`). Use this for normal shutdown.

### Stop and Remove Volumes (Clean Slate)
```bash
docker compose down -v
```
**Warning:** The `-v` flag removes **all named volumes**, including `postgres_data`. Use when you want a completely fresh database with no historical data.

### Check Service Status
```bash
docker compose ps
```

### View Logs
```bash
# All services
docker compose logs

# Specific service (e.g., db)
docker compose logs db
```

## Database Migration Files

The project includes SQL migration files in the `migrations/` directory that create and update the database schema:

### 0001_init_schema.up.sql
Initial schema creation including:
- **Photos table**: Stores photo metadata (path, filename, hash, dimensions, EXIF data)
- **Faces table**: Stores face detection results with bounding boxes and vector embeddings  
- **Albums table**: Album definitions
- **Album_photos junction table**: Links photos to albums

### 0002_add_jobs_table.up.sql
Adds the jobs processing table:
- **Jobs table**: Tracks background processing tasks (thumbnails, AI analysis)

## Applying Database Migrations

The project includes a Go CLI migration tool that automatically tracks and applies migrations in order.

### Using the Migration CLI Tool (Recommended)

```bash
# Apply all pending migrations
go run cmd/migrate/main.go up
```

This will:
1. Check which migrations have already been applied (via `schema_migrations` table)
2. Execute any pending migrations in numerical order
3. Record each successful migration version

### Manual Migration with psql

If you prefer manual control, apply migrations sequentially:

```bash
psql -h localhost -U steadyphoto -d steadyphto -f migrations/0001_init_schema.up.sql
psql -h localhost -U steadyphoto -d steadyphto -f migrations/0002_add_jobs_table.up.sql
```

### Using Docker Exec

```bash
docker cp migrations db:/tmp/migrations
docker exec -it db psql -U steadyphoto -d steadyphto -f /tmp/migrations/0001_init_schema.up.sql
docker exec -it db psql -U steadyphoto -d steadyphto -f /tmp/migrations/0002_add_jobs_table.up.sql
```

### Verify Migration Success

After running migrations, verify the tables were created:

```bash
# Connect to database and list tables
psql -h localhost -U steadyphoto -d steadyphto

# In psql, run:
\dt
-- Should show: photos, faces, albums, album_photos, jobs, schema_migrations

# Check table structure
\d photos
\d faces
\d albums
\d jobs
```

## Running Tests

### Run All Unit Tests
```bash
go test ./... -v
```

The `-v` flag provides verbose output showing each test as it runs.

### Run Tests with Coverage
```bash
go test ./... -coverprofile=coverage.out -v
```

Then view coverage report:
```bash
go tool cover -html=coverage.out
```

### Run Specific Package Tests
```bash
# Example: Test only the API package
go test ./internal/api/... -v

# Example: Test only the domain package  
go test ./internal/domain/... -v
```

### Run Specific Test
```bash
# Run a specific test function
go test ./internal/api/... -run TestHandler_GetPhoto -v

# Run tests matching a pattern
go test ./internal/... -run "Test.*Photo" -v
```

### Race Detection (Important for concurrent code)
```bash
go test ./... -race -v
```

## Testing Workflow

1. **Create Storage Directory** (if it doesn't exist):
   ```bash
   mkdir -p "$STORAGE_DIR"
   ```

2. **Start Database:**
   ```bash
   docker compose up -d
   ```

3. **Set Environment Variables:**
   ```bash
   export DATABASE_URL="postgres://steadyphoto:password@localhost:5432/steadyphoto?sslmode=disable"
   export STORAGE_DIR="/path/to/your/photo/storage"
   ```

4. **Apply Database Migrations (using CLI tool):**
   ```bash
   go run cmd/migrate/main.go up
   ```

5. **Run Tests:**
   ```bash
   go test ./... -v
   ```

6. **Stop Database (when done):**
   ```bash
   docker compose down  # Keeps data
   # OR  
   docker compose down -v  # Removes all volumes and data
   ```

## Test Database Setup

The tests expect a PostgreSQL database running at `localhost:5432` with the following configuration:
- Username: `steadyphoto`
- Password: `password` 
- Database: `steadyphto`

If you need to manually connect to verify:
```bash
psql -h localhost -U steadyphoto -d steadyphto
```

## Frontend Testing (web-next)

Navigate to the Next.js application directory:
```bash
cd web-next
npm test
```

Run the development server for manual testing:
```bash
cd web-next  
npm run dev
# Visit http://localhost:3000
```

## Troubleshooting

### Tests Fail with Connection Error
- Ensure Docker is running
- Verify database container is up: `docker compose ps`  
- Check if port 5432 is available (not used by another PostgreSQL instance)

### Storage Directory Not Found
If tests fail due to missing storage directory:
```bash
# Create the directory
mkdir -p "$STORAGE_DIR"

# Verify it exists  
ls -la "$STORAGE_DIR"
```

### Migration Errors with CLI Tool
Ensure `DATABASE_URL` is set and migrations directory exists:
```bash
export DATABASE_URL="postgres://steadyphoto:password@localhost:5432/steadyphoto?sslmode=disable"
go run cmd/migrate/main.go up
```

If using a custom migrations directory:
```bash
export MIGRATIONS_DIR="/path/to/custom/migrations"  
go run cmd/migrate/main.go up
```

### Permission Issues with Volumes
```bash
# Fix ownership of docker volumes
sudo chown -R $(whoami) ~/.local/share/docker/volumes/
```

### Clear Test State Between Runs
If tests are flaky due to shared state:
```bash
# Remove volumes and restart fresh  
docker compose down -v
docker compose up -d
go run cmd/migrate/main.go up
go test ./... -v
```

## Quick Reference

| Command | Purpose |
|---------|---------|
| `export STORAGE_DIR="/path"` | Set storage directory |
| `mkdir -p "$STORAGE_DIR"` | Create storage directory |  
| `docker compose up -d` | Start services |
| `docker compose down` | Stop (keep data) |
| `docker compose down -v` | Stop and remove all volumes |
| `go run cmd/migrate/main.go up` | Apply pending migrations |
| `psql ... -f migrations/*.sql` | Manual migration application |  
| `go test ./... -v` | Run all tests verbosely |
| `go test ./... -race` | Run with race detector |
