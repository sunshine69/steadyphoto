# ============================================
# Stage 1: Build Angular Application
# ============================================
FROM node:24-alpine AS angular-builder

WORKDIR /app/angular-app

# Copy package files first for better caching
COPY angular-app/package*.json ./
RUN npm install --legacy-peer-deps  # try ci later

# Copy source code
COPY angular-app/ .

# Build the Angular application with production configuration
# Output will go to dist/ directory
RUN npm run build -- --configuration=production


# ============================================
# Stage 2: Build Go Binaries (Server + Migrate)
# ============================================
FROM golang:1.26.3-alpine3.23 AS go-builder

WORKDIR /app/steadyphoto

# Install git (needed for some Go modules), ca-certificates, and postgresql-client (for pg_isready in entrypoint)
RUN apk add --no-cache git ca-certificates 

# Copy go module files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the server binary with static linking for Alpine compatibility
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-X main.version=v1.0.0-$(git rev-parse --short HEAD) -X main.buildTime=$(date '+%Y%m%d_%H%M%S') -extldflags=-static -w -s" --tags "osusergo netgo" -o /app/steadyphoto/server cmd/server/main.go

# Build the migrate binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/steadyphoto/migrate cmd/migrate/main.go

# Stage 3: Production Image
# ============================================
FROM alpine:3.19

# Install ca-certificates, postgresql-client for health checks, and su-exec to drop privileges
RUN apk add --no-cache ca-certificates su-exec postgresql-client && \
    mkdir -p /app/storage

WORKDIR /app

# Copy the Go binaries from go-builder stage
COPY --from=go-builder /app/steadyphoto/server ./server
COPY --from=go-builder /app/steadyphoto/migrate ./migrate
COPY --from=go-builder //app/steadyphoto/migrations ./migrations

# Copy the built Angular application to /ui directory
COPY --from=angular-builder /app/angular-app/dist /app/ui

# Add the entrypoint script and make it executable
COPY entrypoint.sh .
RUN chmod +x entrypoint.sh

# Create a non-root user for security (the main server runs as this user)
RUN addgroup -S appgroup && \
    adduser -S appuser -G appgroup && \
    chown -R appuser:appgroup /app/storage

ENV DATABASE_URL=postgres://steadyphoto:password@db:5432/steadyphoto?sslmode=disable
ENV STORAGE_DIR=/app/storage
ENV API_PORT=8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ui/ || exit 1

# Run the entrypoint script (runs as root to handle init tasks like chown, then drops privileges for the server)
ENTRYPOINT ["./entrypoint.sh"]
