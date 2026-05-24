# ============================================
# Stage 1: Build Angular Application
# ============================================
FROM node:20-alpine AS angular-builder

WORKDIR /app/angular-app

# Copy package files first for better caching
COPY angular-app/package*.json ./
RUN npm ci

# Copy source code
COPY angular-app/ .

# Build the Angular application with production configuration
# Output will go to dist/ directory
RUN npm run build -- --configuration=production

# ============================================
# Stage 2: Build Go Binary
# ============================================
FROM golang:1.22-alpine AS go-builder

WORKDIR /app/steadyphoto

# Install git (needed for some Go modules) and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy go module files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the server binary with static linking for Alpine compatibility
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/steadyphoto/server cmd/server/main.go

# ============================================
# Stage 3: Production Image
# ============================================
FROM alpine:3.19

# Install ca-certificates for HTTPS requests and create necessary directories
RUN apk add --no-cache ca-certificates && \
    mkdir -p /app/storage

WORKDIR /app

# Copy the Go binary from go-builder stage
COPY --from=go-builder /app/steadyphoto/server ./server

# Copy the built Angular application to /ui directory
COPY --from=angular-builder /app/angular-app/dist /app/ui

# Create a non-root user for security
RUN addgroup -S appgroup && \
    adduser -S appuser -G appgroup && \
    chown -R appuser:appgroup /app/storage

USER appuser

# Expose port 8080 (now serves both UI and API)
EXPOSE 8080

# Environment variables with defaults
ENV DATABASE_URL=postgres://steadyphoto:password@db:5432/steadyphoto?sslmode=disable
ENV STORAGE_DIR=/app/storage
ENV API_PORT=:8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ui/ || exit 1

# Run the server
CMD ["./server"]
