# Parity report: Picburrow vs Immich / PhotoPrism (browse + view slice)

Build: 0.1.8 (028e65c): + video playback copies, shared family library, slideshow, sleep mode, French. Date: 2026-11-14.

| | score |
| --- | --- |
| **Overall** | **86.8** (features only; layout not measured, see below) |
| Features | 86.8 / 100, must-haves **18 of 18**, should-haves **11 of 12** (one partial) |
| Layout | not measured |
| Open bugs | 0 S1, 0 S2 (`replica/bugs.md`) |
| Tests at this build | `make test` green; e2e 60 passed (the 5 skips are phone-only tests skipped in the desktop run; they pass in the phone run) |

**Verdict: shippable** by this report's bar: every must-have works, the feature score reaches 80, and no S1 or S2 bug is open (`replica/bugs.md`). That is a family/LAN release; the real-hardware checks below still decide whether it is a *Raspberry Pi 3* release.

Down from 87.6 the same day: the review research (`replica/fixes.md`, M5) found a should-have the matrix lacked, and Picburrow only half does it: new photos are picked up at start-up, at 03:00 or on Scan now, not within minutes. The verdict does not change.

What moved between 87.1 and 87.6:

- **Video transcoding** was "left out on purpose" (thought impossible in 1 GB). It now exists as background 720p H.264 playback copies, so the row is scored as a should-have, done.
- **Partner sharing** went from no to partial: a family member can share one library, but only when the account is created, and a person has either a folder of their own or the shared one.
- Two rows the matrix was missing, both in Immich: **slideshow** (done) and **interface languages** (partial: English and French against dozens).
- **Sleep mode** (photo frame) is ours; not scored.

Between those two reports the score barely moved: one should-have was added and done, the other new rows are could-haves, and two of those are partial. Apart from the rescan row above, everything still missing is a could-have.

## Feature parity (parity.py)

features 86.8  (44 counted, must-haves 18 of 18 done)

## By area, weakest first
- search                        66.7  (2 features)
- organize                      75.0  (8 features)
- sharing                       83.3  (2 features)
- clients                       83.3  (2 features)
- library                       86.4  (9 features)
- browse                        88.9  (8 features)
- view                          92.0  (11 features)
- auth                         100.0  (2 features)

## Missing, in build order
- [should] library: New photos appear within minutes of landing in the folder (no cron, no button), partial  (scan at start-up, 03:00 and on Scan now only)
- [could] browse: Map of geotagged photos, no  (needs map tiles from an external provider)
- [could] browse: On this day / memories, no
- [could] library: Write favourites/albums back to XMP sidecars, no  (Immich silently fails on read-only mounts; we store in DB and only write sidecars when the mount is writable, with a visible warning otherwise)
- [could] library: XMP sidecar read (description rating date GPS tags), no  (Immich xmp-sidecars.md; IMG.jpg.xmp preferred over IMG.xmp)
- [could] organize: Duplicate finder (exact hash), no  (Immich uses ML similarity; exact content hash is our 1 GB substitute)
- [could] organize: Hierarchical tags (read from XMP/IPTC), no  (Immich tags.md)
- [could] organize: Locked folder behind a PIN, no  (Immich locked-folder.md: 15 min unlocked session)
- [could] search: Reverse geocoding (place names) from offline dataset, no  (Immich uses GeoNames loaded locally; a city-level extract keeps it small)
- [could] view: Basic edit (rotate), no
- [could] view: RAW (DNG/CR2/ARW) previews, no  (use embedded JPEG preview only; full RAW decode is too heavy for 1 GB)
- [could] clients: Interface in the user's language, partial  (English and French, chosen by the browser language; no in-app switch)
- [could] sharing: Partner sharing (whole library to another user), partial  (one shared library, set when the account is created; a person has a folder of their own or a shared one, not both)

## Left out on purpose (not scored)
- Phone backup/upload: out of slice; existing sync tools (Syncthing/SMB/phone backup apps) already drop files on the NAS
- Face recognition / people: needs ML container; Immich says ML needs RAM beyond 4 GB - impossible in 1 GB
- Smart (CLIP) search by content: same ML limit
- Native mobile apps: responsive web app first
- Paid-member features (PhotoPrism interactive world maps): PhotoPrism README: members get interactive world maps; free here via could-row map
- WebDAV access to the library: PhotoPrism feature; out of browse+view slice
- OCR text search: Immich searching.md; needs ML

## Yours, not in the original (not scored)
- Keep metadata stable when a file moves (content hash)
- One-command install on Raspberry Pi OS (.deb + systemd)
- Warn when the database or thumbnails are on the SD card
- Automatic database backup before every upgrade
- Full capture details (aperture, shutter, ISO, focal length, flash)
- Batch rating from the selection bar
- Sleep mode: slideshow starts when the screen is left alone (photo frame)
- Tested guide for getting phone photos into the folder (iPhone and Android): not written yet

## Layout diff: not measured

`imgdiff.py` needs a screenshot of the original and of the clone for each screen, at the same viewport and in the same state. There is none of the original: no Immich or PhotoPrism instance was run, and the images in their public docs show other data, other states and other sizes. A score from those would be a made-up number, so there isn't one. Tried again on 2026-11-14: Immich's public demo is not reachable from the build container.

To measure it, run Immich on your own machine with the same fixture library (`e2e/make-fixtures.sh`), screenshot S03, S04, S05 and S06 at 1440×900 and 390×844 into `replica/screens/`, then:

```bash
python3 imgdiff.py replica/screens/S03-timeline.png replica/clone-screens/S03-timeline.png --json > replica/diffs/S03.json
python3 parity.py replica/features.csv --visual replica/diffs/*.json --markdown > replica/parity.md
```

