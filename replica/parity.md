# Parity report: Picburrow vs Immich / PhotoPrism (browse + view slice)

Build: 0.1.17 (0.1.10 + RAW previews, library counts, the phone catch-up: lookup, API keys, upload into yyyy-MM folders, and Delete to Trash). Date: 2026-11-20.

| | score |
| --- | --- |
| **Overall** | **91.9** (features only; layout not measured, see below) |
| Features | 91.9 / 100, must-haves **18 of 18**, should-haves **12.5 of 13** |
| Layout | not measured |
| Open bugs | 0 S1, 0 S2 (`replica/bugs.md`) |
| Tests at this build | `make test` green; e2e 63 passed (the 5 skips are phone-only tests skipped in the desktop run) |

**Verdict: shippable** by this report's bar: every must-have works, the feature score reaches 80, and no S1 or S2 bug is open. As before, that is a family/LAN release. The real-hardware checks still decide whether it is a *Raspberry Pi 3* release.

Up from 91.8 at 0.1.10. What moved:

- **RAW previews**: no to done. Each preview is made from the JPEG the camera embedded. Eight RAW extensions are covered; only a DNG was tested.
- **Phone backup/upload**: was left out ("out of slice"), now scored as a **should, partial**. An iOS Shortcut with a personal API key asks which phone photos are missing, then uploads them into yyyy-MM folders. It isn't background backup and there is no Android flow, so it counts half. Scoring a row that used to be skipped adds to the total, so the score rose by only 0.1 although two features moved.
- **Trash**: still done; photos can now be deleted from the app (selection bar and viewer) and restored.

The one should-have not fully done is phone backup. Everything else still missing is a could-have.

## Feature parity (parity.py)

features 91.9  (45 counted, must-haves 18 of 18 done)

## By area, weakest first
- backup                        50.0  (1 features)
- search                        66.7  (2 features)
- organize                      79.2  (8 features)
- clients                       83.3  (2 features)
- library                       93.2  (9 features)
- browse                        94.4  (8 features)
- view                         100.0  (11 features)
- auth                         100.0  (2 features)
- sharing                      100.0  (2 features)

