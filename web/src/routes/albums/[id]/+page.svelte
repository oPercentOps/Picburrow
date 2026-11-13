<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api, formatWhen, type Album } from '$lib/api';
  import { refresh, say } from '$lib/state.svelte';
  import TileGrid from '$lib/TileGrid.svelte';
  import ShareDialog from '$lib/ShareDialog.svelte';
  import { Share2, Pencil, Trash2, BookImage } from '@lucide/svelte';

  let album = $state<Album | null>(null);
  let error = $state('');
  let width = $state(0);
  let wide = $state(false);
  let sharing = $state(false);

  $effect(() => {
    refresh.n; // reload after "Remove from album"
    api<Album>(`/api/albums/${page.params.id}`).then((a) => (album = a)).catch((e) => (error = e.message));
    wide = matchMedia('(min-width: 768px)').matches;
  });

  async function rename() {
    const name = prompt('Rename the album', album!.name);
    if (!name?.trim() || name === album!.name) return;
    try {
      album = await api<Album>(`/api/albums/${album!.id}`, { method: 'PATCH', body: { name } });
    } catch (e) {
      say((e as Error).message);
    }
  }
  async function remove() {
    if (!confirm(`Delete the album "${album!.name}"? The photos stay in your library and on disk.${album!.links?.length ? ' Its share links stop working.' : ''}`)) return;
    await api(`/api/albums/${album!.id}`, { method: 'DELETE' });
    say(`Deleted the album ${album!.name}`);
    goto('/albums');
  }
</script>

<svelte:head><title>{album?.name ?? 'Album'} · Homeroll</title></svelte:head>

<div class="page">
  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else if album}
    <header>
      <div>
        <a href="/albums" class="muted back">Albums</a>
        <h1>{album.name}</h1>
        <span class="muted">{album.count} {album.count === 1 ? 'item' : 'items'} · updated {formatWhen(album.updated_at)}</span>
      </div>
      <div class="actions">
        <button class="btn secondary" onclick={() => (sharing = true)}><Share2 aria-hidden="true" />Share{#if album.links?.length}{' '}({album.links.length}){/if}</button>
        <button class="btn ghost icon-btn" onclick={rename} aria-label="Rename album"><Pencil aria-hidden="true" /></button>
        <button class="btn ghost icon-btn" onclick={remove} aria-label="Delete album"><Trash2 aria-hidden="true" /></button>
      </div>
    </header>
    {#if album.assets?.length}
      <div bind:clientWidth={width}><TileGrid tiles={album.assets} {width} rowHeight={wide ? 220 : 120} seq /></div>
    {:else}
      <div class="empty"><BookImage aria-hidden="true" /><strong>This album is empty</strong>
        <span class="muted">Select photos in the timeline or a folder, then choose Add to album.</span></div>
    {/if}
    {#if sharing}<ShareDialog {album} onchange={(a) => (album = a)} onclose={() => (sharing = false)} />{/if}
  {/if}
</div>

<style>
  .page { padding: var(--space-5) var(--space-1) var(--space-7); }
  header { display: flex; flex-wrap: wrap; align-items: end; justify-content: space-between; gap: var(--space-4); padding: 0 var(--space-3) var(--space-5); }
  h1 { overflow-wrap: anywhere; }
  .back { text-decoration: none; }
  .back:hover { text-decoration: underline; }
  .actions { display: flex; gap: var(--space-2); }
  .actions :global(svg) { width: 18px; height: 18px; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } header { padding: 0 0 var(--space-5); } }
</style>
