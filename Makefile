# make        -> web UI + server binary for this machine
# make pi     -> arm64 binary for 64-bit Raspberry Pi OS
# make deb-pi -> homeroll_<version>_arm64.deb: on the Pi, sudo apt install ./homeroll_<version>_arm64.deb
# make e2e    -> Playwright end-to-end against the real binary (builds first)
# make test   -> server tests + layout self-check
.PHONY: all web server pi test e2e deb deb-pi

all: web server

web:
	cd web && npm ci --no-audit --no-fund && npm run build
	touch server/dist/.keep

server:
	cd server && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o ../homeroll .

pi: web
	cd server && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ../homeroll-arm64 .

test:
	cd server && go test ./...
	cd web && node --experimental-strip-types --no-warnings src/lib/justify.test.ts && npm run check

e2e: all
	cd e2e && npm ci --no-audit --no-fund && npx playwright test

deb: all
	packaging/build-deb.sh homeroll amd64

deb-pi: pi
	packaging/build-deb.sh homeroll-arm64 arm64
