<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, thumbURL, browserCanShow, formatDay, type Asset } from '$lib/api';
  import { overrides, picker } from '$lib/state.svelte';
  import { SvelteSet } from 'svelte/reactivity';
  import { X, Info, Download, ChevronLeft, ChevronRight, ImageOff, Heart, Archive, ArchiveRestore, BookPlus, CirclePlay, RotateCw, Star } from '@lucide/svelte';

  // seq: ids to step through (a folder, favourites); without it prev/next follow the timeline.
  // base + readonly: a visitor on a shared album reads through the link and can't change anything.
  let { id, seq, onnavigate, onclose, base = '/api', readonly = false }: {
    id: number; seq?: number[]; onnavigate: (id: number) => void; onclose: () => void; base?: string; readonly?: boolean;
  } = $props();

  let asset = $state<Asset | null>(null);
  let error = $state('');
  let showInfo = $state(false);
  // capture settings, read from the file when the details panel opens (owner only)
  let details = $state<Record<string, string> | null>(null);
  $effect(() => {
    details = null;
    if (!showInfo || readonly || !ready) return;
    const ctl = new AbortController();
    api<Record<string, string>>(`/api/assets/${id}/details`, { signal: ctl.signal }).then((d) => (details = d)).catch(() => {});
    return () => ctl.abort();
  });
  const exposure = $derived(details ? [details.FNumber && `f/${details.FNumber}`, details.ExposureTime && `${details.ExposureTime} s`,
    details.ISO && `ISO ${details.ISO}`, details.ExposureCompensation && details.ExposureCompensation !== '0' && `${details.ExposureCompensation} EV`].filter(Boolean).join(' · ') : '');
  const focal = $derived(details?.FocalLength ? details.FocalLength + (details.FocalLengthIn35mmFormat ? ` (${t('{mm} equiv.', { mm: details.FocalLengthIn35mmFormat })})` : '') : '');
  let scale = $state(1);
  let tx = $state(0);
  let ty = $state(0);
  let original = $state(false); // swapped in when zoomed, if the browser can decode it
  let previewFailed = $state(false);
  let dialog: HTMLElement;
  let opener: Element | null = null;
  let saveError = $state('');

  // photos archived/unarchived here have left the list this viewer steps through (BUG-008)
  const left = new SvelteSet<number>();
  const stepIn = (dir: 1 | -1) => {
    if (!seq) return null;
    for (let i = seq.indexOf(id) + dir; i >= 0 && i < seq.length; i += dir) if (!left.has(seq[i])) return seq[i];
    return null;
  };
  const prevId = $derived(seq ? stepIn(-1) : asset?.prev_id);
  const nextId = $derived(seq ? stepIn(1) : asset?.next_id);
  // after next/prev, `asset` still holds the previous photo until the new one loads: act only when they match (BUG-005)
  const ready = $derived(asset?.id === id);
  // a shared album can turn off downloads (and so video playback, which streams the file)
  const canDownload = $derived(!readonly || (asset as (Asset & { allow_download?: boolean }) | null)?.allow_download !== false);
  const fav = $derived(asset ? (overrides[asset.id]?.fav ?? asset.fav) : false);
  const archived = $derived(asset ? (overrides[asset.id]?.archived ?? asset.archived) : false);

  async function save(change: { favorite?: boolean; archived?: boolean }) {
    if (!asset || !ready || readonly) return;
    const target = asset.id;
    saveError = '';
    try {
      await api('/api/assets', { method: 'PATCH', body: { ids: [target], ...change } });
      overrides[target] = { ...overrides[target], ...('favorite' in change ? { fav: change.favorite } : { archived: change.archived }) };
      if ('archived' in change) {
        left.add(target); // it leaves this view
        nextId ? go(nextId) : prevId ? go(prevId) : onclose();
      }
    } catch (e) {
      saveError = (e as Error).message;
    }
  }

  // a quarter turn clockwise, kept in the app: previews are remade turned, the file stays as it is
  const canRotate = $derived(!!asset && !readonly && asset.type === 'image' && !asset.live);
  async function rotate() {
    if (!asset || !ready || !canRotate) return;
    saveError = '';
    try {
      await api('/api/assets', { method: 'PATCH', body: { ids: [asset.id], rotate: 90 } });
      const a = await api<Asset>(`/api/assets/${asset.id}`);
      a.v = Date.now(); // a new address for the previews: the old ones are cached under the old one
      overrides[a.id] = { ...overrides[a.id], w: a.w, h: a.h, v: a.v };
      if (a.id === id) { resetZoom(); previewFailed = false; asset = a; }
    } catch (e) {
      saveError = (e as Error).message;
    }
  }

  $effect(() => {
    const ctl = new AbortController();
    error = '';
    previewFailed = false;
    playingLive = false;
    resetZoom();
    api<Asset>(`${base}/assets/${id}`, { signal: ctl.signal })
      .then((a) => (asset = a))
      .catch((e) => { if (e.name !== 'AbortError') error = e.message; });
    return () => ctl.abort();
  });

  $effect(() => {
    opener = document.activeElement;
    dialog.focus({ preventScroll: true });
    const root = document.documentElement;
    const prev = root.style.overflow;
    root.style.overflow = 'hidden'; // lock the page behind without losing its scroll position
    return () => {
      root.style.overflow = prev;
      (opener as HTMLElement | null)?.focus?.({ preventScroll: true });
    };
  });

  function resetZoom() { scale = 1; tx = 0; ty = 0; original = false; }
  let playingLive = $state(false);
  let rating = $state<number | null>(null);
  $effect(() => { rating = asset?.rating ?? null; });
  async function rate(n: number) {
    if (!asset || !ready || readonly) return;
    const prev = rating;
    rating = n || null;
    try {
      await api('/api/assets', { method: 'PATCH', body: { ids: [asset.id], rating: n } });
    } catch (e) {
      rating = prev;
      saveError = (e as Error).message;
    }
  }
  const go = (to: number | null | undefined) => { if (to) onnavigate(to); };

  function zoomTo(s: number) {
    scale = Math.min(Math.max(s, 1), 5);
    if (scale === 1) resetZoom();
    else if (asset && asset.type === 'image' && browserCanShow(asset.mime) && !asset.rotation) original = true; // the file isn't turned
  }

  function onkey(e: KeyboardEvent) {
    if (e.ctrlKey || e.metaKey || e.altKey) return; // Ctrl+R reloads the page: it must not also rotate the photo
    if (e.key === 'Escape') { showInfo ? (showInfo = false) : onclose(); }
    else if (e.key === 'ArrowLeft') go(prevId);
    else if (e.key === 'ArrowRight') go(nextId);
    else if (e.key === 'f') save({ favorite: !fav });
    else if (/^[0-5]$/.test(e.key) && !readonly) rate(Number(e.key));
    else if (e.key === 'l' && asset?.live) playingLive = !playingLive;
    else if (e.key === 'r') rotate();
    else if (e.key === 'i') showInfo = !showInfo;
    else if (e.key === '+' || e.key === '=') zoomTo(scale * 1.5);
    else if (e.key === '-') zoomTo(scale / 1.5);
    else if (e.key === 'Tab') trapFocus(e);
  }

  function trapFocus(e: KeyboardEvent) {
    const f = [...dialog.querySelectorAll<HTMLElement>('button, a[href], video')].filter((x) => !x.hasAttribute('disabled'));
    if (!f.length) return;
    if (e.shiftKey && document.activeElement === f[0]) { e.preventDefault(); f.at(-1)!.focus(); }
    else if (!e.shiftKey && document.activeElement === f.at(-1)) { e.preventDefault(); f[0].focus(); }
  }

  // gestures: swipe left/right = next/prev, swipe down = close, pinch or double-tap = zoom, drag = pan when zoomed
  const pointers = new Map<number, { x: number; y: number }>();
  let start = { x: 0, y: 0, tx: 0, ty: 0, dist: 0, scale: 1 };
  const dist = () => { const [a, b] = [...pointers.values()]; return Math.hypot(a.x - b.x, a.y - b.y); };

  function down(e: PointerEvent) {
    if ((e.target as HTMLElement).closest('button, a, video, aside')) return;
    pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
    start = { x: e.clientX, y: e.clientY, tx, ty, dist: pointers.size === 2 ? dist() : 0, scale };
  }
  function move(e: PointerEvent) {
    if (!pointers.has(e.pointerId)) return;
    pointers.set(e.pointerId, { x: e.clientX, y: e.clientY });
    if (pointers.size === 2 && start.dist) zoomTo((start.scale * dist()) / start.dist);
    else if (scale > 1) { tx = start.tx + e.clientX - start.x; ty = start.ty + e.clientY - start.y; }
  }
  function up(e: PointerEvent) {
    if (!pointers.delete(e.pointerId) || pointers.size > 0 || start.dist) return;
    const dx = e.clientX - start.x, dy = e.clientY - start.y;
    // a tap or click on the photo closes the details, like tapping outside any sheet
    if (showInfo && e.type === 'pointerup' && Math.hypot(dx, dy) < 10) { showInfo = false; return; }
    if (scale > 1) return;
    if (Math.abs(dx) > 60 && Math.abs(dx) > Math.abs(dy)) go(dx < 0 ? nextId : prevId);
    else if (dy > 100 && dy > Math.abs(dx)) onclose();
  }

  const camera = $derived(asset ? [asset.camera_make, asset.camera_model].filter(Boolean).join(' ').replace(/^(\w+) \1 /i, '$1 ') : '');
  const mb = (n: number) => n < 1048576 ? Math.max(1, Math.round(n / 1024)) + ' KB' : (n / 1048576).toFixed(n > 10485760 ? 0 : 1) + ' MB';
  const time = (taken: string) => taken.slice(11, 16);
  const offset = (m: number | null) => m == null ? '' : ` (UTC${m < 0 ? '−' : '+'}${String(Math.floor(Math.abs(m) / 60)).padStart(2, '0')}:${String(Math.abs(m) % 60).padStart(2, '0')})`;
