# Recon map: Immich / PhotoPrism (self-hosted web)

Scope: browse + view photos already on the NAS (no upload). Library scan, timeline, folders, viewer, favourites, albums.
For: an open-source product for Raspberry Pi / NAS owners. Hard constraints: runs on Pi 3, 4 or 5 with **1 GB RAM**, **64-bit OS only (arm64)**.
Date: 2026-11-12

## Sources

| # | source | URL | notes |
| --- | --- | --- | --- |
| 1 | Immich docs home | https://docs.immich.app/ | read; feature list in sidebar |
| 2 | Immich requirements | https://docs.immich.app/install/requirements | read; **RAM min 6 GB, rec. 8 GB; 4 GB only with ML disabled; Postgres alone ≥ 2 GB; 64-bit; Docker required; thumbnails +10-20% disk** |
| 3 | Immich external libraries | https://docs.immich.app/features/libraries | read; import paths, exclusion globs, nightly scan, experimental inotify watch, deleted→trash, moved file = new asset |
| 4 | Immich feature docs (markdown in repo) | https://github.com/immich-app/immich/tree/main/docs/docs/features | read: folder-view, searching, sharing, supported-formats, xmp-sidecars, tags, partner-sharing, locked-folder, duplicates-utility, reverse-geocoding |
| 5 | Immich admin docs + FAQ | https://github.com/immich-app/immich/tree/main/docs/docs | read: system-settings (3 thumbnails: thumbhash, small webp, large preview), jobs-workers, FAQ "slow on Raspberry Pi" |
| 6 | PhotoPrism README | https://github.com/photoprism/photoprism | read: feature overview, PWA, live photos on hover, WebDAV, 64-bit ARM images, maps gated to paid members |
| 7 | PhotoPrism getting started / Pi / user guide | https://docs.photoprism.app/ | **not read**; RAM numbers for PhotoPrism unverified |

Read as docs only: sparse git checkouts of `README.md` and `docs/**/*.md(x)`; no source files were checked out or read.

## Core loop

Open the gallery on your phone or laptop, scroll back through years of phone photos already backed up to the Pi, tap one, and swipe through it fullscreen.

## Why this product exists (the gap)

Immich needs ≥ 4-6 GB RAM, Postgres, Docker and an ML container (source 2); its own FAQ says it "can be too heavy to run on a Raspberry Pi" (source 5). PhotoPrism ships 64-bit ARM images (source 6) but its RAM needs are unverified, and maps are paid. Neither targets a 1 GB Pi. "Simple file gallery" tools fit but only show folders.

Two documented Immich pain points to beat (source 4): editing metadata on a **read-only mount silently fails**, and **moving a file loses its albums/favourites** (source 3). Both hit exactly this user: a read-only NAS share that a phone-sync tool keeps reorganising. The opening is: **Immich-style timeline + viewer, in a single process with SQLite, no ML, under ~150 MB RSS.**

## Screens

| ID | screen | route / how to reach | purpose | key components | states seen |
| --- | --- | --- | --- | --- | --- |
| S01 | Login | `/login` | single-admin / per-user auth | form, error toast | empty, error |
| S02 | First-run setup | first launch | create admin, add first import path | wizard steps, path picker | empty, path not readable (src 3) |
| S03 | Timeline | `/` | all photos, newest first, grouped by day | thumbnail grid, date headers, scrubber, top bar, select mode | empty (no library), loading (scan running), filled, mobile |
| S04 | Viewer | `/photos/:id` | fullscreen photo/video | image w/ zoom, prev/next, toolbar (fav, info, download, add to album, archive), video player | loading, unsupported format, missing file |
| S05 | Info panel | drawer on S04 | EXIF + file details | key/value list, mini map (could) | no EXIF |
| S06 | Folders | `/folders/*path` | browse disk tree, like a file explorer (src 4 folder-view; opt-in feature there, default here) | breadcrumb, folder tiles, grid | empty folder |
| S07 | Albums list | `/albums` | virtual albums | album cards, create button | empty, filled |
| S08 | Album detail | `/albums/:id` | photos in album | grid, title edit, share button | empty, filled |
| S09 | Favourites | `/favorites` | starred photos | grid | empty, filled |
| S10 | Archive | `/archive` | hidden from timeline | grid, unarchive | empty, filled |
| S11 | Search | `/search?…` | filters from src 4 minus ML: time frame, path/folder, filename/extension, camera/lens, media type, favourites/archive/not-in-album, rating, place | filter bar, grid | no results |
| S12 | Shared link (public) | `/s/:token` | album or selection for guests; random token, expiry, password, allow download (src 4 sharing) | grid, viewer, download | expired, password prompt |
| S13 | Libraries admin | `/admin/libraries` | import paths, exclusion globs, scan now (src 3) | path list, glob list, scan button, last-scan status | path unreadable, scan running |
| S14 | Jobs / status | `/admin/jobs` | thumbnail + scan progress | progress bars, counts, errors | idle, running, errors |
| S15 | Trash | `/trash` | missing-on-disk assets, 30-day restore (src 3) | grid, restore, empty trash | empty, filled |

## Flows

