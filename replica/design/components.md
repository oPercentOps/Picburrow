# Components

Tokens: `tokens.json` → `tokens.css` (`python3 gen_css.py`). Components use `var(--role)`, never raw hex.
Icons: **Lucide** (ISC licence), inlined as SVG, `stroke-width: 2`, 20 px (24 px in the viewer). No icons, art or copy from Immich or PhotoPrism.
Font: the system stack, so the Pi serves zero font bytes.
Global rules:
- Focus-visible ring is `2px solid var(--accent)` with a `2px` offset on every interactive element.
- Touch targets are ≥ `--touch-target` (44 px).
- `prefers-reduced-motion` drops all transitions to 0.
- Glass (bottom bar, toast, selection bar, sticky date header): `color-mix(var(--bg) 90%, transparent)` + `backdrop-filter: blur`. Never below 90%, so muted text keeps 4.5:1 over any photo behind it.
- Buttons and nav items are pills (`radius-pill`); pressing scales them to .97.
- Text placed over a photo always sits on a scrim (`linear-gradient(rgba(0,0,0,.55), transparent)`), because contrast against an arbitrary photo cannot be checked.

```
Button
  variants  primary (accent), secondary (surface + border), ghost (transparent), danger (bg danger, text bg)
  sizes     sm 32px, md 40px (icon-only buttons are 44px square on touch)
  states    default, hover (bg mixes 8% text), active, focus-visible, disabled (opacity .5, no pointer), loading (spinner replaces icon, label kept)
  tokens    primary bg accent / text on-accent; radius md; font sm/600
  a11y      real <button>; icon-only needs aria-label; loading sets aria-busy
  used on   all

Icon button (toolbar)
  variants  default, on-photo (white icon + shadow on-photo), toggled (favourite filled)
  states    default, hover, focus-visible, pressed (aria-pressed)
  tokens    44px square, radius pill
  a11y      aria-label always; favourite toggle uses aria-pressed
  used on   S03 S04 S08 S12

Thumbnail tile
  variants  image, video (play badge + duration, bottom-right), live (live badge, top-left), favourite (heart, bottom-left)
  states    placeholder (thumbhash blur, or surface colour), loaded (fade-in base), error (broken-image icon + filename on surface),
            selectable (circle top-left on hover/select mode), selected (tile inset 8px, 3px accent ring, check filled), focus-visible
  tokens    grid-gap 3px, radius sm in the grid (radius lg on album and folder covers, lift + shadow pop on hover), badges on scrim
  a11y      <a> to the viewer route with alt = filename + date; in select mode role=checkbox + aria-checked; Space toggles, Enter opens
  used on   S03 S06 S08 S09 S10 S11 S12 S15

Justified grid (virtualised)
  variants  justified rows (height 120 mobile / 220 desktop, last row not stretched), square (folders)
  states    empty (see Empty state), loading month (placeholder tiles sized from bucket counts), filled
  behaviour only months in viewport ±1 screen are mounted; month heights estimated from /timeline/buckets, then corrected
  a11y      grid is a list of sections (one per day, labelled by date header); arrow keys move focus between tiles
  used on   S03 S06 S08-S12 S15

Date header
  variants  day ("Sat 14 Sep 2024"), month ("September 2024")
  states    default, sticky (bg with 90% opacity + blur), select-all checkbox on hover/select mode
  tokens    font sm/600, text; padding space-5 space-4
  a11y      <h2>/<h3>; the select-all checkbox is labelled "Select all from <date>"
  used on   S03 S09 S11

Scrubber
  variants  desktop (right rail, year labels), mobile (draggable thumb appears while scrolling)
  states    idle (hidden on mobile), hover (month tooltip), dragging (large month label), focus-visible
  tokens    rail 24px wide, thumb accent, label on surface + shadow pop
  a11y      role=slider, aria-valuetext = "March 2021"; PageUp/PageDown = month, Home/End
  used on   S03

Viewer
  variants  photo, video (native <video controls playsinline>), live (still, hold/hover plays motion clip, muted)
  states    loading (thumbnail scaled up + spinner after 300ms), loaded (large preview, then original on zoom ≥ 2x), zoomed (pan),
            unsupported (message + Download original), missing file (message), first/last (prev/next disabled)
  tokens    bg viewer-bg; toolbar on top scrim; icons viewer-text
  behaviour swipe left/right = next/prev, swipe down = close, pinch/double-tap = zoom; desktop ←/→, Esc, F (favourite), I (info), +/−
  a11y      role=dialog aria-modal, focus trapped, returns focus to the opened tile; image alt as tile; live region announces "Photo 12 of 340"
  used on   S04 S12

Info drawer
  variants  right drawer (desktop, 360px, pushes the viewer), bottom sheet (mobile, swipe up from viewer)
  states    open, closed, field missing (row hidden, not "N/A"), no EXIF date ("Date from file, may be wrong" in warning)
  content   date + time, filename + folder (link to S06), size + dimensions, camera + lens + exposure, place + mini map (could), albums, tags (could)
  tokens    bg viewer-surface, label text-muted xs, value text sm
  a11y      <aside> labelled "Details"; Esc closes; first heading gets focus
  used on   S04

Navigation
  variants  sidebar (≥ bp-md: Photos, Folders, Albums, Favourites, Archive, Search; admin section for admins), bottom nav (< bp-md: Photos, Folders, Albums, Search, More)
  states    item default, hover, active (accent text + 3px accent bar / filled icon), focus-visible
  tokens    sidebar 240px, bottom nav 64px + safe-area-inset-bottom, header 56px
  a11y      <nav aria-label="Main">, aria-current="page" on active item
  used on   all signed-in screens

Top bar
  variants  default (title or search field + user menu), select mode (close X, "3 selected", actions: favourite, add to album, archive, download)
  states    default, select mode (bg surface, stays fixed)
  a11y      selection count is an aria-live region
  used on   S03 S06 S08-S11 S15

Breadcrumb
  variants  full, collapsed (mobile: "…/ parent / current")
  a11y      <nav aria-label="Breadcrumb">, last item aria-current="page"
  used on   S06

Folder tile
  variants  with cover (latest photo, radius md), empty (folder icon on surface)
  content   name (sm/600, 2 lines max), count (xs, text-muted)
  a11y      <a>, label "Holidays 2023, 312 items"
  used on   S06

Album card
  variants  default, shared (link icon)
  content   cover (radius md, square), name, count + date range
  used on   S07

Modal
  variants  dialog (add to album, share link, rename, confirm), sheet on mobile
  states    open, submitting (primary button loading), error (inline message, danger)
  tokens    bg bg, radius lg, shadow pop, backdrop rgba(0,0,0,.5); max 480px
  a11y      <dialog>, labelled by its title, focus trap, Esc closes, focus returns to trigger
  used on   S04 S07 S08

Share-link panel (inside modal)
  fields    password (optional), expires (never / 1 day / 1 week / 1 month / date), allow download (switch)
  states    created (URL in read-only field + Copy button → "Copied" for 2 s), revoked
  used on   S08

Toast
  variants  info, success, error (stays until dismissed), with action ("Undo")
  tokens    bg text / text bg (inverted), radius md, shadow pop; bottom centre above bottom nav
  a11y      role=status (info/success) or role=alert (error); does not steal focus
  used on   all

Progress
  variants  bar determinate (thumbnails 1,240 / 10,000), indeterminate (scanning), inline banner on timeline ("Still indexing: new photos appear as they're ready")
  tokens    track border, fill accent, height 4px
  a11y      role=progressbar with aria-valuenow/max; banner is role=status
  used on   S03 S14

Form field
  variants  text, password (show/hide), path (with "Check" → readable / not readable), glob list (one per line, monospace), switch, select
  states    default, focus-visible (border accent), error (border danger + message below, aria-describedby), disabled
  tokens    height 40px, border border-input, radius sm, label sm/600 above
  a11y      <label for>; errors announced via aria-describedby + aria-invalid
  used on   S01 S02 S11 S13

Filter chips (search)
  variants  toggle (Favourites, Videos), with value (Date: 2021, Camera: Pixel 7) + remove X
  states    default, active (bg accent-tint, text accent, border accent), focus-visible
  a11y      toggle = aria-pressed; remove button labelled "Remove filter Camera: Pixel 7"
  used on   S11

Empty state
  variants  no library yet (CTA "Add a photo folder", admin only), empty folder, no favourites, no search results, empty album, link expired
  content   Lucide icon 48px text-muted, one-line title, one-line hint, optional button. Copy written fresh, for example:
            "No favourites yet. Tap the heart on any photo to keep it here."
  used on   S03 S06-S12 S15

Password gate (shared link)
  content   album name, password field, "Open" button; error "Wrong password"
  used on   S12
```

## Not taken from the originals

- Icons: Lucide, not Immich's Material Design icons and not PhotoPrism's set.
- Colours: Picburrow burnt orange `#b4470f` (light) / `#ff9a5c` (dark), chosen in replica-brand outside the blue/purple family of the apps it was modelled on.
- Copy: every label and empty state above is new.
- Layout patterns (justified grid, date scrubber, black viewer, info drawer) are common to every photo app, not owned by either original.
