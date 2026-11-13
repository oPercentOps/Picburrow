<script lang="ts">
  import { page } from '$app/state';
  import { api } from '$lib/api';
  import { selection, overrides, picker, say, refresh } from '$lib/state.svelte';
  import { X, Heart, Archive, BookPlus, BookMinus, Star } from '@lucide/svelte';

  let busy = $state(false);
  let rating = $state(false); // the star menu is open
  const ids = () => [...selection];
  const inAlbum = $derived(page.route.id === '/albums/[id]');

  async function patch(change: { favorite?: boolean; archived?: boolean }, done: string) {
    busy = true;
    try {
      const list = ids();
      await api('/api/assets', { method: 'PATCH', body: { ids: list, ...change } });
      for (const id of list) overrides[id] = { ...overrides[id], ...('favorite' in change ? { fav: change.favorite } : { archived: change.archived }) };
      selection.clear();
      say(done);
    } catch (e) {
      say((e as Error).message);
    } finally {
      busy = false;
    }
  }
  async function removeFromAlbum() {
    busy = true;
    try {
      const n = selection.size;
      await api(`/api/albums/${page.params.id}/assets`, { method: 'DELETE', body: { ids: ids() } });
      selection.clear();
      refresh.n++;
      say(`Removed ${n} from this album. The files stay on disk.`);
    } catch (e) {
      say((e as Error).message);
    } finally {
      busy = false;
    }
  }
  async function rate(n: number) {
    rating = false;
    busy = true;
    try {
      const count = selection.size;
      await api('/api/assets', { method: 'PATCH', body: { ids: ids(), rating: n } });
      selection.clear();
      say(n ? `Rated ${count} with ${n} ${n === 1 ? 'star' : 'stars'}` : `Cleared the rating of ${count}`);
    } catch (e) {
      say((e as Error).message);
    } finally {
      busy = false;
    }
  }
  const allFav = () => ids().every((id) => overrides[id]?.fav === true);
  const n = $derived(selection.size);
</script>

<svelte:window onkeydown={(e) => { if (e.key !== 'Escape' || page.state.viewer) return; rating ? (rating = false) : selection.clear(); }} />

<div class="bar" role="toolbar" aria-label="Selected photos">
  <button class="btn icon-btn ghost" onclick={() => selection.clear()} aria-label="Clear selection"><X aria-hidden="true" /></button>
  <span class="count" aria-live="polite">{n} selected</span>
  <div class="actions">
    <button class="btn ghost" disabled={busy} onclick={() => patch({ favorite: !allFav() }, allFav() ? `Removed ${n} from favourites` : `Added ${n} to favourites`)}>
      <Heart aria-hidden="true" /><span class="lbl">Favourite</span></button>
    <div class="rate">
      <button class="btn ghost" disabled={busy} aria-expanded={rating} aria-controls="rate-menu" onclick={() => (rating = !rating)}>
        <Star aria-hidden="true" /><span class="lbl">Rate</span></button>
      {#if rating}
        <div class="menu" id="rate-menu" role="group" aria-label="Rate the selected photos">
          {#each [1, 2, 3, 4, 5] as n}<button class="btn ghost" onclick={() => rate(n)}>{'★'.repeat(n)}<span class="sr-only"> {n} {n === 1 ? 'star' : 'stars'}</span></button>{/each}
          <button class="btn ghost" onclick={() => rate(0)}>No rating</button>
        </div>
      {/if}
    </div>
    <button class="btn ghost" disabled={busy} onclick={() => { picker.ids = ids(); }}>
      <BookPlus aria-hidden="true" /><span class="lbl">Add to album</span></button>
    {#if inAlbum}
      <button class="btn ghost" disabled={busy} onclick={removeFromAlbum}><BookMinus aria-hidden="true" /><span class="lbl">Remove from album</span></button>
    {:else}
      <button class="btn ghost" disabled={busy} onclick={() => patch({ archived: true }, `Archived ${n}. Find them under Archive.`)}>
        <Archive aria-hidden="true" /><span class="lbl">Archive</span></button>
    {/if}
  </div>
</div>

<style>
  .bar { position: fixed; z-index: 20; inset: 0 0 auto; display: flex; align-items: center; gap: var(--space-3); min-height: var(--header);
    padding: var(--space-2) var(--space-3); padding-top: max(var(--space-2), env(safe-area-inset-top)); background: var(--surface);
    border-bottom: 1px solid var(--border); box-shadow: var(--shadow-pop); }
  .count { font: 600 var(--text-base) var(--font-sans); flex: 1; }
  .actions { display: flex; gap: var(--space-1); }
  .bar :global(svg) { width: 20px; height: 20px; }
  .rate { position: relative; }
  .menu { position: absolute; top: calc(100% + var(--space-2)); right: 0; display: grid; min-width: 160px; padding: var(--space-2);
    background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius-md); box-shadow: var(--shadow-pop); }
  .menu .btn { justify-content: flex-start; letter-spacing: .1em; }
  @media (max-width: 767px) {
    .lbl { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); white-space: nowrap; } /* icons only on phones */
    .actions .btn { width: var(--touch-target); padding: 0; }
  }
</style>
