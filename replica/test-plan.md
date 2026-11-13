# Test plan: RaspictureGalleryNas

Build: d513470  Date: 2026-11-12  Env: local binary on a fresh data dir, generated fixture library (`e2e/make-fixtures.sh`: 128 dated photos, 3 dateless videos, a 48 MP JPEG, a corrupt file, a Synology `@eaDir`, and 3 photos with accented/emoji/HTML-looking/80-character names in an emoji folder dated 31 Dec 23:30 +01:00).

How to run: `make test` (Go) and `make e2e` (Playwright, Chromium, desktop 1440 px light + phone 390 px dark touch).
Every Playwright test also fails on any console error, page error, 5xx response or unexpected dialog. `axe` (WCAG 2.1 A/AA) runs on every screen marked ♿.

auto: `e2e` = Playwright spec, `go` = `server/app_test.go`, `manual` = needs a person or hardware.
Result: ✅ pass · ⏭ not run (reason) · 🔲 manual, not done yet.

| case | flow | type | steps | expected | auto | result |
| --- | --- | --- | --- | --- | --- | --- |
| F07-H1 | first-run setup | happy ♿ | open `/`, redirected to setup, enter admin + folder | indexing starts, lands on the timeline | e2e | ✅ |
| F07-E1 | | edge: empty form | submit empty | browser blocks it, nothing created | e2e | ✅ |
| F07-E2 | | edge: double submit | two clicks in one tick | one admin, user on the timeline | e2e | ✅ |
| F07-E3 | | edge: second browser runs setup again | POST /api/setup after setup | 409, nothing changes | e2e, go | ✅ |
| F07-N1 | | negative: short password | 5 characters | "at least 8 characters" | e2e, go | ✅ |
| F07-N2 | | negative: folder doesn't exist | /does/not/exist | "doesn't exist" | e2e, go | ✅ |
| F07-N3 | | negative: relative path | photos/backup | "full path" message | e2e | ✅ |
| F07-E6 | | edge: library under a hidden folder | library at ~/.local/photos | photos indexed, hidden subfolders skipped | go | ✅ (BUG-002) |
| F07-E7 | | edge: relative `--data-dir` | `--data-dir ./data` | thumbnails made inside the data dir | go | ✅ (BUG-003) |
| S01-H1 | sign in | happy ♿ | username + password | timeline | e2e (desktop + phone) | ✅ |
| S01-N1 | | negative: wrong password | wrong password | "Wrong username or password", field marked invalid | e2e, go | ✅ |
| S01-N2 | | negative: guessing | 11 tries in a minute | 429 "Too many attempts" | go | ✅ |
| S01-E1 | | edge: deep link without session | /admin/libraries signed out | sign-in page | e2e | ✅ |
| S01-E2 | | edge: expired / forged session | bad cookie | sign-in page | e2e | ✅ |
| S01-E3 | | edge: sign out | Sign out | session gone, /api/me 401 | e2e | ✅ |
| F01-H1 | browse recent photos | happy ♿ | open `/` | days newest first, rows fit, no sideways scroll | e2e (desktop + phone) | ✅ |
| F01-E1 | | edge: still indexing | right after setup | banner says it's indexing | e2e, go | ✅ (BUG-001) |
| F01-E2 | | edge: corrupt file | — | "No preview" tile | e2e | ✅ |
| F01-E3 | | edge: keyboard only | Tab to a photo, Enter, Escape | viewer opens/closes, focus back on the photo | e2e | ✅ |
| F01-E4 | | edge: back button mid-flow | open photo, browser Back | viewer closes, scroll position kept | e2e | ✅ |
| F01-E5 | | edge: slow / failing server | month request fails | "This month didn't load" + Try again works | e2e | ✅ |
| F01-E6 | | edge: voice control / screen reader names | — | tile names include duration and "No preview" | e2e + axe | ✅ (BUG-004) |
| F01-E7 | | edge: second user's data | other account | invisible: 404 on every asset route | go | ✅ |
| F01-E8 | | edge: unmounted share | library folder empty / missing | nothing moves to trash | go | ✅ |
| F02-H1 | find a past month | happy ♿ | scrubber: End, Home | jumps to oldest / newest, announced month correct | e2e | ✅ (BUG-006) |
| F02-E1 | | edge: drag | drag the rail | timeline moves | e2e | ✅ |
| F02-E2 | | edge: time zones | 23:30 on 31 Dec +01:00, browser in Auckland | still 31 Dec | e2e | ✅ |
| F02-E3 | | edge: no EXIF date | dateless videos | file date (today); never 1904 | e2e, go | ✅ |
| F03-H1 | view, zoom, swipe | happy ♿ | deep link, arrows, `i` | next/prev, details incl. camera and UTC offset | e2e | ✅ |
| F03-E1 | | edge: zoom | double-click | original swapped in, back to preview | e2e | ✅ |
| F03-E2 | | edge: video | open video | plays from original, Range 206 | e2e, go | ✅ |
| F03-E3 | | edge: corrupt file | open it | message + Download original | e2e | ✅ |
| F03-E4 | | edge: first / last photo | newest, oldest | no Previous / no Next | e2e | ✅ |
| F03-E5 | | edge: accents, emoji, HTML in names | open the 3 named files | names shown as text, no script runs | e2e | ✅ |
| F03-E6 | | edge: phone gestures | swipe left, Details, swipe down | next photo, sheet opens, viewer closes | e2e (phone) | ✅ |
| F03-E7 | | edge: HEIC from an iPhone | open a real .HEIC | preview shown | manual | 🔲 needs a real iPhone file |
| F04-H1 | browse by folder | happy ♿ | root → DCIM → Camera, breadcrumb back | counts, covers, plain grid, viewer stays in folder | e2e (desktop + phone) | ✅ |
| F04-E2 | | negative: outside the library | /folders/etc, `..` | "isn't in your libraries" / 404 | e2e, go | ✅ |
| F04-E3 | | edge: Synology / emoji folders | — | @eaDir hidden, "Vacances été 🏖️" listed | e2e | ✅ |
| F05-H1 | favourite | happy ♿ | heart, F key, Favourites page, unfavourite | list updates, viewer steps through favourites | e2e | ✅ |
| F05-E1 | | edge: double click on the heart | dblclick | button and server agree | e2e | ✅ |
| F05-E3 | | edge: F while the next photo loads | slow network | the previous photo is never changed | e2e | ✅ (BUG-005) |
| F05-N1 | | negative: someone else's photo | PATCH another user's id | unchanged | go | ✅ |
| F05-A | | albums | — | — | — | ⏭ albums not built |
| F06 | share an album | all | — | — | — | ⏭ sharing not built |
| F07-H2 | admin adds a folder | happy ♿ | add folder, Save | scanned, count +1 | e2e | ✅ |
| F07-E4 | | edge: remove a folder | remove, confirm | asks first, its photos go to trash | e2e | ✅ |
| F07-E5 | | edge: Scan now | button | scan runs | e2e | ✅ |
| F07-E8 | | edge: settings changed mid-scan | PATCH during a scan | one more pass queued | go | ✅ |
| F07-N4 | | negative: bad folder | /no/such/folder | error names it, nothing saved | e2e, go | ✅ |
| F07-N5 | | negative: bad pattern | `**/{broken` | error, nothing saved (not even valid fields) | e2e, go | ✅ |
| F07-N6 | | negative: not an admin | other account | 403 on libraries and scan | go | ✅ |
| F07-H3 | indexing page | happy ♿ | /admin/jobs | progress, corrupt file listed | e2e | ✅ |
| F08-H1 | archive | happy ♿ | Archive in viewer, Archive page, Unarchive | moves on, leaves timeline, comes back | e2e | ✅ |
| X-1 | all | memory on a 1 GB Pi 3 | 10k phone photos incl. HEIC | RSS < 350 MB | manual | 🔲 needs a Pi |
| X-2 | all | Safari / iPhone | the phone flows in Safari | same as Chromium | manual | 🔲 only Chromium is installed here |
| X-3 | all | screen reader pass | VoiceOver / TalkBack through F01-F05 | usable | manual | 🔲 axe covers rules, not the experience |

## Totals

- Automated: 36 Playwright tests in 9 files (35 run, 1 touch-only test skipped on desktop) and 9 Go test functions (several cover many cases each, e.g. TestFlow, TestManage). All pass; 3 full Playwright runs in a row.
- Manual: 4 open (HEIC, Pi memory, Safari, screen reader).
- Not run: albums and sharing (not built).
