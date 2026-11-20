<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, type Tile } from '$lib/api';
  import { say, refresh } from '$lib/state.svelte';
  import TileGrid from '$lib/TileGrid.svelte';
  import { Trash2 } from '@lucide/svelte';

  // Photos deleted here or whose file disappeared from disk. Select deleted ones to restore them; putting a vanished
  // file back (same or another folder) brings it back with its albums and favourites; after 30 days they go for good.
  let tiles = $state<Tile[] | null>(null);
  let error = $state('');
  let width = $state(0);
  let wide = $state(false);
  const load = () => api<Tile[]>('/api/trash').then((t) => (tiles = t)).catch((e) => (error = e.message));
  $effect(() => { refresh.n; load(); wide = matchMedia('(min-width: 768px)').matches; }); // reload after Restore

  async function empty() {
    if (!confirm(t('Forget these {n} photos now? Their albums and favourites go too, and the files deleted here leave the disk for good.', { n: tiles!.length }))) return;
    const r = await api<{ removed: number }>('/api/trash', { method: 'DELETE' });
    say(t('Forgot {n} photos', { n: r.removed }));
    load();
  }
</script>

<svelte:head><title>{t('Trash')} · Picburrow</title></svelte:head>

<div class="page">
  <header>
    <div>
      <h1>{t('Trash')}</h1>
      <p class="muted">{t('Photos you deleted, or whose file is no longer on disk. Select deleted ones to restore them; put a vanished file back and the next scan restores it, albums and favourites included. After 30 days they go for good.')}</p>
    </div>
    {#if tiles?.length}<button class="btn secondary" onclick={empty}><Trash2 aria-hidden="true" />{t('Forget them now')}</button>{/if}
  </header>
  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else if tiles && tiles.length === 0}
    <div class="empty"><Trash2 aria-hidden="true" /><strong>{t('Trash is empty')}</strong><span class="muted">{t('Every photo in the library is still on disk.')}</span></div>
  {:else if tiles}
    <div bind:clientWidth={width}><TileGrid {tiles} {width} rowHeight={wide ? 220 : 120} byDay={false} seq /></div>
  {/if}
</div>

<style>
  .page { padding: var(--space-5) var(--space-1) var(--space-7); }
  header { display: flex; flex-wrap: wrap; gap: var(--space-4); align-items: end; justify-content: space-between; padding: 0 var(--space-3) var(--space-5); }
  header p { margin: var(--space-2) 0 0; max-width: 60ch; }
  header :global(svg) { width: 18px; height: 18px; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } header { padding: 0 0 var(--space-5); } }
</style>
