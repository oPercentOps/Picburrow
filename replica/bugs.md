# Bugs

Found by the test pass of 2026-11-12 (`replica/test-plan.md`). Each fix came with a test that failed before it.
Bugs found earlier, while building, are in `replica/build-log.md`.

| | open | fixed |
| --- | --- | --- |
| S1 | 0 | 2 |
| S2 | 0 | 3 |
| S3 | 0 | 6 |
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

### BUG-007: video length badge unreadable before the photo loads

- Severity: S3 (accessibility, contrast)
- Flow / case: F02-H1 (axe)
- Screen: S03, all grids
- Build: 336b8ba  Browser / device: Chromium, axe

Steps
1. Open the timeline; look at a video tile before its thumbnail has faded in.

Expected: the "0:04" badge readable on anything.
Actual: white text with only a shadow, on the light tile background: axe color-contrast failure (it couldn't measure while the photo was always there; the new fade-in exposed it).
Status: fixed (badges get their own dark backing).

### BUG-008: unarchiving the last photo in the viewer jumps back to one already unarchived

- Severity: S3
- Flow / case: F05-A2
- Screen: S04 opened from S10
- Build: 336b8ba  Browser / device: Chromium 1440px

Steps
1. Archive two photos. Open the first on the Archive page, Unarchive, Unarchive.

Expected: moves to the second, then closes: nothing is left in the archive.
Actual: after the second, it went back to the first photo, which is no longer archived.
Evidence: `F05-A2` (rewritten to step through with the viewer).
Status: fixed (the viewer skips photos that left its list).

### BUG-009: a wrong share-link password sends the visitor to the owner's sign-in page

- Severity: S2 (the visitor can't retry and lands somewhere that makes no sense to them)
- Flow / case: F06-H1
- Screen: S12
- Build: 336b8ba  Browser / device: Chromium 1440px

Steps
1. Open a password-protected share link in a fresh browser, type a wrong password.

Expected: "Wrong password", try again.
Actual: redirected to /login: the API client treated every 401 as an expired session.
Evidence: `F05-A1 … F06-H1` failed at the password gate.
Status: fixed (share-link calls never redirect).

### BUG-010: the scrubber sometimes lands on (and announces) the month next to the one asked for

- Severity: S3
- Flow / case: F02-H1
- Screen: S03
- Build: 336b8ba  Browser / device: Chromium 1440px

Steps
1. Press End on the scrubber in a long timeline.

Expected: the oldest month, announced as such.
Actual: 1 run in 3: months around the target loaded and replaced their estimated heights, the page shifted and the tracker relabelled the slider with a neighbouring month.
Evidence: `F02-H1` failed once in 3 full runs, after the albums work made the page longer.
Status: fixed (re-anchors on the target while the layout settles; 3 full runs green after).

### BUG-011: on a phone, Favourites, Archive, Trash and the admin pages can't be reached

- Severity: S2 (whole features unreachable on the main device, no workaround but typing URLs)
- Flow / case: N-1
- Screen: bottom navigation (all screens < 768 px)
- Build: 996f188  Browser / device: Chromium 390px

Steps
1. Open the gallery on a phone. Look for Favourites, Archive, Trash or Libraries.

Expected: a way in (the design spec had a "More" item in the bottom bar).
Actual: the bottom bar had Photos, Folders, Albums, Search, and nothing else; the sidebar with the rest is desktop-only. Present since milestone 2; found while screenshotting the new Trash page on a phone.
Evidence: `N-1 BUG-011` (phone project).
Status: fixed (a fifth "More" item opens a page with the rest and Sign out).

## To check (not reproduced as bugs)

- **Tap right after a fast swipe is ignored** (phone viewer). In Chromium touch emulation a tap within ~0.5 s of a swipe sends no click: the browser treats it as catching the fling. This is standard browser behaviour, but check on a real phone that buttons don't feel dead after swiping.
- **Real iPhone HEIC files** were never decoded here (no HEIC encoder to make a sample).
- **Safari / iOS**: only Chromium was available; the phone flows have not run in WebKit.
- **Two tabs**: a favourite set in one tab shows in another only after a reload. Probably fine for a family gallery.
- **Memory on a 1 GB Pi 3** with a real library (milestone-1 exit test).
