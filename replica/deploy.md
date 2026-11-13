# Deploy checklist: Picburrow

Date: 2026-11-13  Release: **0.1.1** (commit ed8652c)  Go from user: **yes** (owner published the release and made the repository public)

Picburrow is self-hosted, free software: people run it on their own Pi. So "production" here is **a GitHub release with the `.deb` packages** plus **the landing page on GitHub Pages**. There is no hosted server, production database, payments, OAuth or email to set up, and no mobile build (the phone app is the web app, added to the home screen).

## Preflight (all must pass)

- [x] e2e suite green: **51 passed / 54** (3 skipped: phone-only or desktop-only tests in the other project). Every test fails on console errors, 5xx and axe WCAG 2.1 AA violations.
- [x] no open S1 or S2 bugs: BUG-001 to BUG-011 all fixed (`replica/bugs.md`).
- [x] parity: all must-haves done (**18/18**), feature score **87.1**.
- [x] rebrand sweep clean: `sweep.py . --config replica/brand.json` → "Clean", exit 0 (includes `docs/`).
- [x] store listing: **not applicable**, no app-store app (`replica/launch/launch-plan.md` §5 says why).
- [x] production build passes: `make all test` (Go tests with libvips, exiftool and ffmpeg present; svelte-check 0 errors), then `make deb deb-pi`:
  ```
  b65482d798254c1437c2e360f90b2d68948cfba155705e3a505ca7477f77aafd  picburrow_0.1.0_amd64.deb
  391aaa52e87a4b1fca00d172d7d7f36cc04f6a4ec8a4736aebb591c76e59611b  picburrow_0.1.0_arm64.deb
  ```
  (local build; the release workflow rebuilds them on GitHub, so its sums will differ.)
- [x] privacy and terms: no processors in the product (it makes no outbound connections; checked in `server/`). Privacy section in the README; privacy line on the landing page (GitHub Pages is the only processor: its server logs). Terms = the AGPL-3.0 licence. Cookies: only the sign-in session and one per unlocked share link, both strictly necessary, so no cookie banner.
- [x] account deletion: an admin deletes an account on the People page (e2e P-1); photo files are never deleted.
- [x] favicon, titles, OG image, app icon are Picburrow's (`web/static/`, `docs/img/`). The icon is the placeholder from `brand.md`'s logo brief.

**Open before announcing** (not before tagging a draft): a real Pi 3 with 1 GB, and a real iPhone library (`launch-plan.md` §0). They decide whether the headline "on a 1 GB Raspberry Pi" stays as written.

## Release (replaces "production services")

`.github/workflows/release.yml` runs on a version tag, `0.1.0` or `v0.1.0`:

1. checks the tag (without a leading `v`) equals `VERSION`;
2. installs libvips-tools, exiftool and ffmpeg so the image tests really run;
3. `make all test`;
4. `make deb deb-pi` and `SHA256SUMS`;
5. attaches both `.deb` files and `SHA256SUMS` to the tag's release: if you already made the release on GitHub's Releases page, the files are added to it; if not, a **draft** release is opened for you to publish.

**0.1.0 → 0.1.1 (2026-11-13)**: tag `0.1.0` was made with its release on the Releases page before the workflow accepted tags without `v`, so it got no files. Running the workflow for it then failed in `TestHEICThumbnail`: "heif: Unsupported codec". On Ubuntu 24.04 and newer Debian, the HEVC decoder iPhone photos need is a separate package (`libheif-plugin-libde265`) that libheif only *suggests*, so a plain install can't preview HEIC. Fixed in 0.1.1: the `.deb` depends on `libheif-plugin-libde265 | libheif1 (<< 1.16)` (older libheif has the decoder built in), the workflow installs it, the README lists it. `0.1.0` can't be rebuilt with the fix (its tag points at the old code), so: delete the empty 0.1.0 release and its tag, then release **0.1.1**.

Next versions: bump `VERSION`, merge, then either create the release on the Releases page with a new tag (e.g. `0.2.0`, target `main`), or from git:

```bash
git checkout main && git pull
```

```bash
git tag -a 0.2.0 -m "Picburrow 0.2.0"
```

```bash
git push origin 0.2.0
```

