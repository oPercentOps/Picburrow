# Brand: Picburrow

## Name

**Picburrow**: your pictures, burrowed safely at home. One invented compound, spelled right after hearing it once, warm rather than technical, and it doesn't fight the angle ("the folders you already have, kept at home, on a small board").

Chosen by the owner on 2026-11-13, after Homeroll's checks (below) showed two App Store apps already using that name. Picked from a second round of 12 candidates, the only one with no App Store app and all three checked domains free:

| name | App Store | .com | .app | .org |
| --- | --- | --- | --- | --- |
| **Picburrow** | no app | **free** | **free** | **free** |
| Nestroll | no app | taken | free | free |
| Snaploft | no app | taken | free | free |
| Pictureloft | no app | taken | free | free |
| Foldersnap | no app | taken | free | free |
| Snapcellar | no app | taken | free | free |
| Albumhut | lookup failed | taken | free | free |
| Roomroll | no app | taken | taken | free |
| Hearthroll | no app | taken | taken | free |
| Ferrotype | 1 app | taken | free | free |
| Homefolio | 2 apps | taken | taken | taken |
| Tinyframe | 3 apps | taken | taken | free |

Not confusable with the apps it was modelled on: no shared sound, letters or meaning with Immich or PhotoPrism.

### Checks for Picburrow

Run 2026-11-13 where a public, official lookup was reachable; the rest are **to run**. Screening, not clearance: a trademark lawyer should search before money is spent on the name, and **register picburrow.com / .app / .org soon** if you want them, since "free" today is not a reservation.

| check | how | result (2026-11-13) |
| --- | --- | --- |
| App Store | Apple's iTunes Search API, exact-name hits | none |
| picburrow.com | RDAP (Verisign) | not registered |
| picburrow.app | RDAP (Google Registry) | not registered |
| picburrow.org | RDAP (Public Interest Registry) | not registered |
| US trademark, classes 9 and 42 | tmsearch.uspto.gov | **to run in a browser** |
| EU trademark | tmdn.org/tmview | **to run in a browser** (scripted requests are rejected; not worked around) |
| global | branddb.wipo.int | to run in a browser |
| Canada | ised-isde.canada.ca | to run |
| Google Play, GitHub org, social handles, the web | — | to run |

### Why not Homeroll (first choice, dropped)

App Store already had "Homeroll: Home Inventory" and "Homeroll: Shared Bill Calendar"; homeroll.com (since 2008) and homeroll.app (2026-07) were registered. Runners-up then: Picnook (no app, .app/.org free, .com taken) and Contactsheet (two photo apps, all domains taken). "Raspicture" was the repository name until 2026-11-13 (now `Picburrow`); it starts with "Rasp", next to the Raspberry Pi trademark.

## Palette

Burnt orange: warm, "home", and deliberately outside the blue/purple family of the apps Picburrow was modelled on, and away from raspberry red (Raspberry Pi's trade dress).

| role | light | dark | viewer |
| --- | --- | --- | --- |
| accent | `#b4470f` | `#ff9a5c` | `#ff9a5c` |
| on-accent | `#ffffff` | `#1a0d05` | — |
| accent-tint | `#fdeee6` | `#2b1a10` | — |

Everything else (backgrounds, text, borders, danger, success, warning) is unchanged from `replica/design/tokens.json`. `python3 replica/design/contrast.py replica/design/tokens.json`: **35 pairs, 0 failing AA** (accent on white 5.46:1; dark accent on background 9.09:1). The Playwright suite's axe scans pass on every screen with the new colours.

## Logo brief

- **Idea**: a photo frame tucked safely inside a round burrow entrance: kept at home, sheltered. The current icon (`web/static/icon.svg`: white frame, hill and sun on burnt orange) is a placeholder of the same idea. A designer can make the burrow explicit: an arched doorway around the frame, or the frame resting in a rounded hollow.
- **Mark type**: symbol + wordmark ("Picburrow" in a humanist sans), symbol alone for the app icon and favicon.
- **Must work** at 16 px (favicon: the frame and hill only) and as a 1024 px icon.
- **Deliverables**: SVG mark and wordmark; app icon 1024×1024, no transparency (iOS); favicon set (16, 32, 180 apple-touch, 192, 512 maskable with the mark inside the safe zone); social image 1200×630.
- **Must not resemble** Immich's or PhotoPrism's marks: no petal/flower shapes, no prism or triangle, no blue-purple pairs. Put their logos next to the drafts and check before choosing.

## Voice: plain, warm, honest

| | means | doesn't mean |
| --- | --- | --- |
| **plain** | short words, one idea per sentence, no jargon on screen | curt or robotic |
| **warm** | talks like a helpful family member who set this up | cute, exclamation marks, emoji |
| **honest** | says what happened and what it can't do | apologetic or alarmed |

| do | don't |
| --- | --- |
| "It only reads them: nothing gets moved, renamed or changed." | "Non-destructive read-only indexing pipeline." |
| "This file can't be previewed here." + Download | "Unsupported media type (422)." |
| "This link no longer works. Ask them for a new one." | "Oops! Something went wrong 😕" |
| "Still making previews. Photos appear as they're ready." | "Processing… please wait" with no end |
| "Forget them now? No file is deleted: they are already gone from disk." | "Are you sure? This cannot be undone!!" |

### The most-seen strings, in this voice

| where | string |
| --- | --- |
| first run (title) | Welcome to Picburrow |
| first run (intro) | Make your account, then tell Picburrow where your phone photos land. It only reads them: nothing gets moved, renamed or changed. |
| first run (button) | Show my photos |
| empty timeline | No photos yet. They appear here as soon as the first scan finds them. If nothing shows up, check the folder under Libraries. |
| indexing banner | Still making previews. Photos appear as they're ready. |
| server error | Something went wrong on the server. Try again in a moment; if it keeps happening, the server log says why. |
| no favourites | No favourites yet. Tap the heart on any photo to keep it here. |
| share dialog | Anyone with a link can see the photos in this album, without an account. Nothing else in your library. |
| link gone | This link no longer works. It may have expired, or the owner turned it off. Ask them for a new one. |
| trash | Photos whose file is no longer on disk. Put a file back and the next scan restores it, albums and favourites included. |

## Identity in the code

- App name in the UI, page titles, manifest and README: **Picburrow**.
- Binary `picburrow` (`make pi` → `picburrow-arm64`), default data dir `/var/lib/picburrow`, database `picburrow.db`, settings `PICBURROW_DATA` / `PICBURROW_LISTEN`.
- The repository was renamed from `RaspictureGalleryNas` to `oPercentOps/Picburrow` on 2026-11-13; links, the Go module path (`github.com/oPercentOps/Picburrow/server`), the package homepage and the app's source link follow it.

## Sweep

`python3 sweep.py . --config replica/brand.json` → **Clean** (names Immich, PhotoPrism, FUTO; domains immich.app, photoprism.app, futo.org). The originals' brand colours are not in `brand.json` because they weren't verified from a primary source here.
