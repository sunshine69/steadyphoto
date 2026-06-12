#!/bin/sh
set -e

echo "Waiting for database to be ready..."
MAX_RETRIES=30
RETRY_COUNT=0

while ! pg_isready -h ${DATABASE_HOST:-db} -p ${DATABASE_PORT:-5432} > /dev/null 2>&1; do
    RETRY_COUNT=$((RETRY_COUNT + 1))
    if [ $RETRY_COUNT -ge $MAX_RETRIES ]; then
        echo "ERROR: Database not ready after ${MAX_RETRIES} retries"
        exit 1
    fi
    echo "Still waiting for database... (attempt $RETRY_COUNT/$MAX_RETRIES)"
    sleep 2
done

echo "Database is ready!"

# Ensure /app/storage has correct ownership for appuser:appgroup.
# This handles both cases:
#   - Named volumes (Docker creates them owned by root)
#   - Bind mounts from host that may have wrong ownership
if [ -d /app/storage ]; then
    chown -R appuser:appgroup /app/storage 2>/dev/null || true
fi

# Run migrations as the non-root user for security
echo "Running database migrations..."
if [ -f /app/migrate ]; then
    su-exec appuser:appgroup /app/migrate up 2>/dev/null || true
    echo "Migrations completed successfully."
else
    echo "WARNING: Migrate binary not found at /app/migrate, skipping migrations"
fi

# Start the server as the non-root user (drop privileges)
echo "Starting SteadyPhoto server..."
exec su-exec appuser:appgroup /app/server
