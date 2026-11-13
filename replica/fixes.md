# What users of Immich and PhotoPrism say, and what this gallery does about it

## The sample, honestly

- **383 comments from one source**: Hacker News, via its official Algolia search API (196 mention Immich, 187 PhotoPrism), from 2023-07 to 2026-11. `replica/reviews.csv`, every row linked.
- **App Store: 0 reviews.** Apple's public customer-reviews feed returned an empty list for the Immich iOS app (id 1613945652) in the US, UK, German, French, Canadian and Australian stores.
- **Not collected:** Reddit (its API needs an account and OAuth), Google Play (no official feed), G2/Capterra/Trustpilot (not used for self-hosted apps), the projects' GitHub issues and discussions (not reachable from this session).
- **No star ratings**, so no theme is weighted by stars. `replica/feedback.md` is the raw keyword ranking; it is noisy (many comments just list someone's self-hosted stack), so the lists below are from reading the matching comments by hand.
- **Every theme is thin by the skill's rule** (one source). Read these as "people do say this", not as measured trends.

## 1. What they hate

**H1. Docker, Postgres and Redis to run a photo gallery** (the most frequent install/maintenance complaint I read; keyword theme: 91 comments, mostly noise)
- "Immich wants a Postgres and a redis instance." https://news.ycombinator.com/item?id=50113737
- "Using Immich without Docker also seems complicated." https://news.ycombinator.com/item?id=50114669
- "Many projects, like Immich, only provide docs for docker images/compose deployment." https://news.ycombinator.com/item?id=50112473
- "I wanted to like Photoprism because unlike Ente and Immich, it supports SQLite databases and doesn't require postgres (I want to keep home lab maintenance to a minimum) but the UI was difficult to like" https://news.ycombinator.com/item?id=44428119

**H2. Heavy on small machines** (memory, indexing CPU, SD cards)
- "Just the memory usage is still a bit high." (about Immich) https://news.ycombinator.com/item?id=36681895
- "For me Immich idles at 0.1% on consumer CPU consuming less than 3GB of RAM" (said as a good number, which is still 3x a 1 GB Pi) https://news.ycombinator.com/item?id=50217316
- "hosts my photos with photoprism (by far the most resource intensive service on this server)" https://news.ycombinator.com/item?id=42513680
- "The only obviously slow operations are anything heavily CPU-bound, like compiling (esphome) and photo indexing (photoprism)." https://news.ycombinator.com/item?id=48973573
- "a mini pc with an ssd runs circles around a raspberry pi running off a microsd card" https://news.ycombinator.com/item?id=50217034
- Counter-evidence, kept on purpose: "I have about 120.000 pictures on my Raspberry Pi and for the most part, the experience is flued" (PhotoPrism) https://news.ycombinator.com/item?id=39571424 and "I have a local instance running on a Raspberry Pi with about 149.000 photos inside." https://news.ycombinator.com/item?id=48937814. A Pi with enough RAM already works for some PhotoPrism users.

**H3. Upgrades that break** (thin: few comments, one from 2023)
- "Having used both I can say that PhotoPrism runs MUCH more stable and surprisingly smooth. Immich constantly breaks." https://news.ycombinator.com/item?id=36681133
- "gets new features very quickly (sometimes it's even too fast paced)" https://news.ycombinator.com/item?id=43647085

**H4. Paying for what a family needs** (thin)
- PhotoPrism's multi-user is documented under its paid plan: "Edit: From https://www.photoprism.app/plus/kb/multi-user" https://news.ycombinator.com/item?id=36684996
- "I used Photoprism for over a year, even paid the \"server license\"" https://news.ycombinator.com/item?id=43647085

## 2. What is missing

**M1. Family access that is easy to set up** (thin)
- "I've still gotta get around to figuring how I can provide access to family members." https://news.ycombinator.com/item?id=50216572

**M2. Richer capture details** (thin, one comment)
- "In Apple Photos I can tap the little \"I\" icon to see a lot of capture metadata, and in Immich I did not see an equivalent to how good Apple's one is." https://news.ycombinator.com/item?id=49540322

**M3. Batch editing of tags and metadata** (thin, one comment)
- "Photoprism is lacking batch edit for tags/metadata" https://news.ycombinator.com/item?id=45169861

**M4. Keep metadata intact when moving photos in** (thin, one comment)
- "I found that it didn't preserve metadata when I tried to reupload my photos to Immich..." https://news.ycombinator.com/item?id=50217698

## 3. What is unsolved

**U1. "Just show me the folders I already have."** People value a gallery that indexes existing folders as they are, and they compare products on exactly that:
- "Photoprism is the best one I’ve found that will just take and index your folders as they are." https://news.ycombinator.com/item?id=49539107
- "Immich will also index your folders as they are." https://news.ycombinator.com/item?id=49539299
- "I use a sync app (\"FolderSync\") to upload-then-delete to my server over Tailscale, then Immich picks them up." https://news.ycombinator.com/item?id=50436904
- "Should I import fresh into Immich or setup my existing photoprism as an external library?" https://news.ycombinator.com/item?id=48773538

**U2. Low-maintenance home labs.** The people in H1 aren't asking for more features; they want fewer moving parts ("I want to keep home lab maintenance to a minimum", above).

## 4. Fix plan

Ranked by evidence times cheapness. Already done in this gallery is marked so: those are the positioning, not new work.

| # | what | evidence | size | status / skill |
| --- | --- | --- | --- | --- |
| 1 | **No Docker, no Postgres, no Redis**: one binary + SQLite | H1, U2 | — | done (architecture) |
| 2 | **One-command install on Raspberry Pi OS**: a `.deb` with a systemd unit that pulls in libvips, exiftool and ffmpeg | H1 | S | to build: `/replica-deploy` |
| 3 | **Runs in 1 GB**, measured: 98-136 MB on a typical library, ~385 MB worst case (48 MP HEIC) | H2 | — | done; the real-Pi test is still open |
| 4 | **Warn when the database or thumbnails sit on the SD card**, on setup and the Indexing page, with a one-line fix | H2 (SD-card quote) | S | to build: `/replica-build` |
| 5 | **Automatic database backup before every upgrade**, kept next to the database, plus a documented one-file restore | H3 | S | to build: `/replica-build` (the migration framework is there) |
| 6 | **Family accounts free**, each with their own folder | H4, M1 | — | done (People page) |
| 7 | **Full capture details**: aperture, shutter speed, ISO, focal length, flash, in the info panel | M2 | S | to build: `/replica-build` (one exiftool tag list) |
| 8 | **Batch rating** from the selection bar | M3 (thin) | S | to build: `/replica-build` |
| — | Folders indexed as they are, files never moved or rewritten | U1, M4 | — | done (read-only by design) |

Faces and smart search stay out (they need the RAM this product exists to avoid). Pricing: the gallery is free and open source, so H4 is a positioning point for `/replica-launch`, not a billing change.

## 5. The angle

**A. The low-maintenance one (recommended)**
For home-lab owners who don't want to run Docker, Postgres and Redis just to see their photos, this gallery is one small program that turns the photo folders you already have into a phone-friendly timeline, on a 1 GB Raspberry Pi.
Evidence: H1 + H2 + U2, the most frequent complaints I read (one source, thin).

**B. The "your folders, untouched" one**
For families whose phone backups already land on the NAS, this gallery is Google-Photos-style browsing on top of those folders, without importing, moving or rewriting a single file.
Evidence: U1 + M4 (thin).

**C. The boring-on-purpose one**
For people tired of upgrades that break things, this gallery changes slowly and backs itself up before every upgrade.
Evidence: H3 (thin: two quotes, one from 2023). Too little to lead with.

**Recommendation: A**, with B as the supporting line ("your folders stay as they are"). It is what the product is architected around, the claims are measured, and it is honest about the counter-evidence: PhotoPrism does run on a Pi with more RAM; this is for people who want it without Docker or Postgres, on the smallest board.

Rules for `/replica-brand` and `/replica-launch`: don't put "Immich" or "PhotoPrism" in the name, ads or listing ("PhotoPrism" is a registered trademark); don't quote these reviewers in marketing, since they are research, not testimonials.