Either way the workflow builds and attaches the files. A tag that doesn't match `VERSION` fails at step 1, so nothing mismatched gets attached.

## Live (2026-11-13)

- **Repository**: public, https://github.com/oPercentOps/Picburrow (AGPL source offer in the app now resolves).
- **Release** (marked pre-release until the real-Pi test): https://github.com/oPercentOps/Picburrow/releases/tag/0.1.1, built by release run 43426194169 (tests incl. HEIC, both packages):
  ```
  435c7948551e199692958b1cd24f4689a89a0d018a7551bb872ed33166bc0254  picburrow_0.1.1_amd64.deb
  a003a693b3a8ffb236dc282dce5c36036cb395959bea61b390799812fe8b47cb  picburrow_0.1.1_arm64.deb
  ```
  Checked from outside: the public download of the arm64 package matches `SHA256SUMS`, Version 0.1.1, Depends include the HEVC decoder.
- **Landing page**: https://opercentops.github.io/Picburrow/ (deployment reported success; not opened from the build session, whose network doesn't reach github.io). The repository was renamed to `Picburrow` on 2026-11-13, which moved the page here; GitHub doesn't redirect the old `/RaspictureGalleryNas/` Pages address.
- **Not yet**: domain (picburrow.org unregistered as of the brand check), real Pi 3 and real iPhone library tests.
- **Maintenance note**: the workflow's actions (checkout v4, setup-go v5, setup-node v4) run on GitHub's deprecated Node 20 runtime; bump them when GitHub removes it.

## Landing page

- [x] Settings → Pages → Source: **Deploy from a branch** → `main` / `/docs`. Deployed 2026-11-13 ("pages build and deployment" run 43428336752: success).

## Domain (once you register it; suggested: picburrow.org as the canonical host)

At your registrar's DNS:

| record | name | value |
| --- | --- | --- |
| A | @ | 185.199.108.153 |
| A | @ | 185.199.109.153 |
| A | @ | 185.199.110.153 |
| A | @ | 185.199.111.153 |
| AAAA | @ | 2606:50c0:8000::153 |
| AAAA | @ | 2606:50c0:8001::153 |
| AAAA | @ | 2606:50c0:8002::153 |
| AAAA | @ | 2606:50c0:8003::153 |
| CNAME | www | opercentops.github.io |
| TXT | `_github-pages-challenge-opercentops` | the value GitHub shows under your profile → Settings → Pages → **Verified domains** → Add (verify the domain first; it stops anyone else's Pages site from claiming it) |

Then: Settings → Pages → Custom domain → `picburrow.org` → Save (GitHub adds a `docs/CNAME` commit), wait for the certificate, tick **Enforce HTTPS**. GitHub redirects `www` to the apex itself. Check these GitHub Pages addresses against GitHub's "Managing a custom domain" page when you do it: they rarely change, but they're the one value here you can't undo by typo-fixing later.

**Email:** Picburrow sends no email, so lock the domain against spoofing instead of setting up SPF/DKIM for a sender:

| record | name | value |
| --- | --- | --- |
| TXT | @ | `v=spf1 -all` |
| TXT | _dmarc | `v=DMARC1; p=reject;` |
| MX | @ | `0 .` (null MX: this domain receives no mail) |

If you later want a contact address (e.g. for security reports), replace these with your mail provider's records.

For .com and .app, if you register them too: point them at the same place with a redirect at the registrar (or leave them parked). Only one domain can be the Pages custom domain.

## Watch (first week after publishing)

No uptime checks or error tracking to run: there is no server of ours. Instead:

- [ ] **Release downloads**: Releases page shows per-file counts (arm64 vs amd64 tells you who installs on a Pi).
- [ ] **Issues**: the bug template asks for board, OS, `journalctl` output and the Indexing list. Answer within a day during launch week.
- [ ] **Traffic**: Insights → Traffic (visitors, referrers: which post worked).
- [ ] **Core flow on a real device**: install the arm64 `.deb` on a Pi, set up, open it on your phone, add to home screen, browse, open a photo, share an album link with a password. Note anything that differs from the e2e run.
- [ ] **The two open checks**: Pi 3 memory peak (target under 350 MB with 10k photos) and HEIC/live photos from a real iPhone. Update the landing page if either misses.
