<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, thumbURL, formatDay, type Tile } from '$lib/api';
  import { X, Play, Pause, Maximize, Minimize, Images } from '@lucide/svelte';

  // A passive, random slideshow (issue #20): random batches from the server, each photo decoded before it
  // cross-fades in over the last one, with a slow drift; the screen is kept awake while it runs.
  const CHOICES = [5, 10, 20, 60];
  const FADE = 1500; // ms
  const KEY = 'picburrow.slideshow.seconds';

  let seconds = $state(10);
  try { const s = Number(localStorage.getItem(KEY)); if (CHOICES.includes(s)) seconds = s; } catch {}
  let slides = $state<{ key: number; tile: Tile; src: string; out?: boolean }[]>([]);
  let paused = $state(false);
  let controls = $state(true);
  let empty = $state(false);
  let error = $state('');
  let full = $state(false);
  const canFull = typeof document !== 'undefined' && !!document.fullscreenEnabled;

  let queue: Tile[] = [];
  let key = 0;
  let timer: ReturnType<typeof setTimeout> | undefined;
  let hideTimer: ReturnType<typeof setTimeout> | undefined;
  let busy = false;

  async function show() {
    if (busy) return;
    busy = true;
    try {
      for (let tries = 0; tries < 5; tries++) {
        if (!queue.length) queue = await api<Tile[]>('/api/slideshow');
        if (!queue.length) { empty = true; return; }
        const tile = queue.shift()!;
        const src = thumbURL(tile, 'l');
        const img = new Image();
        img.src = src;
        try { await img.decode(); } catch { continue; } // unreadable preview: skip it
        // the old photo fades out while the new one fades in, so photos of different shapes blend too
        slides = [...slides.slice(-1).map((s) => ({ ...s, out: true })), { key: ++key, tile, src }];
        setTimeout(() => (slides = slides.filter((s) => !s.out)), FADE + 100);
        return;
      }
    } catch (e) {
      error = (e as Error).message;
    } finally {
      busy = false;
      schedule();
    }
  }

  function schedule() {
    clearTimeout(timer);
    if (!paused && !empty && !error) timer = setTimeout(show, seconds * 1000);
  }

  function setSeconds(s: number) {
    seconds = s;
    try { localStorage.setItem(KEY, String(s)); } catch {}
    schedule();
  }

  function togglePause() {
    paused = !paused;
    schedule();
  }

  function wake() {
    controls = true;
    clearTimeout(hideTimer);
    hideTimer = setTimeout(() => (controls = false), 3000);
  }

  function exit() {
    if (document.fullscreenElement) document.exitFullscreen().catch(() => {});
    goto('/');
  }

  async function toggleFull() {
    try {
      if (document.fullscreenElement) await document.exitFullscreen();
      else await document.documentElement.requestFullscreen();
    } catch {}
  }

  function onkey(e: KeyboardEvent) {
    wake();
    if (e.key === 'Escape' && !document.fullscreenElement) exit();
    else if (e.key === ' ' && !(e.target as HTMLElement).closest('button, select')) { e.preventDefault(); togglePause(); }
    else if (e.key === 'ArrowRight') show();
  }

  // keep the screen on while the slideshow runs, like a photo frame; the lock drops when the page is hidden
  $effect(() => {
    let lock: WakeLockSentinel | null = null;
    const take = () => { if (document.visibilityState === 'visible') navigator.wakeLock?.request('screen').then((l) => (lock = l)).catch(() => {}); };
    take();
    document.addEventListener('visibilitychange', take);
    return () => { document.removeEventListener('visibilitychange', take); lock?.release().catch(() => {}); };
  });

  $effect(() => {
    const onfs = () => (full = !!document.fullscreenElement);
    document.addEventListener('fullscreenchange', onfs);
    show();
    wake();
    return () => { clearTimeout(timer); clearTimeout(hideTimer); document.removeEventListener('fullscreenchange', onfs); };
  });
</script>

<svelte:head><title>Slideshow · Picburrow</title></svelte:head>
<svelte:window onkeydown={onkey} />

