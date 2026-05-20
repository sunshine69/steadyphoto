#!/bin/sh

export DATABASE_URL="postgres://steadyphoto:password@localhost:5432/steadyphoto?sslmode=disable"
export STORAGE_DIR="storage"
export LOG_LEVEL=debug
export MAX_UPLOAD_SIZE=5242880
export API_PORT=:8081
killall server.exe

go build -o server.exe cmd/server/main.go
nohup ./server.exe > server.log 2>&1 &

