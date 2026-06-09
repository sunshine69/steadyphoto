#!/bin/bash
set -e

export DATABASE_URL="postgres://steadyphoto:password@localhost:5432/steadyphoto?sslmode=disable"
# Storage directory for photos (absolute path - must exist)
export STORAGE_DIR="storage"
# Optional: For debugging verbose output
export LOG_LEVEL=debug
# API port (if different from default)
export API_PORT=8081
export API_BASE_URL="http://localhost:$API_PORT"

docker compose -f docker-compose-postgres.yml down -v ; docker compose -f docker-compose-postgres.yml up -d
sleep 3

killall server.exe
go build -o server.exe cmd/server/main.go
nohup ./server.exe > server.log 2>&1 &
sleep 3

go run cmd/migrate/main.go up

go build -o scanner.exe cmd/scanner/main.go

rm -rf storage/* ;  ./scanner.exe -u admin@steadyphoto.com -p 1qa2ws -source /mnt/doc/The\ Spit\ Lake\ Somerset -email admin@steadyphoto.com

# ./scanner.exe -source /mnt/doc/Videos/AI-Video -storage storage -email admin@steadyphoto.com
#./scanner.exe -source /mnt/doc/Diana\ Place\ 5/ -storage storage

go build -o worker.exe cmd/worker/main.go
./worker.exe > worker.log 2>&1 &
killall worker.exe

# go test ./... -v -count=1
ps -ef|grep 'ng serv' | awk '{print $2}' | while read pid; do kill $pid; done
( cd angular-app && npx ng serve & )