```
F01 Browse recent photos on phone
    S03 timeline -> scroll
    happy path clicks: 0 (open URL)
    edge: library still scanning (show partial + progress), 50k+ photos (virtualised grid), slow SD card

F02 Find a photo from a past month
    S03 -> drag scrubber to month -> tap photo -> S04
    happy path clicks: 2
    edge: photos with no EXIF date (fall back to file mtime), wrong camera clock

F03 View, zoom, swipe
    S04 -> pinch zoom -> swipe next/prev -> close back to same scroll position
    happy path clicks: 1 to open, swipes after
    edge: HEIC (server preview), portrait orientation flag, video not playable in browser, live photo pair

F04 Browse by folder (how the files are actually laid out)
    S06 root -> subfolder -> photo -> S04
    happy path clicks: 3
    edge: deep trees, folders with 5k files, Synology @eaDir noise (src 3 glob example)

F05 Favourite / add to album
    S04 -> heart  |  S03 select mode -> multi-select -> add to album -> pick/create
    happy path clicks: 1 | 4
    edge: file moved on disk afterwards (Immich loses metadata, src 3 - keep it via content hash)

F06 Share an album with family
    S08 -> share -> copy link -> guest opens S12
    happy path clicks: 3
    edge: expiry, password, LAN-only vs exposed

F07 Admin adds a library
    S02/S13 -> add path -> exclusions -> scan -> S14 progress -> S03 fills
    happy path clicks: 4
    edge: path unreadable, permissions, network mount (no inotify, src 3), file deleted -> S15

F08 Hide a photo from timeline
    S04 -> archive -> appears in S10 only
    happy path clicks: 2
```

## Components

| component | variants | states | used on |
| --- | --- | --- | --- |
| Thumbnail grid (virtualised) | justified rows, square | loading placeholder, selected, video badge, live badge, fav badge | S03 S06 S08-S12 S15 |
| Date header + scrubber | day/month/year | dragging, hover label | S03 |
| Viewer | photo, video, live | loading, zoomed, error | S04 S12 |
| Top bar / nav | desktop sidebar, mobile bottom bar | active | all |
| Select-mode action bar | timeline, album, trash | n selected | S03 S06 S08 S15 |
| Drawer | info | open/closed | S04 |
| Breadcrumb | — | — | S06 |
| Modal | add to album, share link, confirm | — | S04 S08 |
| Toast | info, error | — | all |
| Progress bar | determinate, indeterminate | — | S14 S03 |
| Form inputs | text, password, path, glob list | error, disabled | S01 S02 S13 |

## Inferred data model

```
Library     id, owner_id, import_paths[], exclusion_globs[], last_scan_at
            evidence: src 3   confidence: high

Asset       id, library_id, path, size, mtime, content_hash, type (image|video),
            mime, width, height, taken_at, timezone, orientation, camera_make,
            camera_model, lens, lat, lon, duration, is_favorite, is_archived,
            deleted_at (trash), live_pair_id
            evidence: src 3 (trash on missing file, assets on map/albums), S04/S05   confidence: high for path/trash, medium for fields

Thumbnail   asset_id, kind (thumbhash blur | small webp | large preview), path, generated_at
            evidence: src 2 (+10-20% disk), src 5 system-settings   confidence: high

Sidecar     asset_id, path (IMG.jpg.xmp preferred over IMG.xmp), read: description,
            rating, datetime, GPS, tags (digiKam:TagsList > lr:HierarchicalSubject > IPTC:Keywords)
            evidence: src 4 xmp-sidecars   confidence: high

Tag         id, name, parent_id (hierarchical);  AssetTag(asset_id, tag_id)
            evidence: src 4 tags   confidence: high

Album       id, owner_id, name, cover_asset_id, created_at;  AlbumAsset(album_id, asset_id)
            evidence: src 3 "adding to albums"   confidence: high

User        id, name, password_hash, is_admin
            evidence: src 3 "library belongs to a single user"   confidence: high

SharedLink  id, album_id | asset_ids, token (random, long), expires_at, password_hash, allow_download
            evidence: src 4 sharing   confidence: high

Folder      derived from Asset.path, no table needed   confidence: guess
```

Relationships: User 1-n Library, Library 1-n Asset, Asset 1-n Thumbnail, Asset 0-1 Sidecar, Asset n-n Tag, Album n-n Asset, Album 1-n SharedLink.

Asset also gains `description`, `rating` (src 4 search filters, xmp-sidecars) and `city, state, country` from offline GeoNames reverse geocoding (src 4 reverse-geocoding).

## Feature matrix

See `features.csv`. Must: 18, should: 10, could: 13, skip: 8.

## Out of scope (cannot or should not be cloned)

- Face recognition, CLIP smart search: ML container needs more RAM than the whole device (src 2).
- Server-side video transcoding: not feasible on 1 GB Pi.
- Phone backup/upload: out of the chosen slice; Syncthing/SMB already do it.
- Native mobile apps: responsive web first.
- ML-based duplicate detection (src 4 duplicates-utility relies on ML): exact-hash duplicates only.
- Immich/PhotoPrism names, logos, UI art and code: not used. Both are open source, but this is a clean-room rebuild from docs. "PhotoPrism" is a registered trademark (src 6), so keep it out of the name and marketing.

## Size

Screens 15, flows 8, entities 9. Hard parts:
1. **1 GB RAM**: no Postgres, no Docker requirement, no Node/JVM heavyweight. Points to one Go/Rust binary or a lean Python process + SQLite, streaming thumbnail generation with libvips, one worker.
2. **Thumbnailing 10k-100k photos on a Pi 3** (slow CPU, SD/USB I/O): resumable background queue, lazy generation on first view, embedded EXIF thumbnails as a fast first pass.
3. **HEIC** (iPhone default): needs libheif on the server, the heaviest decode on the list.
4. Virtualised timeline grid that stays smooth on a phone with 50k items.

Target: 64-bit Raspberry Pi OS (arm64) only, same as both originals; one build target plus amd64 for dev.

Size: **M** (a few weeks) for the must + should rows.