</script>

<svelte:window onkeydown={onkey} />

<div class="viewer" role="dialog" aria-modal="true" aria-label={t('Photo viewer')} tabindex="-1" bind:this={dialog}>
  <div class="stage" class:with-info={showInfo} onpointerdown={down} onpointermove={move} onpointerup={up} onpointercancel={up}
    ondblclick={() => zoomTo(scale > 1 ? 1 : 2.5)} onwheel={(e) => { if (e.ctrlKey) { e.preventDefault(); zoomTo(scale * (e.deltaY < 0 ? 1.1 : 0.9)); } }}
    role="presentation">
    {#if error}
      <div class="msg"><ImageOff aria-hidden="true" /><p>{error}</p></div>
    {:else if asset}
      {#if asset.type === 'video'}
        <!-- svelte-ignore a11y_media_has_caption -->
        {#if canDownload}
          <video src="{base}/assets/{asset.id}/play" poster={thumbURL(asset, 'l', base)} controls playsinline autoplay></video>
        {:else}
          <div class="msg"><img src={thumbURL(asset, 'l', base)} alt="{asset.name}, {formatDay(asset.taken)}" /><p>{t('The owner turned off video playback and downloads for this album.')}</p></div>
        {/if}
      {:else if previewFailed || asset.err}
        <div class="msg"><ImageOff aria-hidden="true" /><p>{t("This file can't be previewed here.")}</p>
          {#if canDownload}<a class="btn secondary" href="{base}/assets/{asset.id}/original?download=1">{t('Download original')}</a>{/if}</div>
      {:else}
        <img src={original ? `${base}/assets/${asset.id}/original` : thumbURL(asset, 'l', base)} alt="{asset.name}, {formatDay(asset.taken)}"
          style:transform="translate({tx}px, {ty}px) scale({scale})" draggable="false"
          onerror={() => (original ? (original = false) : (previewFailed = true))} />
        {#if playingLive && asset.live && canDownload}
          <!-- the clip plays over the still, muted like a phone gallery, then the still comes back -->
          <video class="live" src="{base}/assets/{asset.live}/play" muted playsinline autoplay
            onended={() => (playingLive = false)} onerror={() => (playingLive = false)}></video>
        {/if}
      {/if}
    {/if}
  </div>

  <div class="bar top">
    <button class="btn icon-btn" onclick={onclose} aria-label={t('Close')}><X aria-hidden="true" /></button>
    <div class="actions">
      {#if asset && !readonly}
        <button class="btn icon-btn" onclick={() => (picker.ids = [id])} aria-label={t('Add to album')} disabled={!ready}><BookPlus aria-hidden="true" /></button>
        <button class="btn icon-btn" onclick={() => save({ favorite: !fav })} aria-label={t('Favourite')} aria-pressed={fav} disabled={!ready}>
          <Heart aria-hidden="true" fill={fav ? 'currentColor' : 'none'} /></button>
        <button class="btn icon-btn" onclick={() => save({ archived: !archived })} aria-label={archived ? t('Unarchive') : t('Archive|verb')} disabled={!ready}>
          {#if archived}<ArchiveRestore aria-hidden="true" />{:else}<Archive aria-hidden="true" />{/if}</button>
        {#if canRotate}<button class="btn icon-btn" onclick={rotate} aria-label={t('Rotate')} disabled={!ready}><RotateCw aria-hidden="true" /></button>{/if}
      {/if}
      {#if asset?.live && canDownload}
        <button class="btn icon-btn" onclick={() => (playingLive = !playingLive)} aria-label={t('Play live photo')} aria-pressed={playingLive}><CirclePlay aria-hidden="true" /></button>
      {/if}
      <button class="btn icon-btn" onclick={() => (showInfo = !showInfo)} aria-label={t('Details')} aria-pressed={showInfo}><Info aria-hidden="true" /></button>
      {#if asset && canDownload}<a class="btn icon-btn" href="{base}/assets/{id}/original?download=1" aria-label={t('Download original')}><Download aria-hidden="true" /></a>{/if}
    </div>
  </div>
  {#if prevId}<button class="btn icon-btn nav prev" onclick={() => go(prevId)} aria-label={t('Previous')}><ChevronLeft aria-hidden="true" /></button>{/if}
  {#if nextId}<button class="btn icon-btn nav next" onclick={() => go(nextId)} aria-label={t('Next')}><ChevronRight aria-hidden="true" /></button>{/if}
  {#if saveError}<div class="save-error" role="alert">{saveError}</div>{/if}
  {#if asset}<div class="bar bottom" aria-live="polite">{formatDay(asset.taken)} · {time(asset.taken)}</div>{/if}

  {#if showInfo && asset}
    <aside aria-label={t('Details')}>
      <h2>{t('Details')}</h2>
      <dl>
        <dt>{t('Taken')}</dt><dd>{formatDay(asset.taken)} {time(asset.taken)}{offset(asset.tz_offset_min)}</dd>
        <dt>{t('File')}</dt><dd class="wrap">{asset.name}</dd>
        {#if asset.dir}<dt>{t('Folder')}</dt><dd class="wrap">{asset.dir}</dd>{/if}
        {#if asset.size}<dt>{t('Size')}</dt><dd>{mb(asset.size)}{asset.w && asset.h ? ` · ${asset.w} × ${asset.h}` : ''}</dd>{/if}
        {#if camera}<dt>{t('Camera')}</dt><dd>{camera}</dd>{/if}
        {#if asset.lens}<dt>{t('Lens')}</dt><dd>{asset.lens}</dd>{/if}
        {#if exposure}<dt>{t('Exposure')}</dt><dd>{exposure}</dd>{/if}
        {#if focal}<dt>{t('Focal length')}</dt><dd>{focal}</dd>{/if}
        {#if details?.Flash}<dt>{t('Flash')}</dt><dd>{details.Flash}</dd>{/if}
        {#if !readonly}
          <dt id="rating-label">{t('Rating')}</dt>
          <dd>
            <div class="stars" role="radiogroup" aria-labelledby="rating-label">
              {#each [1, 2, 3, 4, 5] as n}
                <button role="radio" aria-checked={rating === n} aria-label={t(n === 1 ? '1 star' : '{n} stars', { n })} disabled={!ready}
                  class:on={(rating ?? 0) >= n} onclick={() => rate(rating === n ? 0 : n)}><Star aria-hidden="true" /></button>
              {/each}
            </div>
          </dd>
        {/if}
        {#if asset.lat != null && asset.lon != null}<dt>{t('Location')}</dt><dd>{asset.lat.toFixed(5)}, {asset.lon.toFixed(5)}</dd>{/if}
      </dl>
    </aside>
  {/if}
</div>

<style>
  .viewer { position: fixed; inset: 0; z-index: 50; background: var(--viewer-bg); color: var(--viewer-text); outline: none; }
  .stage { position: absolute; inset: 0; display: grid; place-items: center; overflow: hidden; touch-action: none; user-select: none; }
  .stage img, .stage video { max-width: 100%; max-height: 100%; object-fit: contain; transition: transform var(--motion-fast) var(--motion-ease); }
  .stage video { width: 100%; height: 100%; }
  .msg { display: grid; justify-items: center; gap: var(--space-3); color: var(--viewer-text-muted); text-align: center; padding: var(--space-5); }
  .msg :global(svg) { width: 48px; height: 48px; }
  .bar { position: absolute; left: 0; right: 0; display: flex; justify-content: space-between; align-items: center; padding: var(--space-2);
    padding-top: max(var(--space-2), env(safe-area-inset-top)); pointer-events: none; }
  .bar > * { pointer-events: auto; }
  .top { top: 0; background: linear-gradient(rgba(0, 0, 0, .55), transparent); }
  .bottom { bottom: 0; justify-content: center; padding: var(--space-6) 0 calc(var(--space-4) + env(safe-area-inset-bottom));
    background: linear-gradient(transparent, rgba(0, 0, 0, .55)); color: var(--viewer-text-muted); font: var(--text-sm) var(--font-sans); pointer-events: none; }
  .actions { display: flex; }
  .save-error { position: absolute; top: 64px; left: 50%; translate: -50% 0; padding: var(--space-3) var(--space-5); border-radius: var(--radius-md);
    background: var(--viewer-surface); color: var(--viewer-text); font: var(--text-sm) var(--font-sans); }
  .viewer .btn.icon-btn { color: var(--viewer-text); background: transparent; }
  .viewer .nav { position: absolute; top: 50%; translate: 0 -50%; background: var(--viewer-surface); }
  .prev { left: var(--space-3); }
  .next { right: var(--space-3); }
  .viewer :global(svg) { width: 24px; height: 24px; }
  @media (pointer: coarse) { .viewer .nav { display: none; } } /* swipe instead */
  aside { position: absolute; z-index: 2; background: var(--viewer-surface); color: var(--viewer-text); overflow-y: auto;
    inset: auto 0 0 0; max-height: 60%; border-radius: var(--radius-lg) var(--radius-lg) 0 0; padding: var(--space-6) var(--space-5) calc(var(--space-6) + env(safe-area-inset-bottom)); }
  aside h2 { font: var(--weight-lg) var(--text-lg) var(--font-sans); margin: 0 0 var(--space-5); }
  dl { display: grid; grid-template-columns: auto 1fr; gap: var(--space-3) var(--space-5); margin: 0; font: var(--text-sm) var(--font-sans); }
  dt { color: var(--viewer-text-muted); }
  dd { margin: 0; }
  .wrap { overflow-wrap: anywhere; }
  .stage .live { position: absolute; inset: 0; width: 100%; height: 100%; object-fit: contain; background: var(--viewer-bg); }
  .stars { display: flex; gap: 0; margin: -10px 0 -10px -10px; }
  .stars button { width: 40px; height: 40px; padding: 0; border: 0; background: none; color: var(--viewer-text-muted); cursor: pointer; display: grid; place-items: center; }
  .stars button :global(svg) { width: 20px; height: 20px; }
  .stars button.on { color: var(--viewer-text); } /* the viewer is always dark: theme colours would fail contrast here */
  .stars button.on :global(svg) { fill: currentColor; }
  @media (min-width: 768px) {
    aside { inset: 0 0 0 auto; width: var(--info-drawer); max-height: none; border-radius: 0; }
    .with-info { right: var(--info-drawer); }
  }
</style>