## Missing, in build order
- [should] backup: Phone backup/upload, partial  (was skip. 0.1.12-0.1.17: a catch-up iOS Shortcut with a personal API key asks POST /api/assets/lookup which phone photos are missing, then sends them to POST /api/assets/upload, saved into yyyy-MM folders of the sender's library, never overwriting. Run by hand or by a Shortcut automation; no background upload, no Android flow. Immich: native apps with background backup)
- [could] browse: Map of geotagged photos, no  (needs map tiles from an external provider)
- [could] library: Write favourites/albums back to XMP sidecars, no  (Immich silently fails on read-only mounts; we store in DB and only write sidecars when the mount is writable, with a visible warning otherwise)
- [could] organize: Duplicate finder (exact hash), no  (Immich uses ML similarity; exact content hash is our 1 GB substitute)
- [could] organize: Locked folder behind a PIN, no  (Immich locked-folder.md: 15 min unlocked session)
- [could] search: Reverse geocoding (place names) from offline dataset, no  (Immich uses GeoNames loaded locally; a city-level extract keeps it small)
- [could] clients: Interface in the user's language, partial  (Immich: dozens of community translations. Here: English and French, chosen by the browser language; no in-app switch)
- [could] library: XMP sidecar read (description rating date GPS tags), partial  (IMG.jpg.xmp preferred over IMG.xmp, as in Immich's xmp-sidecars.md. Read: keywords (digiKam TagsList, Lightroom hierarchicalSubject, dc:subject), description, stars (a rating set in the app wins). Shown in Details, found by Search; a sidecar that appears, changes or goes away later is followed without remaking previews. Not read from the sidecar: date and GPS)
- [could] organize: Hierarchical tags (read from XMP/IPTC), partial  (Immich tags.md. Tag paths from sidecars (People/Ann) are kept and searchable; no tag tree to browse, and tags stored inside the photo file are not read)

## Left out on purpose (not scored)
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
- Tested guide for getting phone photos into the folder (iPhone and Android)
## Layout diff: not measured

`imgdiff.py` needs a screenshot of the original and of the clone for each screen, at the same viewport and in the same state. There is none of the original: no Immich or PhotoPrism instance was run, and the images in their public docs show other data, other states and other sizes. A score from those would be a made-up number, so there isn't one. Tried on 2026-11-14 and not retried for 0.1.17: Immich's public demo was not reachable from the build container then.

To measure it, run Immich on your own machine with the same fixture library (`e2e/make-fixtures.sh`), screenshot S03, S04, S05 and S06 at 1440×900 and 390×844 into `replica/screens/`, then:

```bash
python3 imgdiff.py replica/screens/S03-timeline.png replica/clone-screens/S03-timeline.png --json > replica/diffs/S03.json
python3 parity.py replica/features.csv --visual replica/diffs/*.json --markdown > replica/parity.md
```

Layout is 20% of the overall score; colour is ignored on purpose (the rebrand changes it).

Clone against itself (layout mode, this 0.1.17 e2e run against the committed screenshots): no regression. The low scores have three causes: thumbnails that had not loaded when the screenshot was taken (S03, S06, S19), a different random photo (S18 slideshow, 2-4), and the new **API keys** item in the sidebar and on More (S17, 55). On More at 390 px that item pushes the AGPL line under the floating nav, but the page has bottom padding for the nav and scrolls to it. The committed screenshots are kept, since the new ones show half-loaded grids; they lack the API keys item.

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
| F03 viewer | favourite, archive, info, download, plus edit, add to album, share, stacks | favourite, archive, info, download, zoom, swipe, add to album, rotate (kept in the app, the file is not written) | gap: other edits, stacks |
| F05 albums | albums, shared with users as editor/viewer | albums from a selection or the viewer; share with anyone by link | gap: sharing with other accounts |
| Accounts | users with their own libraries; partner sharing, one-way, switched on later between existing accounts | family accounts with their own folder; an admin shares a library with any account, new or existing, and can stop (favourites, ratings and archive shared; albums personal); deleting the owner hands the library to another member | different: sharing is two-way inside a library, and set by an admin, not by each person |
| Trash | deleted-from-disk files go to trash, 30 days | same, and putting the file back restores albums and favourites; since 0.1.16 Delete in the app moves the file into a hidden `.picburrow-trash` folder next to it, Restore moves it back, and emptying or 30 days removes it | keep (better on restore); to check: a two-way sync tool may pass that move back to the phone as a delete; the guide does not mention it |
| Search | metadata filters + CLIP smart search + OCR + faces | words in name/folder/camera, date range, type, favourites | keep; ML search stays out (no RAM) |
| F06 share | public link with expiry, password, allowed actions | link with password, expiry, view-only; visitors see nothing else | parity |
| Loading | thumbhash blur, small, large | 16 px WebP blur, small, large | parity |
| Video that the browser can't decode (HEVC, 4K, 10-bit, AVI/MKV) | transcoded by policy, hardware acceleration optional | a 720p H.264 copy made in the background, one at a time, paused above 75 °C; the video being watched goes first; an admin switch turns it off | parity on the outcome; slower: until the copy exists the original plays, and may still freeze |
| Slideshow | from the viewer, in order or shuffled | random, full screen, from Photos; starts by itself after idle time if asked (photo frame) | keep (better as a photo frame); gap: no in-order slideshow of an album |
| New photos in the folder | external libraries: nightly scan, optional folder watcher (its docs) | a check every 15 minutes that scans only when a folder changed, plus the 03:00 full scan | parity |
| Memories | "on this day" from earlier years, shown on the timeline | an On this day button when there is something to see, opening those photos by year | parity on the basics; no generated clips |
| Tags from other tools | reads XMP sidecars: description, rating, date, GPS, tags | reads keywords, description and stars from sidecars; shows them in Details; Search finds them | gap: date and GPS from sidecars, tags inside the file, a tag tree |
| Language | dozens of translations, chosen in settings | English or French, following the browser | gap: more languages, an in-app switch |
| Live photos | PhotoPrism plays them on hover | one item with a Live badge; Play shows the clip over the still | parity (Apple pairs; Google/Samsung embedded clips not read) |
| Phone app | Immich native apps; PhotoPrism installable PWA | installable web app (manifest, icons) | parity with PhotoPrism |
| Phone backup | Immich: native apps with background upload; its iOS reviews call it unreliable (fixes.md H5) | an iOS Shortcut with an API key: lookup finds the photos not yet indexed, upload sends them into yyyy-MM folders, never overwriting; run by hand or by a Shortcut automation | gap: not tried on a real iPhone, nothing for Android, no background upload |
| Promise on the landing page | (none) | `docs/index.html` still says "Your folders, untouched … Nothing is moved, renamed or rewritten". Since 0.1.15/0.1.16 upload adds files and Delete moves them. The README already says this correctly | **fix**: reword to "never rewrites your photos; Delete moves a file to a hidden trash folder you can restore from" |

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

1. **The real-hardware pass**, still the release decider: a 1 GB Pi 3 with real iPhone HEICs, live photos and a real RAW from a camera other than the DNG fixture, memory during a video conversion and during an upload (exiftool reads each uploaded file), Safari on an iPhone, a screen reader.
2. **Run the catch-up Shortcut on a real iPhone**: lookup, upload, a second run that finds nothing to send. This is the only should-have not fully done, and the biggest user complaint about the originals (fixes.md H5).
3. **Fix the landing page's "untouched" claim** (behaviour diff above). It is one sentence, but it is a promise that is no longer true.
4. **Android catch-up**: the same lookup + upload routes from an HTTP-capable automation app, or a tested Syncthing recipe in the guide. Upload would then count as done for both phones.
5. **Duplicate finder** by exact hash. Uploads and sync tools together make duplicates more likely, and the content hash is already stored.
