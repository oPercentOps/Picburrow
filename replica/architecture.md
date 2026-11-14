# Architecture: Picburrow (a rebuild of Immich / PhotoPrism's browse + view features)

Constraint that drives every choice: **Raspberry Pi 3/4/5, 1 GB RAM, 64-bit Raspberry Pi OS (arm64)**, self-hosted, published as an open-source product. The skill's default stack (Next.js, Supabase, Vercel) is cloud-hosted and multi-GB, so it is replaced wholesale.

## Stack

| layer | choice | why |
| --- | --- | --- |
| server | **Go**, one static binary, `CGO_ENABLED=0` | ~20-40 MB RSS, cross-compiles to arm64 from any laptop/CI, no runtime to install |
| web UI | **Svelte (SvelteKit, adapter-static SPA)** + TypeScript, embedded in the binary with `go:embed` | small bundles for phones; Node is needed only at build time, never on the Pi. Swap for: vanilla JS |
| styling | plain CSS with tokens from replica-design | no build plugin to maintain |
| database | **SQLite** (WAL) via `modernc.org/sqlite` (pure Go) | no server process, one file; Postgres alone needs ≥ 2 GB per Immich docs. `cache_size` capped at 16 MB |
| image pipeline | **`vipsthumbnail`** (libvips + libheif, Debian package) spawned per file | shrink-on-load keeps peak RAM low; a crash or leak dies with the child process, not the server; HEIC/AVIF/WebP for free |
| metadata | **`exiftool -stay_open`**, one long-lived process | reads EXIF/XMP/QuickTime/HEIC dates, GPS and the live-photo pairing ID in one tool |
| video poster | `ffmpeg -frames:v 1` (optional dependency) | first frame only, no transcoding |
| auth | local users, bcrypt (`golang.org/x/crypto`), session cookie (HttpOnly, SameSite=Lax, hashed in DB) | LAN product, no third-party auth |
| payments | none | free open-source product |
| email | none | nothing to send in this slice |
| jobs | in-process worker goroutine + `time.Ticker` | scan, thumbnails and purge; no queue infra. Concurrency 1 by default (flag to raise on 4-8 GB Pis) |
| hosting / packaging | `.deb` for arm64 + amd64 with a systemd unit; optional Docker image | `apt install ./raspicture.deb` pulls in libvips-tools, libimage-exiftool-perl, ffmpeg |
| config | flags / env (`--data-dir`, `--listen`, `--workers`) | libraries and users live in the DB, set from the UI |

Memory budget (1 GB Pi, ~750 MB usable after the OS): server 40 MB + SQLite cache 16 MB + exiftool 30 MB + one vipsthumbnail child at 50-250 MB (a 48 MP HEIC is the worst case) ≈ **≤ 340 MB peak**, 100 MB idle.

## Schema

Tables: 9 (`users, sessions, libraries, assets, albums, album_assets, shared_links, tags, asset_tags`). See `server/schema.sql`, which is checked to load, enforce its checks and cascade on delete.

Access rules: **data-layer checks.** SQLite has no RLS. Every query that touches user data goes through a small repository function that takes the caller's `user_id` and joins `libraries.owner_id` / `albums.owner_id`. Public links resolve `token → album_id` and may read only `album_assets` of that album.

Decisions:
- **No thumbnails table.** Thumbnail files sit at `<data-dir>/thumbs/<id % 256>/<id>-{s,l}.webp`; `assets.thumbs_at` says whether they exist. `assets_todo` partial index = the job queue.
- **`taken_local` as wall-clock text.** EXIF dates usually have no offset; the timeline groups by the day the photo was taken *where it was taken*, which is what users expect. `tz_offset_min` is kept when known.
- **`quick_hash`** (size + first/last 64 KiB) instead of a full hash: hashing 400 GB on a Pi SD/USB bus takes hours. On rescan, a new path whose quick_hash matches a just-missing asset is **relinked** (keeps favourites, albums, tags), which fixes the documented Immich weakness of losing metadata when a file moves. shortcut: collisions are possible for identical-size files with identical heads/tails; upgrade to a full hash on match if reports come in.
- **Favourite/archive live on `assets`**, because a library has one owner (same as Immich). shortcut: per-user favourites need a `user_assets` table, so add it if partner sharing ships.
- **Read-only mounts are the default.** Edits go to the DB only. XMP write-back (a could) runs only when the mount is writable, and otherwise the UI says why instead of failing silently like Immich.

