<script lang="ts">
  import { thumbURL, formatDay, formatDuration, type Tile } from '$lib/api';
  import { justify, aspectOf } from '$lib/justify';
  import { overrides, openViewer, selection } from '$lib/state.svelte';
  import { Play, Heart, ImageOff, Check } from '@lucide/svelte';

  // byDay: date headers per day (timeline, favourites); off for a plain folder listing.
  // seq: when set, the viewer steps through these tiles instead of the whole timeline.
  // base / onopen / selectable: a shared album reads through its link, opens its own viewer and can't select.
  let { tiles, width, rowHeight, byDay = true, seq = false, base = '/api', onopen, selectable = true }: {
    tiles: Tile[]; width: number; rowHeight: number; byDay?: boolean; seq?: boolean;
    base?: string; onopen?: (id: number, e: MouseEvent) => void; selectable?: boolean;
  } = $props();

  const GAP = 2;
  let broken = $state<Record<number, boolean>>({});
  let loaded = $state<Record<number, boolean>>({});

  const groups = $derived.by(() => {
    if (width <= 0) return [];
    const gs: { day: string; items: Tile[] }[] = [];
    for (const t of tiles) {
      const day = byDay ? t.taken.slice(0, 10) : '';
      if (gs.at(-1)?.day !== day || !gs.length) gs.push({ day, items: [] });
      gs.at(-1)!.items.push(t);
    }
    return gs.map((g) => ({ ...g, rows: justify(g.items.map((t) => aspectOf(t.w, t.h)), width, rowHeight, GAP) }));
  });
  const ids = $derived(seq ? tiles.map((t) => t.id) : undefined);
  const fav = (t: Tile) => overrides[t.id]?.fav ?? t.fav;
  // the name must contain the text the tile shows (WCAG 2.5.3, voice control): duration, "No preview"
  const label = (t: Tile) =>
    `${t.type === 'video' ? 'Video' : 'Photo'} from ${formatDay(t.taken)}` +
    (t.type === 'video' && t.duration_ms ? `, ${formatDuration(t.duration_ms)}` : '') +
    (t.err || broken[t.id] ? ', No preview' : '') +
    (t.live ? ', Live' : '') +
    (fav(t) ? ', favourite' : '');

  const toggle = (id: number) => (selection.has(id) ? selection.delete(id) : selection.add(id));

  function click(t: Tile, e: MouseEvent) {
    if (selectable && (selection.size > 0 || e.shiftKey)) {
      e.preventDefault(); // picking, not opening
      if (!pressed) toggle(t.id);
      pressed = false;
      return;
    }
    if (pressed) { e.preventDefault(); pressed = false; return; } // the click that ends a long-press
    (onopen ?? ((id, ev) => openViewer(id, ev, ids)))(t.id, e);
  }

  // long-press on touch starts picking, like phone galleries
  let timer: ReturnType<typeof setTimeout> | undefined;
  let pressed = false;
  function down(t: Tile, e: PointerEvent) {
    if (!selectable || e.pointerType !== 'touch') return;
    timer = setTimeout(() => { toggle(t.id); pressed = true; navigator.vibrate?.(10); }, 500);
  }
  const cancel = () => clearTimeout(timer);
</script>

