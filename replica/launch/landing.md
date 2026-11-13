# Landing page

Built at `docs/index.html` (GitHub Pages can serve `docs/` from `main` with no build step). Angle A from `fixes.md`: the low-maintenance one, with B ("your folders stay as they are") as the supporting line. Voice: plain, warm, honest (`brand.md`).

Rules kept: no testimonials, user counts, stars or "trusted by" rows (there are no users yet); no reviewer quotes; neither original's name appears anywhere on the page (`sweep.py` checks `docs/`).

## 1. Hero

- **Headline**: Your phone photos, as a timeline, on a 1 GB Raspberry Pi.
- **Under it**: Picburrow is one small program that turns the photo folders you already have into a timeline you can scroll on your phone. No Docker, no database server. It only reads your photos.
- **Button**: Install on your Pi → `#install`. Secondary link: Source code.
- **Image**: the real timeline, desktop and phone (`docs/img/timeline.webp`, `timeline-phone.webp`, from the Playwright run).

## 2. The problem (hate themes H1 and H2, paraphrased)

- **Too many moving parts.** A photo gallery at home usually means containers, a database server and a cache to install, update and repair, just to look at pictures.
- **Too heavy for a small board.** Those setups want several gigabytes of memory, and constant writes wear out a Pi's SD card.

## 3. How it works (the core flow: install → setup → browse)

1. **Install.** One command on 64-bit Raspberry Pi OS. It brings what it needs and starts by itself.
2. **Point it at your photos.** Open the Pi's address in a browser, make your account and pick the folder your phone backups land in.
3. **Scroll.** Your photos appear by day, newest first. Add Picburrow to your phone's home screen and it opens like an app.

## 4. Features (fixes first; each row → `fixes.md` plan number)

| feature | copy | fix |
| --- | --- | --- |
| One program, one file of data | No containers, no database server, no cache: one program and a SQLite file. | #1 (H1, U2) |
| One-command install | `sudo apt install ./picburrow_0.1.0_arm64.deb` sets up the service, a system user and the image tools. | #2 (H1) |
| Fits in 1 GB | About 20 MB for the program; 100–140 MB while making previews in our tests. Previews are made one at a time so a big photo can't starve the board. | #3 (H2) |
| Looks after your SD card | Warns you if its data sits on the SD card, with the one-line fix. | #4 (H2) |
| Safe upgrades | Before an upgrade changes its database, it saves a copy and keeps the last three. | #5 (H3) |
| The whole family, no tier | Give each person an account and their own folder. Nothing is behind a paid plan. | #6 (H4, M1) |
| Full camera details | Aperture, shutter speed, ISO, focal length and flash in the photo's details. | #7 (M2) |
| Rate many at once | Pick photos and give them stars in one go. | #8 (M3) |
| Your folders, untouched | It reads your folders as they are. Nothing is moved, renamed or rewritten. | U1, M4 |

Then the price of entry, one line each: timeline with a date scrubber; full-screen viewer with swipe; live photos; folders; albums; share links with a password and an end date; search by name, date and type; favourites, archive and trash; works on phone and desktop, light and dark.

## 5. Pricing

Free. All of it. Licensed under the GNU AGPL v3: you can use, change and share it. No paid plan, no account with us, and it never calls home. (See `pricing.md`.)

## 6. FAQ (the objections people raised in the research, answered honestly)

- **Will it run on my Pi 3?** It is built for 64-bit Raspberry Pi OS on a Pi 3, 4 or 5 with 1 GB of memory, and measured well under that on a 64-bit test machine. Testing on real Pi boards is still in progress; please report what you see.
- **Does it change my photos?** No. It only reads them. Its own data (previews, albums, favourites) lives in a separate folder.
- **How do photos get from my phone to the Pi?** With the sync app you already use (Syncthing, a NAS app, anything that copies files into a folder). Picburrow shows whatever lands there.
- **Can I move from another photo app?** If that app keeps your originals in normal folders, point Picburrow at them. Albums and faces from other apps are not imported.
- **Does it recognise faces or search by "beach"?** No, on purpose. Those need several gigabytes of memory, which is what Picburrow exists to avoid.
- **Can I see my photos away from home?** Yes, through a VPN such as WireGuard or Tailscale, or behind a web server that adds HTTPS. Picburrow has no HTTPS of its own, so don't open its port straight to the internet.
- **iPhone photos (HEIC) and live photos?** Yes: HEIC previews come from libvips, and live photos play their motion clip. Tested with generated files; real iPhone libraries are welcome as bug reports.
- **What does it cost to run?** Nothing beyond your Pi and disk. There is no paid plan.

## 7. Final call to action

- **Headline**: Give your photo folders a home.
- **Button**: Download for Raspberry Pi → GitHub releases. Link: Read the source.

## Proof section

Left out. There are no users yet. Add real quotes only from real users, with their permission.

## Screenshots: to replace before announcing

The images show the test library (generated colour charts), so they are honest but not inviting. Before the Show HN / r/selfhosted posts, replace them with screenshots of a library of your own photos, without people who haven't agreed to it, with the same file names in `docs/img/`.
