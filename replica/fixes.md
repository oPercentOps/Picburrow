# What users of Immich and PhotoPrism say, and what Picburrow does about it

Second pass, 2026-11-14. The first pass (2026-11-13) had one source; this one has three.

## The sample, honestly

- **1,198 rows from three sources**, every row linked, in `replica/reviews.csv` (the ranking tool counts 1,197: it drops one as a duplicate).
  - **Hacker News: 383 comments** (196 mention Immich, 187 PhotoPrism), 2023-07 to 2026-11, from its official Algolia search API. Unchanged from the first pass.
  - **App Store: 221 reviews of the Immich iPhone app**, 2022-12 to 2026-11, from Apple's public customer-reviews feed for eight English-language stores (US 139, Canada 28, UK 23, India 21, Australia 8, Ireland 1, Singapore 1, New Zealand 0). Average 4.1 stars: 139 five-star, 27 four, 17 three, 13 two, 25 one. The first pass got an empty feed; this time it answered.
  - **Lemmy: 594 comments** (300 naming Immich, 294 PhotoPrism), 2023-09 to 2026-11, from lemmy.world's public search API, newest first. Links go to each comment's home server.
- **Read by hand:** all 55 App Store reviews with 1 to 3 stars, all 27 with 4 stars, and the Lemmy sentences about hardware, install, upgrades, folders, video, family and named features (about 160 sentences). The other Lemmy comments and the 139 five-star reviews were only ranked by keyword.
- **What the sources are about.** The App Store reviews are about the *phone app*, which Picburrow does not have. Most Lemmy and Hacker News rows are people listing what they self-host; complaints are a minority. People mostly like Immich.
- **Not collected:** Reddit (its API needs an account), Google Play (no official feed), G2/Capterra/Trustpilot (not used for self-hosted apps), both projects' GitHub boards and changelogs (outside this session's GitHub access). So "the original already fixed this" is checked only against what commenters say.
- `replica/feedback.md` is the keyword ranking. It is still noisy, so the lists below come from reading. Counts are of the reviews I read and cite unless a line says "keyword count".

## 1. What they hate

**H1. Docker, Postgres and Redis to run a photo gallery** (3 sources; 8 cited)
- "Immich wants a Postgres and a redis instance." https://news.ycombinator.com/item?id=50113737
- "Using Immich without Docker also seems complicated." https://news.ycombinator.com/item?id=50114669
- "Many projects, like Immich, only provide docs for docker images/compose deployment." https://news.ycombinator.com/item?id=50112473
- "I wanted to like Photoprism because unlike Ente and Immich, it supports SQLite databases and doesn't require postgres (I want to keep home lab maintenance to a minimum) but the UI was difficult to like" https://news.ycombinator.com/item?id=44428119
- "Wanted to try immich but the install was overly complicated." https://lemmy.world/comment/15599070
- "I have to stick with Photoprism for now as it offers a install without docker and it does the job for me." https://lemmy.ml/comment/7071573
- "with no nice deb packages to ease installation: Paperless-ngx and Photoprism" https://lemmy.world/comment/15123723
- "love the idea but would need to all in one installer still don't know how to use it" (3 stars) https://apps.apple.com/au/app/immich/id1613945652?see-all=reviews#11610914924
- Counter-evidence: "I don't feel Immich is hard to setup or maintain" https://lemmy.world/comment/25925012 and "It’s fast, easy to setup using docker compose/portainer stacks." (4 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#11588945927. People at home with Docker don't have this problem.

**H2. Heavy on small machines** (2 sources; 8 cited)
- "Just the memory usage is still a bit high." (about Immich) https://news.ycombinator.com/item?id=36681895
- "For me Immich idles at 0.1% on consumer CPU consuming less than 3GB of RAM" (said as a good number, which is still 3x a 1 GB Pi) https://news.ycombinator.com/item?id=50217316
- "hosts my photos with photoprism (by far the most resource intensive service on this server)" https://news.ycombinator.com/item?id=42513680
- "a mini pc with an ssd runs circles around a raspberry pi running off a microsd card" https://news.ycombinator.com/item?id=50217034
- "I tried immich first on a resource constrained system and it was a real pig naturally." https://slrpnk.net/comment/13593898
- "i want something lightweight, and i specifically mentioned immich isn't what i want in the OP." https://lemmy.today/comment/26722628
- "Immich runs great on my server with an i5-7500 CPU and uses about 800MB of RAM, with around 3-4GB while actively running the ML models on new uploads" (said as praise) https://lemmy.blahaj.zone/comment/22209314
- Counter-evidence, kept on purpose: "I have about 120.000 pictures on my Raspberry Pi and for the most part, the experience is flued" (PhotoPrism) https://news.ycombinator.com/item?id=39571424. A Pi with enough RAM already works for some PhotoPrism users.