## API

JSON under `/api`, cookie auth. Image routes send `ETag` + `Cache-Control: private, max-age=31536000, immutable` (URL contains `thumbs_at`).

| method path | does | who | input | output | flow |
| --- | --- | --- | --- | --- | --- |
| POST /api/setup | create first admin + first library | anyone, only while `users` is empty | username, password, import path | session cookie | F07 |
| POST /api/login | sign in | anyone (rate-limited) | username, password | session cookie | — |
| POST /api/logout | end session | user | — | 204 | — |
| GET /api/me | current user | user | — | user | — |
| GET /api/timeline/buckets | months with counts, for scrubber + virtual scroll heights | user | archived?, library? | `[{month, count}]` | F01 F02 |
| GET /api/timeline | assets of one month | user | month=YYYY-MM | `[{id, w, h, type, thumbhash, taken, fav, live}]` | F01 F02 |
| GET /api/assets/:id | full metadata (info panel) | owner | — | asset | F03 |
| GET /api/assets/:id/thumb/:size | small (grid) or large (viewer) WebP; generates on demand, jumping the queue | owner | size=s\|l | image | F01 F03 |
| GET /api/assets/:id/original | original file with HTTP Range (video seek) | owner | — | file | F03 |
| PATCH /api/assets | bulk favourite / archive / rating / description | owner | ids[], fields | 204 | F05 F08 |
| GET /api/folders | subfolders + assets of a directory | owner | library, path | `{dirs[], assets[]}` | F04 |
| GET /api/search | filters: from, to, path, filename, camera, type, fav, archived, rating, place | user | query string | assets page (cursor) | F02 |
| GET /api/albums | list | user | — | albums | F05 |
| POST /api/albums | create | user | name, asset ids? | album | F05 |
| GET /api/albums/:id | album + assets | owner | — | album | F05 |
| PATCH /api/albums/:id | rename, cover | owner | name, cover | album | F05 |
| DELETE /api/albums/:id | delete (files untouched) | owner | — | 204 | — |
| POST /api/albums/:id/assets | add | owner | ids[] | 204 | F05 |
| DELETE /api/albums/:id/assets | remove | owner | ids[] | 204 | F05 |
| POST /api/albums/:id/links | create share link | owner | password?, expires_at?, allow_download | link with token | F06 |
| DELETE /api/links/:id | revoke | owner | — | 204 | F06 |
| GET /api/s/:token | public album | anyone with token (+ unlock cookie if password) | — | album, assets | F06 |
| POST /api/s/:token/unlock | password check | anyone (rate-limited) | password | link cookie | F06 |
| GET /api/s/:token/assets/:id/{thumb/:size,original} | public media, only assets in that album; original only if allow_download | link holder | — | image/file | F06 |
| GET /api/trash | missing-on-disk assets | owner | — | assets | F07 |
| GET/POST/PATCH/DELETE /api/libraries[/:id] | manage import paths + exclusion globs; validates each path is a readable dir | admin | paths, globs | library | F07 |
| POST /api/libraries/:id/scan | queue rescan | admin | — | 202 | F07 |
| GET /api/jobs | scan + thumbnail progress, recent errors | admin | — | `{scan, thumbs: {done, todo}, errors[]}` | F07 |
| GET/POST/DELETE /api/users[/:id] | manage users | admin | username, password | user | — |
| GET /* | the SPA (embedded) | anyone | — | HTML/JS | all |

Routes: 30 rows (36 endpoints counting each method).

Webhooks in: none. Webhooks out: none.

Jobs:
- **scan**, on start, nightly 03:00 local, on demand, and every 15 minutes when a folder's modified time changed since the last scan (one stat per folder, no folder read): walk import paths, skip exclusion globs, upsert by `(library_id, path)` when size/mtime changed, read metadata via exiftool, read the XMP sidecar next to each file (keywords, description, stars; re-read alone when only the sidecar changed), pair live photos, mark vanished files `missing_at`, relink moved files by quick_hash.
- **thumbs**, continuous at low priority (`nice 10`, `ionice idle`): take from `assets_todo`, newest first so the top of the timeline fills first; write small (~360 px) + large (~1440 px) WebP and the thumbhash.
- **purge**, daily: delete DB rows (never files) and thumbnails for assets missing more than 30 days.
- **sessions cleanup**, daily: delete expired sessions.

## The parts that bite

- **Time zones:** EXIF has no offset on most phone photos, so sort and group by `taken_local` and never convert it. Files with no EXIF date fall back to file mtime (flag it in the info panel). Videos store UTC in QuickTime metadata, so convert those to local using the device offset when present, else the server's TZ.
- **Memory:** one image decode at a time; `VIPS_CONCURRENCY=1`; cap the SQLite cache; stream originals with `http.ServeContent` (never load a file into memory); paginate every list. A worst-case 48 MP HEIC must be tested on a real 1 GB Pi 3.
- **I/O on SD/USB:** the DB and thumbnails belong on the fastest disk (SSD/USB 3), not the SD card if avoidable. Thumbnails add 10-20% of library size (Immich figure).
- **Idempotency:** scan is an upsert keyed on path; a crash mid-scan just resumes. Thumbnail writes go to a temp file and are renamed into place, so a half-written WebP is never served.
- **Races:** a single writer goroutine for scan state; SQLite `busy_timeout=5000`; an on-demand thumbnail request and the worker on the same asset are de-duplicated with a per-id in-flight map.
- **Rate limits:** login and share-link unlock limited per IP (e.g. 10/min) to stop LAN brute force.
- **File size limits:** none on read; originals are streamed. Skip files over a configurable decode limit (e.g. 200 MP) with a `thumb_error`.
- **Search:** `LIKE` on path/filename over 100k rows is a few ms in SQLite. shortcut: upgrade to FTS5 if it isn't.
- **Security:** never serve a path from the request, only by asset id → DB path, and check the path is still under an import path (symlink escape). Share tokens are ≥ 128 bits. Thumbnails of other users' assets are reachable only through ownership checks.
- **Offline / realtime:** none needed; the job-progress page polls every 2 s.
- **Network mounts:** no inotify (Immich says it fails on network drives), so rely on scheduled + manual rescans.
- **Multi-tenancy / GDPR:** single household; deleting a user cascades their libraries' DB rows and thumbnails, never the photo files.

## Build order

1. **Vertical slice** (proves the Pi can do it): S02 setup → scan one folder → S03 timeline → S04 viewer.
   Tables: users, sessions, libraries, assets. Routes: setup, login, timeline/buckets, timeline, thumb, original, assets/:id. Jobs: scan, thumbs.
   **Exit test:** 10k phone photos on a 1 GB Pi 3, RSS < 350 MB throughout, timeline usable before thumbnails finish.
2. **Must-haves:** S01 login, S05 info panel, S06 folders, S09 favourites, video playback, HEIC, scrubber, rescan + missing detection, S13 libraries admin, S14 jobs page, mobile layout.
   Routes: folders, PATCH assets, libraries, scan, jobs.
3. **Should-haves:** S07/S08 albums, S10 archive, S11 search, multi-select, S12 share links, live photos, quick_hash relink, PWA manifest, thumbhash placeholders, multiple users.
   Tables: albums, album_assets, shared_links.
4. **Could-haves:** S15 trash UI, tags (tables: tags, asset_tags), XMP read then write-back, ratings, offline reverse geocoding, exact duplicates, map, memories, rotate, RAW embedded previews, locked folder.
5. **Fixes from replica-entrepreneur:** to be added once it has run.
