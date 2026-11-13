<script lang="ts">
  import { api, thumbURL, browserCanShow, formatDay, type Asset } from '$lib/api';
  import { overrides } from '$lib/state.svelte';
  import { X, Info, Download, ChevronLeft, ChevronRight, ImageOff, Heart, Archive, ArchiveRestore } from '@lucide/svelte';

  // seq: ids to step through (a folder, favourites); without it prev/next follow the timeline
  let { id, seq, onnavigate, onclose }: { id: number; seq?: number[]; onnavigate: (id: number) => void; onclose: () => void } = $props();

  let asset = $state<Asset | null>(null);
  let error = $state('');
  let showInfo = $state(false);
  let scale = $state(1);
  let tx = $state(0);
  let ty = $state(0);
  let original = $state(false); // swapped in when zoomed, if the browser can decode it
  let previewFailed = $state(false);
  let dialog: HTMLElement;
  let opener: Element | null = null;
  let saveError = $state('');

  const at = $derived(seq ? seq.indexOf(id) : -1);
  const prevId = $derived(seq ? (at > 0 ? seq[at - 1] : null) : asset?.prev_id);
  const nextId = $derived(seq ? (at >= 0 && at < seq.length - 1 ? seq[at + 1] : null) : asset?.next_id);
  // after next/prev, `asset` still holds the previous photo until the new one loads: act only when they match (BUG-005)
  const ready = $derived(asset?.id === id);
  const fav = $derived(asset ? (overrides[asset.id]?.fav ?? asset.fav) : false);
  const archived = $derived(asset ? (overrides[asset.id]?.archived ?? asset.archived) : false);

  async function save(change: { favorite?: boolean; archived?: boolean }) {
    if (!asset || !ready) return;
    const target = asset.id;
    saveError = '';
    try {
      await api('/api/assets', { method: 'PATCH', body: { ids: [target], ...change } });
      overrides[target] = { ...overrides[target], ...('favorite' in change ? { fav: change.favorite } : { archived: change.archived }) };
      if ('archived' in change) nextId ? go(nextId) : prevId ? go(prevId) : onclose(); // it leaves this view
    } catch (e) {
      saveError = (e as Error).message;
    }
  }

  $effect(() => {
    const ctl = new AbortController();
    error = '';
    previewFailed = false;
    resetZoom();
    api<Asset>(`/api/assets/${id}`, { signal: ctl.signal })
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
  const go = (to: number | null | undefined) => { if (to) onnavigate(to); };

  function zoomTo(s: number) {
    scale = Math.min(Math.max(s, 1), 5);
    if (scale === 1) resetZoom();
    else if (asset && asset.type === 'image' && browserCanShow(asset.mime)) original = true;
  }

  function onkey(e: KeyboardEvent) {
    if (e.key === 'Escape') { showInfo ? (showInfo = false) : onclose(); }
    else if (e.key === 'ArrowLeft') go(prevId);
    else if (e.key === 'ArrowRight') go(nextId);
    else if (e.key === 'f') save({ favorite: !fav });
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
    if (!pointers.delete(e.pointerId) || pointers.size > 0 || scale > 1 || start.dist) return;
    const dx = e.clientX - start.x, dy = e.clientY - start.y;
    if (Math.abs(dx) > 60 && Math.abs(dx) > Math.abs(dy)) go(dx < 0 ? nextId : prevId);
    else if (dy > 100 && dy > Math.abs(dx)) onclose();
  }

  const camera = $derived(asset ? [asset.camera_make, asset.camera_model].filter(Boolean).join(' ').replace(/^(\w+) \1 /i, '$1 ') : '');
  const mb = (n: number) => n < 1048576 ? Math.max(1, Math.round(n / 1024)) + ' KB' : (n / 1048576).toFixed(n > 10485760 ? 0 : 1) + ' MB';
  const time = (taken: string) => taken.slice(11, 16);
  const offset = (m: number | null) => m == null ? '' : ` (UTC${m < 0 ? '−' : '+'}${String(Math.floor(Math.abs(m) / 60)).padStart(2, '0')}:${String(Math.abs(m) % 60).padStart(2, '0')})`;
</script>

<svelte:window onkeydown={onkey} />

<div class="viewer" role="dialog" aria-modal="true" aria-label="Photo viewer" tabindex="-1" bind:this={dialog}>
  <div class="stage" class:with-info={showInfo} onpointerdown={down} onpointermove={move} onpointerup={up} onpointercancel={up}
    ondblclick={() => zoomTo(scale > 1 ? 1 : 2.5)} onwheel={(e) => { if (e.ctrlKey) { e.preventDefault(); zoomTo(scale * (e.deltaY < 0 ? 1.1 : 0.9)); } }}
    role="presentation">
    {#if error}
      <div class="msg"><ImageOff aria-hidden="true" /><p>{error}</p></div>
    {:else if asset}
      {#if asset.type === 'video'}
        <!-- svelte-ignore a11y_media_has_caption -->
        <video src="/api/assets/{asset.id}/original" poster={thumbURL(asset, 'l')} controls playsinline autoplay></video>
      {:else if previewFailed || asset.err}
        <div class="msg"><ImageOff aria-hidden="true" /><p>This file can't be previewed here.</p>
          <a class="btn secondary" href="/api/assets/{asset.id}/original?download=1">Download original</a></div>
      {:else}
        <img src={original ? `/api/assets/${asset.id}/original` : thumbURL(asset, 'l')} alt="{asset.name}, {formatDay(asset.taken)}"
          style:transform="translate({tx}px, {ty}px) scale({scale})" draggable="false"
          onerror={() => (original ? (original = false) : (previewFailed = true))} />
      {/if}
    {/if}
  </div>

  <div class="bar top">
    <button class="btn icon-btn" onclick={onclose} aria-label="Close"><X aria-hidden="true" /></button>
    <div class="actions">
      {#if asset}
        <button class="btn icon-btn" onclick={() => save({ favorite: !fav })} aria-label="Favourite" aria-pressed={fav} disabled={!ready}>
          <Heart aria-hidden="true" fill={fav ? 'currentColor' : 'none'} /></button>
        <button class="btn icon-btn" onclick={() => save({ archived: !archived })} aria-label={archived ? 'Unarchive' : 'Archive'} disabled={!ready}>
          {#if archived}<ArchiveRestore aria-hidden="true" />{:else}<Archive aria-hidden="true" />{/if}</button>
      {/if}
      <button class="btn icon-btn" onclick={() => (showInfo = !showInfo)} aria-label="Details" aria-pressed={showInfo}><Info aria-hidden="true" /></button>
      {#if asset}<a class="btn icon-btn" href="/api/assets/{id}/original?download=1" aria-label="Download original"><Download aria-hidden="true" /></a>{/if}
    </div>
  </div>
  {#if prevId}<button class="btn icon-btn nav prev" onclick={() => go(prevId)} aria-label="Previous"><ChevronLeft aria-hidden="true" /></button>{/if}
  {#if nextId}<button class="btn icon-btn nav next" onclick={() => go(nextId)} aria-label="Next"><ChevronRight aria-hidden="true" /></button>{/if}
  {#if saveError}<div class="save-error" role="alert">{saveError}</div>{/if}
  {#if asset}<div class="bar bottom" aria-live="polite">{formatDay(asset.taken)} · {time(asset.taken)}</div>{/if}

  {#if showInfo && asset}
    <aside aria-label="Details">
      <h2>Details</h2>
      <dl>
        <dt>Taken</dt><dd>{formatDay(asset.taken)} {time(asset.taken)}{offset(asset.tz_offset_min)}</dd>
        <dt>File</dt><dd class="wrap">{asset.name}</dd>
        <dt>Folder</dt><dd class="wrap">{asset.dir}</dd>
        <dt>Size</dt><dd>{mb(asset.size)}{asset.w && asset.h ? ` · ${asset.w} × ${asset.h}` : ''}</dd>
        {#if camera}<dt>Camera</dt><dd>{camera}</dd>{/if}
        {#if asset.lens}<dt>Lens</dt><dd>{asset.lens}</dd>{/if}
        {#if asset.lat != null && asset.lon != null}<dt>Location</dt><dd>{asset.lat.toFixed(5)}, {asset.lon.toFixed(5)}</dd>{/if}
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
  @media (min-width: 768px) {
    aside { inset: 0 0 0 auto; width: var(--info-drawer); max-height: none; border-radius: 0; }
    .with-info { right: var(--info-drawer); }
  }
</style>
