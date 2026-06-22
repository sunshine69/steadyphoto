#!/bin/bash -e

set -a

. .env

killall server.exe || true

go build -o server.exe cmd/server/main.go
nohup ./server.exe > server.log 2>&1 &

