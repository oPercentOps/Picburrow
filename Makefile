# make        -> web UI + server binary for this machine
# make pi     -> arm64 binary for 64-bit Raspberry Pi OS
# make test   -> server tests + layout self-check
.PHONY: all web server pi test

all: web server

web:
	cd web && npm ci --no-audit --no-fund && npm run build
	touch server/dist/.keep

server:
	cd server && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../raspicture .

pi: web
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ../raspicture-arm64 .

test:
	cd server && go test ./...
	cd web && node --experimental-strip-types --no-warnings src/lib/justify.test.ts && npm run check
