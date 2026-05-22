#!/bin/bash

export DATABASE_URL="postgres://steadyphoto:password@localhost:5432/steadyphoto?sslmode=disable"
# Storage directory for photos (absolute path - must exist)
export STORAGE_DIR="storage"
# Optional: For debugging verbose output
export LOG_LEVEL=debug
# API port (if different from default)
export API_PORT=:8081

docker compose down -v ; docker compose up -d
sleep 5
go run cmd/migrate/main.go up
go build -o scanner.exe cmd/scanner/main.go
rm -rf storage/* ;  ./scanner.exe -source /mnt/doc/The\ Spit\ Lake\ Somerset -storage storage -email admin@steadyphoto.com
./scanner.exe -source /mnt/doc/Videos/AI-Video -storage storage -email admin@steadyphoto.com
#./scanner.exe -source /mnt/doc/Diana\ Place\ 5/ -storage storage
go build -o worker.exe cmd/worker/main.go
./worker.exe > worker.log 2>&1 &
killall server.exe
go build -o server.exe cmd/server/main.go
nohup ./server.exe > server.log 2>&1 &
sleep 20
killall worker.exe
# go test ./... -v -count=1
ps -ef|grep 'ng serv' | awk '{print $2}' | while read pid; do kill $pid; done
( cd angular-app && npx ng serve & )
