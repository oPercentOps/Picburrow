<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, type Tile } from '$lib/api';
  import { overrides } from '$lib/state.svelte';
  import TileGrid from '$lib/TileGrid.svelte';

  let { month, count, width, rowHeight }: { month: string; count: number; width: number; rowHeight: number } = $props();

  let el: HTMLElement;
  let near = $state(false);
  let tiles = $state<Tile[] | null>(null);
  let failed = $state(false);
  let measured = $state(0);

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

  const shown = $derived(tiles?.filter((t) => !overrides[t.id]?.archived) ?? []);
  // placeholder height before the month is loaded, or after it scrolls far away
  const estimate = $derived(Math.ceil((count * 1.33 * rowHeight) / Math.max(width, 1)) * (rowHeight + 2) + 48);
</script>

<section bind:this={el} id="m-{month}" aria-label={month} style:min-height="{near && tiles ? 0 : measured || estimate}px">
  {#if near && tiles}
    <div bind:offsetHeight={measured}><TileGrid tiles={shown} {width} {rowHeight} /></div>
  {:else if failed}
    <p class="muted pad">{t("This month didn't load.")} <button class="btn ghost" onclick={() => (failed = false)}>{t('Try again')}</button></p>
  {/if}
</section>

<style>
  .pad { padding: var(--space-5) 0; }
</style>