{#each groups as g (g.day)}
  {#if byDay}<h2 class="day">{formatDay(g.day)}</h2>{/if}
  {#each g.rows as row}
    <div class="row" style:height="{row[0].height}px">
      {#each row as box (g.items[box.index].id)}
        {@const t = g.items[box.index]}
        {@const picked = selection.has(t.id)}
        <div class="cell" class:picked class:picking={selection.size > 0} style:width="{box.width}px">
          <a class="tile" href="/photos/{t.id}" onclick={(e) => click(t, e)} aria-label={label(t)}
            onpointerdown={(e) => down(t, e)} onpointerup={cancel} onpointercancel={cancel} onpointermove={cancel}
            oncontextmenu={(e) => selectable && e.preventDefault()}>
            {#if t.err || broken[t.id]}
              <span class="ph err"><ImageOff aria-hidden="true" /><span>No preview</span></span>
            {:else}
              {#if t.ph && !loaded[t.id]}<span class="blur" style:background-image="url(data:image/webp;base64,{t.ph})" aria-hidden="true"></span>{/if}
              <img src={thumbURL(t, 's', base)} alt="" loading="lazy" decoding="async" class:in={loaded[t.id]}
                onload={() => (loaded[t.id] = true)} onerror={() => (broken[t.id] = true)} />
            {/if}
            {#if t.type === 'video'}<span class="badge br"><Play aria-hidden="true" />{formatDuration(t.duration_ms)}</span>{/if}
            {#if fav(t)}<span class="badge bl"><Heart aria-hidden="true" fill="currentColor" /></span>{/if}
            {#if t.live}<span class="badge tr">Live</span>{/if}
          </a>
          {#if selectable}
            <button class="check" onclick={() => toggle(t.id)} aria-pressed={picked} aria-label="Select {label(t)}">
              <Check aria-hidden="true" />
            </button>
          {/if}
        </div>
      {/each}
    </div>
  {/each}
{/each}

<style>
  .day { font: 600 var(--text-sm) var(--font-sans); margin: 0; padding: var(--space-6) 0 var(--space-3); }
  .row { display: flex; gap: 2px; margin-bottom: 2px; }
  .cell { position: relative; flex: none; height: 100%; background: var(--surface); }
  .tile { position: absolute; inset: 0; overflow: hidden; color: var(--viewer-text); display: block; -webkit-touch-callout: none;
    transition: inset var(--motion-fast) var(--motion-ease); }
  .tile img { position: relative; width: 100%; height: 100%; object-fit: cover; display: block; opacity: 0; transition: opacity var(--motion-base) var(--motion-ease); }
  .tile img.in { opacity: 1; }
  .blur { position: absolute; inset: -8px; background-size: cover; background-position: center; filter: blur(8px); }
  .ph.err { position: absolute; inset: 0; display: grid; place-content: center; justify-items: center; gap: var(--space-2);
    color: var(--text-muted); font: var(--text-xs) var(--font-sans); }
  /* own dark backing: readable on any photo and before the photo loads (BUG-007) */
  .badge { position: absolute; display: flex; align-items: center; gap: 4px; padding: 2px 6px; border-radius: var(--radius-pill);
    background: rgba(0, 0, 0, .6); color: var(--viewer-text); font: 600 var(--text-xs) var(--font-sans); }
  .badge :global(svg), .ph :global(svg) { width: 16px; height: 16px; }
  .br { right: 6px; bottom: 6px; }
  .bl { left: 6px; bottom: 6px; }
  .tr { right: 6px; top: 6px; letter-spacing: .04em; }
  /* select circle: on hover/focus, always while picking */
  .check { position: absolute; top: 2px; left: 2px; width: var(--touch-target); height: var(--touch-target); padding: 0; border: 0;
    background: none; cursor: pointer; display: grid; place-items: center; opacity: 0; transition: opacity var(--motion-fast); }
  .check :global(svg) { width: 16px; height: 16px; padding: 2px; border-radius: 50%; border: 2px solid var(--viewer-text);
    color: transparent; filter: drop-shadow(var(--shadow-on-photo)); box-sizing: content-box; }
  .cell:hover .check, .check:focus-visible, .picking .check { opacity: 1; }
  .picked .check :global(svg) { background: var(--accent); border-color: var(--accent); color: var(--on-accent); filter: none; }
  .picked .tile { inset: 8px; border-radius: var(--radius-sm); outline: 3px solid var(--accent); }
  @media (prefers-reduced-motion: reduce) { .tile img { transition: none; } }
</style>
