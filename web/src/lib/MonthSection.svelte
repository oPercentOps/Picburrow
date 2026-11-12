<script lang="ts">
  import { api, thumbURL, formatDay, formatDuration, type Tile } from '$lib/api';
  import { justify, aspectOf } from '$lib/justify';
  import { Play, Heart, ImageOff } from '@lucide/svelte';

  let { month, count, width, rowHeight, onopen }: {
    month: string; count: number; width: number; rowHeight: number; onopen: (id: number, e: MouseEvent) => void;
  } = $props();

  const GAP = 2;
  let el: HTMLElement;
  let near = $state(false);
  let tiles = $state<Tile[] | null>(null);
  let failed = $state(false);
  let measured = $state(0);
  let broken = $state<Record<number, boolean>>({});

  // only months within ~1.5 screens are mounted: keeps a 50k-photo timeline light on a phone
  $effect(() => {
    const io = new IntersectionObserver(([e]) => (near = e.isIntersecting), { rootMargin: '150% 0px' });
    io.observe(el);
    return () => io.disconnect();
  });

  $effect(() => {
    if (near && !tiles && !failed) {
      api<Tile[]>(`/api/timeline?month=${month}`).then((t) => (tiles = t)).catch(() => (failed = true));
    }
  });

  const days = $derived.by(() => {
    if (!tiles || width <= 0) return [];
    const groups: { day: string; items: Tile[] }[] = [];
    for (const t of tiles) {
      const day = t.taken.slice(0, 10);
      if (groups.at(-1)?.day !== day) groups.push({ day, items: [] });
      groups.at(-1)!.items.push(t);
    }
    return groups.map((g) => ({
      ...g,
      rows: justify(g.items.map((t) => aspectOf(t.w, t.h)), width, rowHeight, GAP)
    }));
  });

  // placeholder height before the month is loaded, or after it scrolls far away
  const estimate = $derived(Math.ceil((count * 1.33 * rowHeight) / Math.max(width, 1)) * (rowHeight + GAP) + 48);
</script>

<section bind:this={el} aria-label={month} style:min-height="{near && tiles ? 0 : measured || estimate}px">
  {#if near && tiles}
    <div bind:offsetHeight={measured}>
      {#each days as d (d.day)}
        <h2 class="day">{formatDay(d.day)}</h2>
        {#each d.rows as row}
          <div class="row" style:height="{row[0].height}px">
            {#each row as box (d.items[box.index].id)}
              {@const t = d.items[box.index]}
              <a class="tile" href="/photos/{t.id}" style:width="{box.width}px" onclick={(e) => onopen(t.id, e)}
                aria-label="{t.type === 'video' ? 'Video' : 'Photo'} from {formatDay(t.taken)}">
                {#if t.err || broken[t.id]}
                  <span class="ph err"><ImageOff aria-hidden="true" /><span>No preview</span></span>
                {:else}
                  <img src={thumbURL(t, 's')} alt="" loading="lazy" decoding="async" onerror={() => (broken[t.id] = true)} />
                {/if}
                {#if t.type === 'video'}<span class="badge br"><Play aria-hidden="true" />{formatDuration(t.duration_ms)}</span>{/if}
                {#if t.fav}<span class="badge bl"><Heart aria-hidden="true" fill="currentColor" /><span class="sr-only">Favourite</span></span>{/if}
              </a>
            {/each}
          </div>
        {/each}
      {/each}
    </div>
  {:else if failed}
    <p class="muted pad">This month didn't load. <button class="btn ghost" onclick={() => (failed = false)}>Try again</button></p>
  {/if}
</section>

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
  .pad { padding: var(--space-5) 0; }
</style>
