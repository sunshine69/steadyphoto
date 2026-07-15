#!/bin/bash
#set -e

set -a
. .env
set +a

docker compose -f docker-compose-postgres.yml down -v ; docker compose -f docker-compose-postgres.yml up -d
sleep 3

export CGO_ENABLED=0

go run cmd/migrate/main.go up

killall server.exe
go build -ldflags="-extldflags=-static -w -s" --tags "osusergo netgo" -o server.exe cmd/server/main.go
nohup ./server.exe > server.log 2>&1 &
sleep 3

go build -ldflags="-extldflags=-static -w -s" --tags "osusergo netgo" -o scanner.exe cmd/scanner/main.go

# Scan and upload image from local fs using api.
# Upload processor should create a thumnail jobs for the worker to process later on.
# It should also process exif data and update the field captured_date from exif data
rm -rf storage ; mkdir storage;  ./scanner.exe -u ${ADMIN_EMAIL} -p ${ADMIN_PASSWORD}  -source /mnt/doc/tmp/testimg > scanner.log 2>&1

./scanner.exe -source /mnt/doc/Videos/AI-Video -u ${ADMIN_EMAIL} -p ${ADMIN_PASSWORD}  >> scanner.log  2>&1
./scanner.exe -source /mnt/doc/Diana\ Place\ 5/ -u ${ADMIN_EMAIL} -p ${ADMIN_PASSWORD}  >> scanner.log  2>&1

# Generate thumbnail
go build -ldflags="-extldflags=-static -w -s" --tags "osusergo netgo" -o worker.exe cmd/worker/main.go
./worker.exe > worker.log 2>&1

# Run the exif manually - we should not need to run it as it should be in the upload
# processor. but we can always re-run tomanually update if required
go build -ldflags="-extldflags=-static -w -s" --tags "osusergo netgo" -o exif-update cmd/exif/main.go
#./exif-update

# This comes after exif-update but this function should be at processing time as well
# go run cmd/update-capture-date/main.go

# Front end
ps -ef|grep 'ng serv' | awk '{print $2}' | while read pid; do kill $pid || true; done
( cd angular-app && npx ng serve & )

