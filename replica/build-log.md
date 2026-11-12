# Build log

Milestone 1 (vertical slice from architecture.md): S02 setup, then scan, S03 timeline, S04 viewer, plus S01 login and S05 info.
Milestone 2: S03 scrubber, S04 favourite/archive, S06 folders, S09 favourites, S10 archive, S13 libraries, S14 indexing.
Checked by `server/app_test.go` (Go, real files), `e2e/slice.cjs` and `e2e/milestone2.cjs` (Playwright against the real binary, 390 px dark + 1440 px light).

| ID | screen | date | status | missing | harder than expected |
| --- | --- | --- | --- | --- | --- |
| — | shell: routing for every screen, sidebar / bottom nav, tokens wired, auth gate | 2026-11-12 | done | stub pages for S06-S15 | — |
| S02 | First-run setup | 2026-11-12 | done | — | setup must be one-shot even if two browsers race it (insert … where not exists) |
| S01 | Login | 2026-11-12 | done | — | — |
| S03 | Timeline | 2026-11-12 | partial | select mode, thumbhash blur placeholders | rows must be measured from the inner container, not page width minus padding (was 4 px too wide on mobile); a full-height scrubber over the photos stole taps on mobile, so there only its thumb is touchable |
| S04 | Viewer | 2026-11-12 | partial | add to album, live photos | SvelteKit resets focus on `goto`, which killed arrow keys on deep-linked photos: keys now on window + keepFocus. The overlay moved to the layout so every grid can open it with its own sequence |
| S05 | Info panel | 2026-11-12 | partial | mini map, albums, tags | — |
| S06 | Folders | 2026-11-12 | done | — | subfolder counts come from one range scan on the dir index, folded into immediate children |
| S09 | Favourites | 2026-11-12 | done | paging beyond 5000 (shortcut) | — |
| S10 | Archive | 2026-11-12 | done | paging beyond 5000 (shortcut) | — |
| S13 | Libraries | 2026-11-12 | done | create/delete a second library | a PATCH saved valid fields before rejecting a bad one; now validates all, then writes once. Removing a folder asks first: its photos go to Trash |
| S14 | Indexing | 2026-11-12 | done | — | — |
| S07 S08 S11 S12 S15 | albums, search, shared link, trash | — | stub | should/could features | — |

## Backend findings

- **Unmounted share = empty folder.** A naive rescan would mark the whole library missing and the 30-day purge would drop every album and favourite. Roots that error or come back empty while they used to have files are now treated as offline. Covered by a test that fails without the guard.
- **Dateless videos landed in 1904.** A zero QuickTime date passed through exiftool's UTC conversion becomes the QuickTime epoch. Dates ≤ 1904 now count as missing, so the file time is used.
- **Thumbnail decodes needed one global limit.** On-demand requests for different photos aren't de-duplicated, so a phone scrolling a fresh library could start about six vipsthumbnail processes at once. Worker and on-demand now share `--workers` (default 1).
- **WebP encoding was the cost, not decoding.** `effort=0` gave about 2x on encode; indexing the fixture library went from 67 s to 42 s (x86 container). The small thumbnail is now made from the large one, so the original is decoded once (matters for HEIC).

- **A scan requested during a scan was dropped.** Saving library settings mid-scan would not pick up the new folder until the nightly run (found because the Go test went flaky). Now it queues one more pass.

## Measured (x86 container, not a Pi)

- 133 files (128 photos, 3 videos, 1 x 48 MP JPEG, 1 corrupt) indexed in 42 s.
- Peak RSS of the whole process tree (server + exiftool + vipsthumbnail/ffmpeg): **98-136 MB** across runs, both e2e suites included; server alone 18-19 MB.
- Binary: 11 MB (amd64), 10 MB (arm64, static).

## Not verified yet

- A real Raspberry Pi 3 with 1 GB RAM (the milestone-1 exit test: 10k phone photos, RSS < 350 MB).
- iPhone HEIC files: this build box has a HEIC decoder but no encoder to make a sample. The 48 MP test file is a JPEG, which decodes cheaply, so the HEIC worst case is still unmeasured.
