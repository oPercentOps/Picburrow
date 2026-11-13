# Brand: Homeroll

## Name

**Homeroll**: *home* + *camera roll*. Your camera roll, at home. Two plain words, said once and spelled right, and it says where the thing runs.

Chosen by the owner from 5 shortlisted out of 20 candidates (2026-11-13):

| style | candidates |
| --- | --- |
| compound | **Homeroll**, Tinyroll, Littleroll, Snapshelf, Snapnook, **Picnook**, Homeframe |
| descriptive | Foldr Photos, **Contactsheet**, Fotobox |
| metaphor | Lightbox, Attic, Shoebox, Grain, Negatives, Folio, Keepsake |
| invented | Pictarium |
| existing | **Raspicture** (the repo's name) |

Cut: Lightbox and Darkroom-style names are taken by photo apps; Attic, Grain, Folio, Keepsake are too common to own; Fotobox collides with photo booths; Raspicture starts with "Rasp", next to the Raspberry Pi trademark (the Raspberry Pi Foundation polices "Raspberry Pi" and "Pi" in product names), so it stays the repo name only.

Not confusable with the apps it was modelled on: no shared sound, letters or meaning with Immich or PhotoPrism.

### Checks

Run 2026-11-13 from the build environment where a public, official lookup was reachable; the rest are marked **to run** with where to run them. These are screening checks, not clearance: a trademark lawyer should search before money is spent on the name.

| check | how | result (2026-11-13) |
| --- | --- | --- |
| **App Store** | Apple's iTunes Search API, term "homeroll" | ⚠️ **taken twice**: "Homeroll: Home Inventory" (Lifestyle) and "Homeroll: Shared Bill Calendar" (Finance). Different categories, but the same name in the same store |
| homeroll.com | RDAP (Verisign) | registered since 2008-03-16 |
| homeroll.app | RDAP (Google Registry) | registered 2026-07-28 (possibly by one of the apps above) |
| homeroll.net / .dev / .org / .photos | RDAP | not registered (HTTP 404) |
| US trademark, classes 9 and 42 | tmsearch.uspto.gov | **to run in a browser** (the search service can't be queried from here) |
| EU trademark | tmdn.org/tmview | **to run in a browser** (its firewall rejects scripted requests, and that wasn't worked around) |
| global | branddb.wipo.int | to run in a browser |
| Canada | ised-isde.canada.ca | to run |
| GitHub org "homeroll" | api.github.com | unknown: the request was refused from this environment |
| Google Play, handles, the web | — | to run |

**Runners-up, same checks:**

| name | App Store | .com | .app | .org |
| --- | --- | --- | --- | --- |
| **Picnook** | no app with the name | registered 2013-05-30 | **not registered** | **not registered** |
| Contactsheet | 2 photo apps ("Contact Sheet Generator", "Contact Sheet: AI Photo Review") | registered 1998 | registered 2025 | registered 2003 |

**What this means:** Homeroll already names two App Store apps and its .com/.app are taken. That is not a legal conflict by itself (different categories), but it costs findability and invites confusion. Picnook is the cleaner of the shortlist on everything checked here. Switching is the owner's call; the rename is a sweep away (`sweep.py` + `brand.json`).

## Palette

Burnt orange: warm, "home", and deliberately outside the blue/purple family of the apps Homeroll was modelled on, and away from raspberry red (Raspberry Pi's trade dress).

| role | light | dark | viewer |
| --- | --- | --- | --- |
| accent | `#b4470f` | `#ff9a5c` | `#ff9a5c` |
| on-accent | `#ffffff` | `#1a0d05` | — |
| accent-tint | `#fdeee6` | `#2b1a10` | — |

Everything else (backgrounds, text, borders, danger, success, warning) is unchanged from `replica/design/tokens.json`. `python3 replica/design/contrast.py replica/design/tokens.json`: **35 pairs, 0 failing AA** (accent on white 5.46:1; dark accent on background 9.09:1). The Playwright suite's axe scans pass on every screen with the new colours.

## Logo brief

- **Idea**: a photo frame you keep at home. The current icon (`web/static/icon.svg`: white frame, hill and sun on burnt orange) is a placeholder of the same idea. A designer may make it warmer: a frame with a roof line, or a frame resting on a shelf.
- **Mark type**: symbol + wordmark ("Homeroll" in a humanist sans), symbol alone for the app icon and favicon.
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
| first run (title) | Welcome to Homeroll |
| first run (intro) | Make your account, then tell Homeroll where your phone photos land. It only reads them: nothing gets moved, renamed or changed. |
| first run (button) | Show my photos |
| empty timeline | No photos yet. They appear here as soon as the first scan finds them. If nothing shows up, check the folder under Libraries. |
| indexing banner | Still making previews. Photos appear as they're ready. |
| server error | Something went wrong on the server. Try again in a moment; if it keeps happening, the server log says why. |
| no favourites | No favourites yet. Tap the heart on any photo to keep it here. |
| share dialog | Anyone with a link can see the photos in this album, without an account. Nothing else in your library. |
| link gone | This link no longer works. It may have expired, or the owner turned it off. Ask them for a new one. |
| trash | Photos whose file is no longer on disk. Put a file back and the next scan restores it, albums and favourites included. |

## Identity in the code

- App name in the UI, page titles, manifest and README: **Homeroll**.
- Binary `homeroll` (`make pi` → `homeroll-arm64`), default data dir `/var/lib/homeroll`, database `homeroll.db`, settings `HOMEROLL_DATA` / `HOMEROLL_LISTEN`.
- The Go module path keeps the repository's name (`RaspictureGalleryNas`) until the repository itself is renamed.

## Sweep

`python3 sweep.py . --config replica/brand.json` → **Clean** (names Immich, PhotoPrism, FUTO; domains immich.app, photoprism.app, futo.org). The originals' brand colours are not in `brand.json` because they weren't verified from a primary source here.
