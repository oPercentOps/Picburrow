<script lang="ts">
  import { thumbURL, formatDay, formatDuration, type Tile } from '$lib/api';
  import { justify, aspectOf } from '$lib/justify';
  import { overrides, openViewer } from '$lib/state.svelte';
  import { Play, Heart, ImageOff } from '@lucide/svelte';

  // byDay: date headers per day (timeline, favourites); off for a plain folder listing.
  // seq: when set, the viewer steps through these tiles instead of the whole timeline.
  let { tiles, width, rowHeight, byDay = true, seq = false }: {
    tiles: Tile[]; width: number; rowHeight: number; byDay?: boolean; seq?: boolean;
  } = $props();

  const GAP = 2;
  let broken = $state<Record<number, boolean>>({});

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
</script>

{#each groups as g (g.day)}
  {#if byDay}<h2 class="day">{formatDay(g.day)}</h2>{/if}
  {#each g.rows as row}
    <div class="row" style:height="{row[0].height}px">
      {#each row as box (g.items[box.index].id)}
        {@const t = g.items[box.index]}
        <a class="tile" href="/photos/{t.id}" style:width="{box.width}px" onclick={(e) => openViewer(t.id, e, ids)}
          aria-label="{t.type === 'video' ? 'Video' : 'Photo'} from {formatDay(t.taken)}{fav(t) ? ', favourite' : ''}">
          {#if t.err || broken[t.id]}
            <span class="ph err"><ImageOff aria-hidden="true" /><span>No preview</span></span>
          {:else}
            <img src={thumbURL(t, 's')} alt="" loading="lazy" decoding="async" onerror={() => (broken[t.id] = true)} />
          {/if}
          {#if t.type === 'video'}<span class="badge br"><Play aria-hidden="true" />{formatDuration(t.duration_ms)}</span>{/if}
          {#if fav(t)}<span class="badge bl"><Heart aria-hidden="true" fill="currentColor" /></span>{/if}
        </a>
      {/each}
    </div>
  {/each}
{/each}

<style>
  .day { font: 600 var(--text-sm) var(--font-sans); margin: 0; padding: var(--space-6) 0 var(--space-3); }
  .row { display: flex; gap: 2px; margin-bottom: 2px; }
  .tile { position: relative; flex: none; height: 100%; overflow: hidden; background: var(--surface); color: var(--viewer-text); display: block; }
  .tile img { width: 100%; height: 100%; object-fit: cover; display: block; }
  .ph.err { position: absolute; inset: 0; display: grid; place-content: center; justify-items: center; gap: var(--space-2);
    color: var(--text-muted); font: var(--text-xs) var(--font-sans); }
  .badge { position: absolute; display: flex; align-items: center; gap: 4px; font: 600 var(--text-xs) var(--font-sans); filter: drop-shadow(var(--shadow-on-photo)); }
  .badge :global(svg), .ph :global(svg) { width: 16px; height: 16px; }
  .br { right: 6px; bottom: 6px; }
  .bl { left: 6px; bottom: 6px; }
</style>
