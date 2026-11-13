# Pricing

**Decision (owner, 2026-11-13): Picburrow is free and open source, AGPL-3.0-or-later. No paid tier, no locked feature, no account with us.** So this file records the market for context, and how the project can be supported without selling features.

## What the alternatives charge

Read 2026-11-13. Prices change, so re-read before quoting any of this.

| product | model | price | what money buys | source |
| --- | --- | --- | --- | --- |
| PhotoPrism | free community edition + memberships | Community **€0**; Essentials **€2.00/month**; Plus **from €6.00/month**; memberships "billed annually"; a hosted partner plan from **$6.50/month** | Essentials: more user roles (the free edition has Super Admin, Admin and Guest; paid adds User and Viewer), 3D/satellite maps, better geocoding. Plus: a user-management web UI, vector graphics, private support chat. | https://www.photoprism.app/editions |
| Immich | everything free; optional one-time "product key" | the price is on https://buy.immich.app, which renders only with JavaScript and was **not read** | Nothing: its docs say "all features will be free, forever! Nothing will ever be put behind any paywalls"; the key funds full-time development | https://docs.immich.app/overview/support-the-project, https://docs.immich.app/FAQ |
| **Picburrow** | free software, AGPL-3.0 | **€0** | everything, including family accounts | this repo |

(Research only. Keep both names out of the landing page, ads and listings: `brand.json` `avoid`.)

## What reviewers said about price

From `feedback.md` (383 Hacker News comments, one source, so every theme is **thin**):

- "Price and paywalls": **26 comments (7%)**.
- "Paid tiers and licence": **12 comments (3%)**.
- The pointed ones, read by hand (`fixes.md` H4): multi-user documented under a paid plan, and someone who "even paid the server license" and then left. Read as "people notice what a family needs being behind a plan", not as a measured trend.

## Picburrow's model

- **One tier: free, for everyone in the home.** The thing people pointed at (family access behind a plan) is free here: accounts, each with its own folder.
- **No billing, so none of the billing complaints can happen**: nothing to cancel, no renewals, no per-seat jumps. Nothing to build in the product.
- **No Stripe products to create.**
- **It never calls home** (checked: the server makes no outbound network requests), so there is no licence check or account server to keep running either.

## If you want support later (optional, owner's call)

Donations that buy **nothing** in the product, so the "free, all of it" promise holds:

| option | fit | note |
| --- | --- | --- |
| GitHub Sponsors | lowest effort: the money and the code live in the same place | needs a GitHub account with payouts set up; add `.github/FUNDING.yml` |
| Liberapay | recurring donations, non-profit platform | European, no fees beyond payment processing |
| Open Collective | transparent ledger if more maintainers join | more setup |

Not recommended: a "supporter" badge in the app, feature bounties, or a paid hosted version. Each pulls the roadmap toward whoever pays, which is the thing the research says people distrust.

If you add one, put a single "Support the project" link in the README and on the landing page; don't nag in the app.
