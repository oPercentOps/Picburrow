# Picburrow

The photo folders you already have, as a timeline on your phone. Picburrow is one small program for a Raspberry Pi NAS, with no Docker, no Postgres and no Redis:
a timeline by day, a fast fullscreen viewer, and photo details. It runs on a Pi 3, 4 or 5 with **1 GB RAM**
and 64-bit Raspberry Pi OS, and only reads your photo folders, never changes them.

Status: timeline with date scrubber, viewer, live photos, star ratings, folders, favourites, archive, trash, albums, multi-select, search, share links, family accounts, library settings; installable on a phone's home screen. Feature parity 87/100 (`replica/parity.md`). See `replica/build-log.md` and `replica/features.csv`.

To hear about new versions: **Watch → Custom → Releases** on GitHub. The project page is `docs/index.html`.

## Install on a Raspberry Pi (64-bit Raspberry Pi OS)

```bash
sudo apt install ./picburrow_0.1.1_arm64.deb
```

That one command pulls in libvips (with the HEVC decoder iPhone HEIC photos need), exiftool and ffmpeg, creates a `picburrow` system user and starts the service.
Open `http://<pi-address>:8080`, create your account and enter the folder your phone backups land in.
Build the package on any machine with `make deb-pi`.

- **Settings**: `/etc/default/picburrow` (`PICBURROW_DATA`, `PICBURROW_LISTEN`), then `sudo systemctl restart picburrow`.
- **Put the data on an SSD or USB drive, not the SD card.** Picburrow warns on the Indexing page when it isn't.
- **Photo folder permissions**: the `picburrow` user only needs to read your photos, e.g. `sudo usermod -aG <folder group> picburrow`.
- **Upgrades**: install the newer `.deb` the same way. Before changing its database, Picburrow saves a copy next to it
  (`picburrow.db.v<N>-<date>.bak`, newest 3 kept). To go back: `sudo systemctl stop picburrow`, copy the backup over
  `picburrow.db`, reinstall the older package, start again.
- **Uninstalling** never deletes `/var/lib/picburrow` (albums, favourites, previews) and never touches your photos.

## Run it anywhere else

```bash
sudo apt install libvips-tools libheif-plugin-libde265 libimage-exiftool-perl ffmpeg
make            # builds the web UI and ./picburrow (needs Go 1.26+ and Node 22)
./picburrow --data-dir ./data --listen :8080
```

On Debian 12 or older, leave out `libheif-plugin-libde265`: the decoder is built into libheif there.

| flag | default | |
| --- | --- | --- |
| `--data-dir` | `/var/lib/picburrow` | database and thumbnails (put it on the SSD/USB disk, not the SD card if you can) |
| `--listen` | `:8080` | address to serve on |
| `--workers` | `1` | parallel thumbnail decodes; keep 1 on a 1 GB Pi |

## Develop

```bash
make test        # Go tests + layout self-check + svelte-check
make e2e         # Playwright: starts the real binary on a fresh data dir with a generated photo library
cd web && npm run dev   # UI on :5173, proxies /api to a server on :8080
```

The e2e suite (`e2e/*.spec.ts`) follows the flows in `replica/test-plan.md` on desktop (1440 px, light) and phone
(390 px, dark, touch). Every test fails on console errors, 5xx responses or WCAG 2.1 AA violations (axe).
Screenshots land in `replica/clone-screens/`.

Planning docs: `replica/` (recon, architecture, design tokens, build log).

## Licence

Picburrow is free software: you can use, study, share and change it under the
[GNU Affero General Public License v3.0 or later](LICENSE). If you run a changed version for other people,
including over the network, you must offer them your changed source; the "Source code" link in the app
(`web/src/lib/source.ts`) is where it points. The libraries built into each release are listed with their licences
in `/usr/share/doc/picburrow/THIRD_PARTY_NOTICES` (made by `packaging/notices.sh`).

## Privacy

Picburrow makes no outbound connections: no telemetry, no update checks, no accounts with us. Photos, previews,
accounts and sessions stay in your photo folders and `/var/lib/picburrow`. The only cookie is the sign-in session
(plus one per unlocked share link), which the app needs to work. Whoever runs a Picburrow server is responsible for the
people they give accounts or share links to.
