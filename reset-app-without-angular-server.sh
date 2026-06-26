#!/bin/bash -x
# set -e

set -a
. .env
set +a

docker compose -f docker-compose-postgres.yml down -v ; docker compose -f docker-compose-postgres.yml up -d
sleep 3

go run cmd/migrate/main.go up

killall server.exe
go build -o server.exe cmd/server/main.go
nohup ./server.exe > server.log 2>&1 &
# sleep 3


go build -o scanner.exe cmd/scanner/main.go

rm -rf storage/* ;  ./scanner.exe -u ${ADMIN_EMAIL} -p ${ADMIN_PASSWORD}  -source /mnt/doc/tmp/testimg

# ./scanner.exe -source /mnt/doc/Videos/AI-Video -storage storage -email admin@steadyphoto.com
#./scanner.exe -source /mnt/doc/Diana\ Place\ 5/ -storage storage

go build -o worker.exe cmd/worker/main.go
./worker.exe > worker.log 2>&1 &
killall worker.exe

# go test ./... -v -count=1
#ps -ef|grep 'ng serv' | awk '{print $2}' | while read pid; do kill $pid; done
#cd angular-app && npx ng serve &
cd angular-app && npx ng build
cd ..

