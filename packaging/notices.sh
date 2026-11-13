#!/bin/sh
# Prints the licences of every library built into Picburrow: the Go modules linked into the binary
# (plus the Go standard library) and the web UI's bundled packages (written by `npm run build`).
# Usage: packaging/notices.sh <amd64|arm64> > THIRD_PARTY_NOTICES (plain text: the licence files carry their own markdown)
set -eu
ARCH=${1:?usage: notices.sh <arch>}
TOP=$(cd "$(dirname "$0")/.." && pwd)
WEB="$TOP/web/.svelte-kit/output/client/.vite/license.md"
[ -f "$WEB" ] || { echo "notices.sh: $WEB missing, run the web build first" >&2; exit 1; }

echo "Third-party notices"
echo
echo "Picburrow is licensed under the GNU AGPL v3 or later (see copyright). It includes the libraries below."
echo
cd "$TOP/server"
GOOS=linux GOARCH=$ARCH go list -deps -f '{{with .Module}}{{if not .Main}}{{.Path}}@{{.Version}} {{.Dir}}{{end}}{{end}}' . | sort -u |
  { cat; echo "Go-standard-library@$(go env GOVERSION) $(go env GOROOT)"; } |
  while read -r mod dir; do
    echo "================================================================"
    echo "$mod"
    for f in "$dir"/LICENSE* "$dir"/PATENTS; do
      [ -f "$f" ] || continue
      echo "---- $(basename "$f")"; cat "$f"
    done
    echo
  done
echo "================================================================"
echo "Web UI"
echo
sed '1,/^## /{/^## /!d}' "$WEB" # drop Vite's own heading, keep each package's section
