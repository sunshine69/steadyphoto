## Test It Now

```bash
# 1. Stop everything and remove volumes (clean slate)
docker compose down -v

# 2. Rebuild from scratch
docker compose build --no-cache

# 3. Start up
docker compose up -d

# 4. Watch the logs to see migrations run
docker compose logs -f app
```

You should see in the logs:
```
Waiting for database to be ready...
Database is ready!
Running database migrations...
Applying migration version 1...
Successfully applied migration 1.
Migrations completed successfully.
Starting SteadyPhoto server...
```

