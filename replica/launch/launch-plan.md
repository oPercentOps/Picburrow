# Launch plan

Picburrow is free, open-source, self-hosted software, so "launch" means: a release people can install, a page that explains it, and posts in the places self-hosters read. No app stores, no ads, no paid acquisition.

## 0. Before announcing (blockers)

| # | what | why | who |
| --- | --- | --- | --- |
| 1 | **Run it on a real Pi 3 with 1 GB** (10k phone photos; peak memory under 350 MB; service starts on boot; SD-card warning appears) | The page's main promise. It's measured on a 64-bit test machine, not on a Pi yet. | owner (needs the hardware) |
| 2 | **A real iPhone library** (HEIC + live photos) | Tested only with generated files so far | owner |
| 3 | **Tag v0.1.0 and attach the `.deb`** (arm64 + amd64, with SHA-256 sums) | The landing page and README point to the releases page, which is empty | `/replica-deploy` |
| 4 | **Turn on GitHub Pages**: Settings → Pages → Deploy from a branch → `main` / `/docs` | Publishes the landing page | owner (one click) |
| 5 | **Replace the screenshots** in `docs/img/` with your own library (same file names) | Today's show generated colour charts: honest, but not inviting | owner |
| 6 | Register picburrow.com / .app / .org, and run the trademark searches in a browser (`brand.md`) | Free today is not a reservation | owner |

Items 1 and 2 decide whether "runs on a 1 GB Pi" stays on the page as written. If the Pi 3 peak is over budget, change the headline to "Pi 4 or 5" before posting.

## 1. Waitlist / beta

No mailing list (that's another service to run and a privacy promise to keep). Instead:

- **GitHub "Watch → Custom → Releases"** is the waitlist. Say so in the README and at the end of every post.
- **A beta call** before the big posts: one post asking for **Pi 3 owners and iPhone users** to try v0.1.0 and report memory use and HEIC results. Those reports fill blockers 1 and 2 if you don't have the hardware.

## 2. Analytics and error tracking

Deliberately none in the product: "it never calls home" is part of the pitch (checked: the server makes no outbound requests). Measure with what GitHub already gives you:

- release download counts (per asset) and repository traffic (Insights → Traffic);
- issues opened with the bug template added in this change (`.github/ISSUE_TEMPLATE/bug.md`), which asks for the Pi model, OS, `journalctl -u picburrow` output and the Indexing page's "Files without a preview".

The landing page on GitHub Pages has no analytics script either.

## 3. Where the people with the problem talk

From the research (`fixes.md`): Hacker News threads about self-hosted photo libraries and home-lab stacks, where the Docker/Postgres and memory complaints came from. Not researched here, but the obvious places for the same audience: r/selfhosted, r/raspberry_pi, r/homelab, the Raspberry Pi forums' "Networking and servers" board. Read each community's self-promotion rules before posting; most allow one post per release if you answer comments.

## 4. The posts (lead with the fix, not the clone)

**Show HN** (title 77/80 characters):

> Show HN: Picburrow – a photo timeline for your folders on a 1 GB Raspberry Pi

> I wanted to scroll my phone backups on a Pi without running containers, a database server and a cache. Picburrow is one Go program with SQLite that reads the folders you already have (it never writes to them) and serves a phone-friendly timeline, viewer, albums and share links. About 20 MB on its own, 100–140 MB while making previews, one at a time. One `apt install` on 64-bit Raspberry Pi OS. No faces or smart search, on purpose: that's where the memory goes. AGPL-3.0, no paid tier, never calls home. I'd love reports from Pi 3 owners and iPhone (HEIC) users.

Post on a weekday morning US time, link the repo (not the landing page), and stay in the thread for the first few hours. Don't name other projects in the post; if commenters compare, answer plainly about what Picburrow doesn't do.

**r/selfhosted** (title): "Picburrow: a photo timeline for the folders you already have, in one small program (no Docker, runs on a 1 GB Pi)". Body: the Show HN text plus two screenshots (phone timeline, viewer) and the install command.

## 5. Listings (instead of app stores)

| where | text | limit |
| --- | --- | --- |
| GitHub "About" | Your phone photos as a timeline, from the folders you already have. One small program with SQLite: no Docker, no database server. Runs on a 1 GB Raspberry Pi. Read-only, AGPL-3.0. | 350 chars (this is 179) |
| GitHub topics | `photo-gallery` `self-hosted` `raspberry-pi` `photos` `timeline` `sqlite` `golang` `svelte` `pwa` `heic` `nas` `homelab` | 20 max, lowercase |
| GitHub website | the Pages URL (or picburrow.org once registered) | |
| awesome-selfhosted (Photo Galleries) | "Photo timeline for existing folders, designed for small single-board computers." | **Not yet eligible**: they require the first release to be more than 4 months old. If v0.1.0 ships in November 2026, submit in March 2027. Their guide asks to avoid "free", "open-source" and "self-hosted" in the description. |
| Debian package `Description` | already in `packaging/build-deb.sh` | |

**Why no App Store or Google Play listing:** there is no native app. On a phone, Picburrow is the web app served by your own Pi, added to the home screen from the browser. A store app that only wraps that web view would fall foul of Apple's minimum-functionality rule (Guideline 4.2), and it would need a server address setup flow, store accounts ($99/year for Apple) and review cycles, for no gain to users. So `listing.json` / `listing.py` were not used; the texts above were checked for the `avoid` names instead (`brand.json`).

## 6. The first 10 users, by hand

- **Two or three people you know** with a Pi and a phone full of photos: install it with them, watch where they get stuck, fix that first.
- **The beta call's first responders**: reply to every report within a day; ask for their `journalctl` output and one screenshot.
- **New threads** on HN and r/selfhosted where someone asks for a light photo gallery for a Pi or NAS: answer the question honestly, mention Picburrow only where it fits, and say what it doesn't do.
- Don't cold-message the people from the research comments: they were research, not leads.

Ask each one the same three questions: what did you expect after install that didn't happen; what did you look for and not find; would you leave it running. Write the answers into `replica/feedback.md` as a new source.