**H3. Upgrades that break, and a phone app that must match the server** (3 sources; 7 cited. No longer thin)
- "Having used both I can say that PhotoPrism runs MUCH more stable and surprisingly smooth. Immich constantly breaks." https://news.ycombinator.com/item?id=36681133
- "every time I go to check on it they have released another breaking update (or at least released several new updates and the app refuses to connect anymore) and I am tired of it." https://piefed.ca/comment/7015927
- "My issue with immich is that new releases kept breaking things." https://lemmy.world/comment/17374694
- "But lately I'm leaning towards photoprism because of updates breaking immich sometimes apparently" https://lemmy.world/comment/18135681
- "How do people update server when away from home network and the app won’t sync with the server at all" (2 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#14621409470
- "Error..'Server out of date please update to the latest minor version' How to do this, backup is stopped, Help." (3 stars) https://apps.apple.com/in/app/immich/id1613945652?see-all=reviews#11761466627
- "Backup doesn’t work after app update" (1 star) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#13283723920

**H5. The phone app's backup can't be trusted** (1 source, but the biggest complaint in it: 27 of the 55 reviews with 1 to 3 stars mention backup, sync or upload; keyword count)
- "Backup stuck at ~40%. Many users reporting this problem in forums since mid-2025." (2 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#13615094258
- "the iOS syncing is extremely unreliable, particularly the background upload" (2 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#11334372139
- "325 of more than 21k have been uploaded in a more than 6 month period." (4 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#14706922813
- "the lack of background upload is a dealbreaker" (1 star) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#13503691215
- Picburrow has no backup at all, so this is not ours to fix. It matters because it is why people fall back on a sync app and a folder (U1).

**H6. The app fails where the browser works** (1 source; 7 low-star reviews mention signing in, keyword count; 3 cited)
- "Web gui works, app login doesn’t even click let alone log in" (1 star) https://apps.apple.com/ca/app/immich/id1613945652?see-all=reviews#11025912049
- "The web ui works great I can scroll to any point in time and play a video or load a photo but the app has terrible performance issues." (3 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#13840167614
- "Using the web UI as a mobile app for now because it supports tag-based searching. Works surprisingly well on mobile devices." (4 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#13520380732

**H4. Paying, or not knowing whether you have to** (3 sources, few comments: thin)
- "Only the mobile app is free, the desktop app costs $25. Or $100 for access for multiple people." (1 star; the reviewer's reading, not checked against Immich's licence page) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#14625550665
- "þe features PhotoPrism has locked behind a paywall which Immich provides for free" https://piefed.zip/comment/5791277
- "I used Photoprism for over a year, even paid the \"server license\"" https://news.ycombinator.com/item?id=43647085

**H7. Fear that the tool itself loses or deletes photos** (thin: 2 reviews, 1 source)
- "if you aren’t extremely cautious, you will lose photos" (1 star) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#12083626303
- "Why would this app think it’s important enough to have permission to delete photos from your device instead of just the app itself?" (1 star) https://apps.apple.com/gb/app/immich/id1613945652?see-all=reviews#11576549305

## 2. What is missing

**M5. New photos that show up by themselves** (1 source, 4 comments, all PhotoPrism users: thin by the one-source rule). People who sync phones into a folder end up scripting the re-index.
- "I run my photoprism in docker and I always had to manually trigger the index after changes to the folder." https://lemmy.world/comment/12564717
- "A cron job that runs the `photoprism import` command every few minutes." https://midwest.social/comment/12517455
- "I simply sync the Photos from my smartphone to a folder and make PhotoPrism scan it on a regular basis using ofelia." https://lemmy.sdf.org/comment/13463630
- "I use a folder sync app, to upload all my pictures from my phone to a directory on my server, which PhotoPrism imports from every 15 mins(Might be a bit more or less)." https://endlesstalk.org/comment/6065284
- **Picburrow had this gap until 2026-11-14**: it scanned at start-up, at 03:00 and on Scan now, so a photo synced at noon appeared the next morning. Fixed by fix 9.

**M1. A library the whole family sees** (3 sources, 1 comment each: thin)
- "It only lets 2 users to share and view the photos in the same timeline, while the third user can only view photos from one another user in timeline." (1 star) https://apps.apple.com/in/app/immich/id1613945652?see-all=reviews#13851507950
- "My family photos need to be shared as external libraries while our phone photos cannot be shared unless you share accounts." https://lemmy.today/comment/20642954
- "I've still gotta get around to figuring how I can provide access to family members." https://news.ycombinator.com/item?id=50216572

**M6. Memories / on this day** (1 source, one thread: thin). The original has it; Picburrow does not.
- "The lack of memories would be a deal breaker for them to use immich." https://lemmy.world/comment/26260398
- "immich absolutely has a memories with pictures on that day in previous years" https://sh.itjust.works/comment/28088994

**M10. HDR photos and video shown as HDR** (1 source, 3 reviews: thin)
- "cannot replace the Photos app until it can show pictures and video in glorious HDR" (4 stars) https://apps.apple.com/ca/app/immich/id1613945652?see-all=reviews#11960256618
- "On any HDR capable iOS device, colors are washed out and look wrong." (3 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#10524974145

**M7. Rotate a photo** (thin, one comment)
- "Neither solution offers any convenience editing such as simply rotating a photo." https://lemmy.world/comment/18094011

**M8. Tags and sidecars written by other tools** (thin, one comment)
- "Photoprism recognises the keywords and sidecar data from digikam, which lets me search and access the images from anywhere." https://lemmy.blahaj.zone/comment/10090255

**M9. Delete a bad photo from the gallery** (thin, one comment; it argues against "read-only")
- "After my wife complained _again_ about not being able to delete photos in PhotoPrism, I finally bit þe bullet and migrated to Immich." https://piefed.zip/comment/5791277

**From the first pass, all done since:** richer capture details (M2, https://news.ycombinator.com/item?id=49540322), batch editing (M3, https://news.ycombinator.com/item?id=45169861), metadata kept when photos move (M4, https://news.ycombinator.com/item?id=50217698).

**One review each, not counted as themes:** search results sorted by date, AirPlay, animated GIFs, video trim.

## 3. What is unsolved

**U1. "Just show me the folders I already have", filled by whatever sync tool works** (2 sources; 6 cited)
- "Photoprism is the best one I’ve found that will just take and index your folders as they are." https://news.ycombinator.com/item?id=49539107
- "I use a sync app (\"FolderSync\") to upload-then-delete to my server over Tailscale, then Immich picks them up." https://news.ycombinator.com/item?id=50436904
- "Should I import fresh into Immich or setup my existing photoprism as an external library?" https://news.ycombinator.com/item?id=48773538
- "A lot of photographers have organised folders by date/event that span years/decades, so it’s not practical to create these manually with albums." https://lemmy.world/comment/10606483
- "Same story for old family photos (Photoprism, indexing a directory tree on a Synology NAS)" https://lemmy.nz/comment/9101488
- "Immich will also index your folders as they are." https://news.ycombinator.com/item?id=49539299 (so the original covers this now; the difference is what it takes to run it)

**U2. Low-maintenance home labs.** The people in H1 and H3 want fewer moving parts, not more features.
- "I want to keep home lab maintenance to a minimum" (in the H1 quote above) https://news.ycombinator.com/item?id=44428119
- "Lts are super important to me as I don’t want to spend every weekend reconfiguring services for my family." https://lemmy.world/comment/17374694

**U3. Getting the family to accept it** (2 sources; 3 cited: thin). What decides it is that it just works on their phone.
- "This is essential if I’m going to get buy-in from the family." (2 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#13577306658
- "I really want to move my family to self hosted but I can’t sell my wife on this." (3 stars) https://apps.apple.com/us/app/immich/id1613945652?see-all=reviews#13840167614
- "The lack of memories would be a deal breaker for them to use immich." https://lemmy.world/comment/26260398

**U4. Being the one responsible for the only copy** (1 source; 2 cited: thin)
- "I was a bit nervous being in charge of uptime/backups of my families photos" https://feddit.uk/comment/27946887
- "How is everyone backing up Immich off site without breaking the bank?" https://startrek.website/comment/24986396

**Evidence against Picburrow's own approach** (1 source, 3 comments). Picburrow leaves phone backup to a separate sync tool. Some people left PhotoPrism for exactly that reason:
- "having to set it up for each user on boþ server and mobile is tedious, and þe whole Rube Goldberg system is harder to keep track of" https://piefed.zip/comment/5791277
- "I did consider just one thing for syncing, one for the gallery, but that gets already too complicated for other people." https://discuss.tchncs.de/comment/4736575
- "it worked ok for me, but I wasn't going to set it up on my partner's phone, for example." https://startrek.website/comment/19706246

This is the biggest risk to the product: the all-in-one app is what people want, even though its backup is what they complain about most (H5).

## 4. Fix plan

### Round 1 (first pass): all done

| # | what | evidence | status |
| --- | --- | --- | --- |
| 1 | No Docker, no Postgres, no Redis: one binary + SQLite | H1, U2 | done (architecture) |
| 2 | One-command install on Raspberry Pi OS (`.deb` + systemd) | H1 | done |
| 3 | Runs in 1 GB, measured on the build box | H2 | done; the real-Pi test is still open |
| 4 | Warning when the database or thumbnails sit on the SD card | H2 | done |
| 5 | Automatic database backup before every upgrade | H3 | done |
| 6 | Family accounts free, each with their own folder, or sharing one library | H4, M1 | done (sharing since 0.1.6) |
| 7 | Full capture details in the info panel | M2 | done |
| 8 | Batch rating from the selection bar | M3 | done |
| — | Folders indexed as they are, files never moved, rewritten or deleted | U1, M4, H7 | done (read-only by design) |
| — | No phone app to install, sign in to or keep in step with the server | H3, H6 | done (it is a web page; add to home screen) |

### Round 2: what to build next

Ranked by evidence times cheapness.

| # | what | evidence | size | skill |
| --- | --- | --- | --- | --- |
| 9 | **New photos appear within minutes**: every 15 minutes, a scan if any folder's modified time changed; the 03:00 full scan stays | M5, U1 | S | **done** 2026-11-14 (`TestFoldersChanged`) |
| 10 | **A tested guide for getting phone photos into the folder**, iPhone and Android, one sync tool each, linked from the empty timeline after setup | the evidence against our approach, H5 | S (writing) + phones to test on | `/replica-launch` docs; can't be tested from a build box |
| 11 | **On this day**: a button on the timeline when earlier years have photos from today, opening a page of them | M6, U3 (thin) | S | **done** 2026-11-14 (`TestOnThisDay`) |
| 12 | **Share a library with an account that already exists**, and let a person have their own folder plus a shared one | M1 (thin, 3 sources) | S | **done** 2026-11-14 (`TestShareWithExistingAccount`, e2e P-1) |
| 13 | **Rotate**, stored in the database and applied to previews (files stay untouched) | M7 (thin) | S | `/replica-build` |
| 14 | **Read XMP sidecars**: keywords, ratings and descriptions written by digiKam and others, searchable | M8 (thin) | M | `/replica-build` |
| 15 | **The real-hardware pass**: 1 GB Pi 3, real iPhone library, Safari | the H2 claim rests on it | — | `/replica-test` |

Not building, and why:
- **Phone backup.** The most wanted thing and the most complained about (H5). On iPhone the original's team has not made it reliable in years of reviews; a web page cannot upload in the background at all. Fix 10 is the honest answer.
- **Delete from the gallery** (M9). One comment, and it breaks "your files are never touched". Archive hides a photo today. Revisit if it comes up again.
- **HDR** (M10). Three reviews, one source, and HDR previews mean heavier decoding on the Pi.
- **Faces and smart search.** They need the RAM this product exists to avoid.

Pricing: Picburrow is free and open source. H4 is a line for `/replica-launch` ("free for the whole family, nothing to unlock"), not a billing change.

## 5. The angle

**A. The low-maintenance one (recommended, unchanged)**
For home-lab owners who don't want to run Docker, Postgres and Redis just to see their photos, Picburrow is one small program that turns the photo folders you already have into a phone-friendly timeline, on a 1 GB Raspberry Pi.
Evidence: H1 (3 sources), H2 (2 sources), H3 (3 sources), U2. This was one source yesterday; it now holds across three.

**B. The "your folders, untouched" one**
For families whose phone backups already land on the NAS, Picburrow is Google-Photos-style browsing on top of those folders, without importing, moving, rewriting or deleting a single file, so the backup you already have still covers everything.
Evidence: U1 (2 sources), H7 and U4 (thin).

**C. The "nothing to install on anyone's phone" one (new)**
For families where the photo app is the part that keeps breaking, Picburrow opens in the phone's browser, sits on the home screen, and can never be out of step with the server.
Evidence: H6 and H3, mostly App Store (221 reviews, 55 with 1 to 3 stars). Honest limit: those reviewers want backup fixed, and Picburrow doesn't back up. It only answers the viewing half, so it can't lead.

**Recommendation: A**, with B as the supporting line and one sentence from C ("no app to install or update"). Yesterday's option C, "boring on purpose", folds into A now that broken upgrades have evidence from three sources: "it backs itself up before every upgrade" is a proof point for low maintenance. The landing page should say plainly that phone backup is left to a sync app, and link fix 10's guide; hiding it would cost trust with exactly the people in the counter-evidence.

Rules for `/replica-brand` and `/replica-launch`, unchanged: don't put "Immich" or "PhotoPrism" in the name, ads or listing ("PhotoPrism" is a registered trademark); don't quote these reviewers in marketing, since they are research, not testimonials.
