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

# Run migrations
echo "Running database migrations..."
if [ -f /app/migrate ]; then
    /app/migrate up
    echo "Migrations completed successfully."
else
    echo "WARNING: Migrate binary not found at /app/migrate, skipping migrations"
fi

# Start the server
echo "Starting SteadyPhoto server..."
exec /app/server
