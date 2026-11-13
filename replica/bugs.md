# Bugs

Found by the test pass of 2026-11-12 (`replica/test-plan.md`). Each fix came with a test that failed before it.
Bugs found earlier, while building, are in `replica/build-log.md`.

| | open | fixed |
| --- | --- | --- |
| S1 | 0 | 2 |
| S2 | 0 | 1 |
| S3 | 0 | 3 |
| S4 | 0 | 0 |

### BUG-001: right after setup the library looks empty and idle

- Severity: S3
- Flow / case: F07-H1, F01-E1
- Screen: S03
- Build: 42f7141  Browser / device: Chromium 1440px

Steps
1. Fresh install, complete setup.
2. Look at the timeline immediately.

Expected: the indexing banner, photos appearing.
Actual: "No photos here yet" and no banner until the scan goroutine happened to start; `/api/jobs` said idle with nothing to do.
Evidence: the e2e setup step "finished indexing" in 1.5 s; `TestStartScanIsVisibleImmediately`.
Suspected cause: setup started the scan with `go a.scanAll()`, which marked it running only once scheduled.
Status: fixed in 8115f94 (`startScan` marks the scan running before returning; used by setup, Scan now and library edits).

### BUG-002: a library inside a hidden folder indexes nothing

- Severity: S1 (core flow blocked, silently)
- Flow / case: F07-E6
- Screen: S02, S03
- Build: 42f7141  Browser / device: any

Steps
1. Set up with a folder under a dot-folder, e.g. `/home/pi/.local/share/photos` (here: `e2e/.fixtures`).
2. Wait for indexing.

Expected: all photos indexed.
Actual: zero photos, no error anywhere.
Evidence: every e2e timeline test found no tiles; `TestLibraryUnderHiddenFolder`.
Suspected cause: exclusion patterns were matched against the full path, so the default "skip hidden folders" pattern matched the library's own parent folder.
Status: fixed in 8115f94 (patterns match the path inside the library folder).

### BUG-003: with a relative data folder no thumbnails are ever made

- Severity: S1 (the README's own command; no previews at all)
- Flow / case: F07-E7
- Screen: S03, S04
- Build: 42f7141  Browser / device: any

Steps
1. Run `./raspicture --data-dir ./data` as the README said.
2. Set up and wait.

Expected: previews.
Actual: every file got "rename …tmp.webp: no such file or directory" and showed "No preview".
Evidence: 136 of 136 thumb_errors in the e2e database; `TestRelativeDataDir`.
Suspected cause: `vipsthumbnail -o` resolves a relative output path against the input photo's folder, not the working directory.
Status: fixed in 8115f94 (absolute output path; the data dir is made absolute at start-up too).

### BUG-004: tile names leave out what the tile shows

- Severity: S3 (accessibility, WCAG 2.5.3 Label in Name)
- Flow / case: F01-E6
- Screen: S03, S06, S09, S10
- Build: 42f7141  Browser / device: Chromium, axe

Steps
1. Open the timeline with a video and a corrupt file.
2. Run axe, or say "click 0:04" with voice control.

Expected: the accessible name contains the visible "0:04" / "No preview".
Actual: names were only "Video from …" / "Photo from …"; axe `label-content-name-mismatch`.
Evidence: axe scan on S03, S06; `F01-E6 BUG-004`.
Status: fixed in d513470.

### BUG-005: favourite / archive hit the previous photo while the next one loads

- Severity: S2 (the wrong photo is changed; recoverable, but archive hides it from the timeline)
- Flow / case: F05-E3
- Screen: S04
- Build: 42f7141  Browser / device: Chromium 1440px, slow network

Steps
1. Open a photo, press → and then F straight away (or tap Archive) on a slow Pi.

Expected: the new photo is favourited, or nothing happens until it has loaded.
Actual: the photo you just left was favourited / archived, because the viewer still held its details.
Evidence: `F05-E3 BUG-005` (route delayed 1.5 s); first seen as a flaky F05-H1.
Status: fixed in d513470 (actions and buttons wait until the loaded photo is the one on screen; download uses the current id).

### BUG-006: the scrubber announces the wrong month at the end of the timeline

- Severity: S3
- Flow / case: F02-H1
- Screen: S03
- Build: 42f7141  Browser / device: Chromium 1440px

Steps
1. Focus the scrubber, press End.

Expected: "December 2025" (the oldest month).
Actual: jumped there, then the slider announced and drew "January 2026": the oldest month can't reach the top of the page, so the scroll tracker picked the month above it.
Evidence: `F02-H1` failed on one run in three.
Status: fixed in d513470 (at the bottom of the page the last month is current).

## To check (not reproduced as bugs)

- **Tap right after a fast swipe is ignored** (phone viewer). In Chromium touch emulation a tap within ~0.5 s of a swipe sends no click: the browser treats it as catching the fling. This is standard browser behaviour, but check on a real phone that buttons don't feel dead after swiping.
- **Real iPhone HEIC files** were never decoded here (no HEIC encoder to make a sample).
- **Safari / iOS**: only Chromium was available; the phone flows have not run in WebKit.
- **Two tabs**: a favourite set in one tab shows in another only after a reload. Probably fine for a family gallery.
- **Memory on a 1 GB Pi 3** with a real library (milestone-1 exit test).
