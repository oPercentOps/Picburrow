<script lang="ts">
  import { api, type Tile } from '$lib/api';
  import { say } from '$lib/state.svelte';
  import TileGrid from '$lib/TileGrid.svelte';
  import { Trash2 } from '@lucide/svelte';

  // Photos whose file disappeared from disk. Putting the file back (same or another folder) brings them back
  // with their albums and favourites; after 30 days they leave the library for good.
  let tiles = $state<Tile[] | null>(null);
  let error = $state('');
  let width = $state(0);
  let wide = $state(false);
  const load = () => api<Tile[]>('/api/trash').then((t) => (tiles = t)).catch((e) => (error = e.message));
  $effect(() => { load(); wide = matchMedia('(min-width: 768px)').matches; });

  async function empty() {
    if (!confirm(`Forget these ${tiles!.length} photos now? Their albums and favourites go too. No file is deleted: they are already gone from disk.`)) return;
    const r = await api<{ removed: number }>('/api/trash', { method: 'DELETE' });
    say(`Forgot ${r.removed} photos`);
    load();
  }
</script>

<svelte:head><title>Trash · Picburrow</title></svelte:head>

<div class="page">
  <header>
    <div>
      <h1>Trash</h1>
      <p class="muted">Photos whose file is no longer on disk. Put a file back and the next scan restores it, albums and favourites included. After 30 days they leave the library.</p>
    </div>
    {#if tiles?.length}<button class="btn secondary" onclick={empty}><Trash2 aria-hidden="true" />Forget them now</button>{/if}
  </header>
  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else if tiles && tiles.length === 0}
    <div class="empty"><Trash2 aria-hidden="true" /><strong>Trash is empty</strong><span class="muted">Every photo in the library is still on disk.</span></div>
  {:else if tiles}
    <div bind:clientWidth={width}><TileGrid {tiles} {width} rowHeight={wide ? 220 : 120} byDay={false} seq selectable={false} /></div>
  {/if}
</div>

<style>
  .page { padding: var(--space-5) var(--space-1) var(--space-7); }
  header { display: flex; flex-wrap: wrap; gap: var(--space-4); align-items: end; justify-content: space-between; padding: 0 var(--space-3) var(--space-5); }
  header p { margin: var(--space-2) 0 0; max-width: 60ch; }
  header :global(svg) { width: 18px; height: 18px; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } header { padding: 0 0 var(--space-5); } }
</style>
