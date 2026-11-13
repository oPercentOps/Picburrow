<script lang="ts">
  import { api, thumbURL, type Album } from '$lib/api';
  import { picker, selection, say } from '$lib/state.svelte';
  import { BookImage } from '@lucide/svelte';

  let dialog: HTMLDialogElement;
  let albums = $state<Album[] | null>(null);
  let name = $state('');
  let error = $state('');
  let busy = $state(false);

  $effect(() => {
    if (picker.ids) {
      albums = null;
      name = '';
      error = '';
      dialog.showModal();
      api<Album[]>('/api/albums').then((a) => (albums = a)).catch((e) => (error = e.message));
    } else if (dialog.open) dialog.close();
  });

  async function addTo(albumId: number, albumName: string) {
    busy = true;
    try {
      const ids = picker.ids!;
      await api(`/api/albums/${albumId}/assets`, { method: 'POST', body: { ids } });
      say(`Added ${ids.length} to ${albumName}`);
      selection.clear();
      picker.ids = null;
    } catch (e) {
      error = (e as Error).message;
    } finally {
      busy = false;
    }
  }
  async function create(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    try {
      const ids = picker.ids!;
      const a = await api<Album>('/api/albums', { method: 'POST', body: { name, ids } });
      say(`Made the album ${a.name} with ${ids.length} ${ids.length === 1 ? 'photo' : 'photos'}`);
      selection.clear();
      picker.ids = null;
    } catch (err) {
      error = (err as Error).message;
    } finally {
      busy = false;
    }
  }
</script>

<dialog bind:this={dialog} aria-labelledby="picker-title" onclose={() => (picker.ids = null)}>
  <h2 id="picker-title">Add {picker.ids?.length ?? 0} to an album</h2>
  <form onsubmit={create} class="new">
    <div class="field">
      <label for="new-album">New album</label>
      <input id="new-album" bind:value={name} placeholder="Summer 2026" maxlength="200" required />
    </div>
    <button class="btn primary" disabled={busy}>Create</button>
  </form>
  {#if albums?.length}
    <h3>Or add to</h3>
    <ul>
      {#each albums as a (a.id)}
        <li>
          <button class="pick" disabled={busy} onclick={() => addTo(a.id, a.name)}>
            <span class="cover">{#if a.cover}<img src={thumbURL(a.cover, 's')} alt="" />{:else}<BookImage aria-hidden="true" />{/if}</span>
            <span class="name">{a.name}</span><span class="muted">{a.count}</span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
  {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
  <div class="foot"><button class="btn secondary" onclick={() => (picker.ids = null)}>Cancel</button></div>
</dialog>

<style>
  dialog { width: min(100% - 2 * var(--space-5), 480px); max-height: 80dvh; padding: var(--space-6); border: 0; border-radius: var(--radius-lg);
    background: var(--bg); color: var(--text); box-shadow: var(--shadow-pop); }
  dialog::backdrop { background: rgba(0, 0, 0, .5); }
  h2 { margin: 0 0 var(--space-5); }
  h3 { font: 600 var(--text-sm) var(--font-sans); color: var(--text-muted); margin: var(--space-6) 0 var(--space-3); }
  .new { display: flex; gap: var(--space-3); align-items: end; }
  .new .field { flex: 1; }
  ul { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--space-1); }
  .pick { width: 100%; display: flex; align-items: center; gap: var(--space-4); padding: var(--space-2); border: 0; border-radius: var(--radius-md);
    background: none; color: inherit; font: var(--text-base) var(--font-sans); text-align: left; cursor: pointer; min-height: var(--touch-target); }
  .pick:hover { background: var(--surface); }
  .cover { width: 40px; height: 40px; border-radius: var(--radius-sm); overflow: hidden; background: var(--surface); display: grid; place-items: center; flex: none; }
  .cover img { width: 100%; height: 100%; object-fit: cover; }
  .cover :global(svg) { width: 20px; height: 20px; color: var(--text-muted); }
  .name { flex: 1; overflow-wrap: anywhere; }
  .foot { display: flex; justify-content: flex-end; margin-top: var(--space-6); }
</style>
