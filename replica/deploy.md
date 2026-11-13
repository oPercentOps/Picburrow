# Deploy checklist: Picburrow

Date: 2026-11-13  Commit: the merge of this PR on `main`  Go from user: **not yet** (asked in the session; the release is a draft until you press Publish)

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

`.github/workflows/release.yml`, on a tag `v<VERSION>`:

1. checks the tag equals `v$(cat VERSION)`;
2. installs libvips-tools, exiftool and ffmpeg so the image tests really run;
3. `make all test`;
4. `make deb deb-pi` and `SHA256SUMS`;
5. creates a **draft** release with both `.deb` files and `SHA256SUMS`, notes generated from merged PRs.

To release, after this PR is merged:

```bash
git checkout main && git pull
```

```bash
git tag -a v0.1.0 -m "Picburrow 0.1.0"
```

```bash
git push origin v0.1.0
```

Then, on GitHub → Releases: read the draft, tick **"Set as a pre-release"** while the Pi 3 test is open, and press **Publish**. Next versions: bump `VERSION`, merge, tag the same way.

## Landing page

- [ ] Settings → Pages → Source: **Deploy from a branch** → `main` / `/docs` → Save. It appears at `https://opercentops.github.io/RaspictureGalleryNas/` within a minute or two.

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