Layout is 20% of the overall score; colour is ignored on purpose (the rebrand changes it).

Clone against itself (`--mode pixel`, screenshots from this e2e run against the committed ones): no layout regression. The pages differ only where the fixture photos are random or a thumbnail had not loaded yet. `S14-indexing.png` is refreshed: it now shows the video conversion switch.

## Behaviour diff (from the originals' public docs vs the clone as tested)

| flow | original does | clone does | fix or keep |
| --- | --- | --- | --- |
| Install | Immich: Docker Compose, Postgres, ML container; **6 GB RAM minimum, 4 GB without ML**, "can be too heavy to run on a Raspberry Pi" (its FAQ) | one 10 MB static arm64 binary + `apt install libvips-tools libimage-exiftool-perl ffmpeg`; peak RAM measured below | keep: this is the reason to exist |
| F07 first run | register admin, then Administration → External libraries → create, add import path, mount it in Docker, scan (several screens) | one form: username, password, folder; indexing starts and the timeline fills in | keep (fewer steps) |
| F07 library under a hidden folder / Synology `@eaDir` | exclusion globs against the full path | globs against the path inside the library (BUG-002 found and fixed here) | keep |
| F01/F07 unmounted share | a file gone on rescan goes to trash; purged after 30 days | an unreadable or suddenly empty library folder is treated as offline: nothing goes to trash | keep (better) |
| Moved file | Immich docs: "If you move an asset to another location within the library all such metadata will be lost upon rescan" (a known issue) | relinked by a quick content hash: favourite kept (tested) | keep (better) |
| Edits on a read-only mount | metadata edits "silently fail" (Immich docs, danger box) | favourites and archive live in the database and work on read-only mounts | keep (better); XMP write-back is a could |
| F04 folder view | opt-in under Account Settings | on by default, URL mirrors the disk | keep |
| F02 find a month | date scrubber | scrubber, keyboard slider, sized by photo count | parity |
| F03 viewer | favourite, archive, info, download, plus edit, add to album, share, stacks | favourite, archive, info, download, zoom, swipe, add to album | gap: edit, stacks |
| F05 albums | albums, shared with users as editor/viewer | albums from a selection or the viewer; share with anyone by link | gap: sharing with other accounts |
| Accounts | users with their own libraries; partner sharing, one-way, switched on later between existing accounts | family accounts with their own folder, or sharing one library (favourites, ratings and archive shared; albums personal); deleting the owner hands the library to another member | gap: sharing with an account that already exists, and having both your own and a shared library |
| Trash | deleted-from-disk files go to trash, 30 days | same, and putting the file back restores albums and favourites | keep (better) |
| Search | metadata filters + CLIP smart search + OCR + faces | words in name/folder/camera, date range, type, favourites | keep; ML search stays out (no RAM) |
| F06 share | public link with expiry, password, allowed actions | link with password, expiry, view-only; visitors see nothing else | parity |
| Loading | thumbhash blur, small, large | 16 px WebP blur, small, large | parity |
| Video that the browser can't decode (HEVC, 4K, 10-bit, AVI/MKV) | transcoded by policy, hardware acceleration optional | a 720p H.264 copy made in the background, one at a time, paused above 75 °C; the video being watched goes first; an admin switch turns it off | parity on the outcome; slower: until the copy exists the original plays, and may still freeze |
| Slideshow | from the viewer, in order or shuffled | random, full screen, from Photos; starts by itself after idle time if asked (photo frame) | keep (better as a photo frame); gap: no in-order slideshow of an album |
| Language | dozens of translations, chosen in settings | English or French, following the browser | gap: more languages, an in-app switch |
| Live photos | PhotoPrism plays them on hover | one item with a Live badge; Play shows the clip over the still | parity (Apple pairs; Google/Samsung embedded clips not read) |
| Phone app | Immich native apps; PhotoPrism installable PWA | installable web app (manifest, icons) | parity with PhotoPrism |

## Measured on the build box (x86 container, not a Pi)

| | peak RSS |
| --- | --- |
| Server alone | 13-19 MB |
| Server + exiftool + thumbnailer, 137-file fixture library, both e2e suites | 98-136 MB |
| `vipsthumbnail` on a generated **12 MP HEIC** (iPhone default size) | 123 MB |
| `vipsthumbnail` on a generated **48 MP HEIC** | **341 MB**, so about 385 MB with the server and exiftool |

Not measured: memory while ffmpeg makes a video playback copy. It runs after all previews are done, so it does not add to the 48 MP peak, but on a 1 GB Pi it is the next number to take.

The 48 MP case is over the 350 MB target set in `architecture.md`. It still fits a 1 GB Pi (about 750 MB usable) only because thumbnails are made one at a time (`--workers 1`). Keep that default. A real 1 GB Pi 3 with real iPhone files is still the deciding test.

## Top five to build next

Reordered by what users of the originals say (`replica/fixes.md`, round 2):

1. **New photos appear within minutes**: a light rescan every 15 minutes. The only should-have not fully done.
2. **The real-hardware pass**: a 1 GB Pi 3 with real iPhone HEICs and live photos, memory during a video conversion, Safari on an iPhone, a screen reader. Still open in `deploy.md` and `test-plan.md`; it decides the release.
3. **On this day**: a small row on the timeline.
4. **Share a library with an account that already exists**: turns partner sharing from partial to done; the membership table is already there.
5. **Rotate**, stored in the database and applied to previews.

Then XMP sidecar read and the duplicate finder. Outside the code: a tested guide for getting phone photos into the folder.
