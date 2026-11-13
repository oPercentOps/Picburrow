#!/bin/sh
# Builds picburrow_<version>_<arch>.deb from a built binary. Usage: packaging/build-deb.sh <binary> <amd64|arm64>
# Install on a Pi with one command: sudo apt install ./picburrow_<version>_arm64.deb
set -eu
BIN=${1:?usage: build-deb.sh <binary> <arch>}
ARCH=${2:?usage: build-deb.sh <binary> <arch>}
HERE=$(cd "$(dirname "$0")" && pwd)
VERSION=$(cat "$HERE/../VERSION")
ROOT=$(mktemp -d)
chmod 0755 "$ROOT" # mktemp makes it 0700; that would be the package's / permissions
trap 'rm -rf "$ROOT"' EXIT
install -D -m 0755 "$BIN" "$ROOT/usr/bin/picburrow"
install -D -m 0644 "$HERE/picburrow.service" "$ROOT/lib/systemd/system/picburrow.service"
install -D -m 0644 "$HERE/default" "$ROOT/etc/default/picburrow"
install -d "$ROOT/DEBIAN"
for f in postinst prerm postrm; do install -m 0755 "$HERE/$f" "$ROOT/DEBIAN/$f"; done
echo /etc/default/picburrow > "$ROOT/DEBIAN/conffiles"
cat > "$ROOT/DEBIAN/control" <<CONTROL
Package: picburrow
Version: $VERSION
Architecture: $ARCH
Maintainer: Picburrow maintainers <noreply@picburrow.invalid>
Depends: libvips-tools, libimage-exiftool-perl, adduser
Recommends: ffmpeg
Section: graphics
Priority: optional
Homepage: https://github.com/oPercentOps/RaspictureGalleryNas
Description: photo gallery for the folders you already have
 Picburrow turns the photo folders on a Raspberry Pi or small NAS into a
 phone-friendly timeline: viewer, folders, albums, search and share links.
 One small program with SQLite: no Docker, no Postgres. It only reads your
 photos; nothing is moved, renamed or changed. Runs in 1 GB of RAM.
CONTROL
OUT="picburrow_${VERSION}_${ARCH}.deb"
dpkg-deb --root-owner-group --build "$ROOT" "$OUT" >/dev/null
echo "$OUT"