<div class="show" class:idle={!controls} role="region" aria-label="Slideshow" onpointermove={wake} onpointerdown={wake}
  style:--drift="{seconds + FADE / 1000}s">
  {#each slides as s (s.key)}
    <div class="slide" class:out={s.out}><img src={s.src} alt="{formatDay(s.tile.taken)}" draggable="false" /></div>
  {/each}

  {#if empty}
    <div class="msg"><Images aria-hidden="true" /><p>No photos to show yet. They appear here once their previews are made.</p>
      <button class="btn secondary" onclick={exit}>Back to Photos</button></div>
  {:else if error}
    <div class="msg"><p role="alert">{error}</p><button class="btn secondary" onclick={exit}>Back to Photos</button></div>
  {/if}

  <div class="bar" class:hidden={!controls}>
    <button class="btn icon-btn" onclick={exit} aria-label="Stop slideshow"><X aria-hidden="true" /></button>
    <span class="spacer"></span>
    <label class="every"><span class="sr-only">Change photo every</span>
      <select value={seconds} onchange={(e) => setSeconds(Number(e.currentTarget.value))}>
        {#each CHOICES as c}<option value={c}>{c < 60 ? `${c} s` : '1 min'}</option>{/each}
      </select>
    </label>
    <button class="btn icon-btn" onclick={togglePause} aria-label={paused ? 'Play' : 'Pause'}>
      {#if paused}<Play aria-hidden="true" />{:else}<Pause aria-hidden="true" />{/if}
    </button>
    {#if canFull}
      <button class="btn icon-btn" onclick={toggleFull} aria-label={full ? 'Exit full screen' : 'Full screen'}>
        {#if full}<Minimize aria-hidden="true" />{:else}<Maximize aria-hidden="true" />{/if}
      </button>
    {/if}
  </div>
  {#if controls && slides.length}<p class="date" aria-hidden="true">{formatDay(slides.at(-1)!.tile.taken)}</p>{/if}
</div>

<style>
  .show { position: fixed; inset: 0; z-index: 100; background: #000; color: #fff; overflow: hidden; }
  .show.idle { cursor: none; }
  .slide { position: absolute; inset: 0; animation: fade-in 1.5s ease forwards; }
  .slide.out { animation: fade-out 1.5s ease forwards; }
  .slide img { width: 100%; height: 100%; object-fit: contain; animation: drift var(--drift) linear forwards; }
  @keyframes fade-in { from { opacity: 0; } to { opacity: 1; } }
  @keyframes fade-out { from { opacity: 1; } to { opacity: 0; } }
  @keyframes drift { from { transform: scale(1); } to { transform: scale(1.06); } }
  .bar { position: absolute; inset: 0 0 auto; display: flex; align-items: center; gap: var(--space-2);
    padding: max(var(--space-3), env(safe-area-inset-top)) var(--space-3) var(--space-6);
    background: linear-gradient(rgba(0,0,0,.55), transparent); transition: opacity .3s; }
  .bar.hidden { opacity: 0; pointer-events: none; }
  .spacer { flex: 1; }
  .bar :global(.icon-btn) { color: #fff; background: transparent; }
  .bar :global(svg) { width: 24px; height: 24px; filter: drop-shadow(0 1px 2px rgba(0,0,0,.6)); }
  select { height: 40px; padding: 0 var(--space-3); border-radius: var(--radius-md); border: 1px solid rgba(255,255,255,.5);
    background: rgba(0,0,0,.45); color: #fff; font: inherit; }
  select:focus-visible, .bar :global(.btn:focus-visible) { outline: 2px solid #fff; outline-offset: 2px; }
  .date { position: absolute; left: 0; right: 0; bottom: 0; margin: 0; padding: var(--space-6) var(--space-4) max(var(--space-4), env(safe-area-inset-bottom));
    background: linear-gradient(transparent, rgba(0,0,0,.55)); font: var(--text-sm) var(--font-sans); }
  .msg { position: absolute; inset: 0; display: grid; place-content: center; justify-items: center; gap: var(--space-4); padding: var(--space-5); text-align: center; }
  .msg :global(svg) { width: 48px; height: 48px; color: #b8bec7; }
  @media (prefers-reduced-motion: reduce) {
    .slide { animation-duration: .6s; }
    .slide.out { animation-duration: .6s; }
    .slide img { animation: none; } /* no drift */
    .bar { transition: none; }
  }
</style>
