# RaspictureGalleryNas

An alternative to simple file galleries for browsing the phone photos you back up to a Raspberry Pi NAS:
a timeline by day, a fast fullscreen viewer, and photo details. It runs on a Pi 3, 4 or 5 with **1 GB RAM**
and 64-bit Raspberry Pi OS, and only reads your photo folders, never changes them.

Status: timeline with date scrubber, viewer, folders, favourites, archive, albums, multi-select, search, share links, library settings; installable on a phone's home screen. Feature parity 80/100 (`replica/parity.md`). See `replica/build-log.md` and `replica/features.csv`.

## Run it

```bash
sudo apt install libvips-tools libimage-exiftool-perl ffmpeg
make            # builds the web UI and ./raspicture (needs Go 1.26+ and Node 22)
./raspicture --data-dir ./data --listen :8080
```

Open http://<pi-address>:8080, create the admin account and enter the folder your phone backups land in.

For a Pi, build on any machine with `make pi` and copy `raspicture-arm64` over: it's a single static binary.

| flag | default | |
| --- | --- | --- |
| `--data-dir` | `/var/lib/raspicture` | database and thumbnails (put it on the SSD/USB disk, not the SD card if you can) |
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
