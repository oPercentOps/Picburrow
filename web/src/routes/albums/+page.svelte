<script lang="ts">
  import { goto } from '$app/navigation';
  import { api, thumbURL, type Album } from '$lib/api';
  import { BookImage, Plus, Link } from '@lucide/svelte';

  let albums = $state<Album[] | null>(null);
  let error = $state('');
  api<Album[]>('/api/albums').then((a) => (albums = a)).catch((e) => (error = e.message));

  async function create() {
    const name = prompt('Name the new album');
    if (!name?.trim()) return;
    try {
      const a = await api<Album>('/api/albums', { method: 'POST', body: { name } });
      goto(`/albums/${a.id}`);
    } catch (e) {
      error = (e as Error).message;
    }
  }
</script>

<svelte:head><title>Albums · Homeroll</title></svelte:head>

<div class="page">
  <header>
    <h1>Albums</h1>
    <button class="btn secondary" onclick={create}><Plus aria-hidden="true" />New album</button>
  </header>
  {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
  {#if albums && albums.length === 0}
    <div class="empty">
      <BookImage aria-hidden="true" />
      <strong>No albums yet</strong>
      <span class="muted">Select photos anywhere (long-press on a phone, or the circle on a photo), then choose Add to album.</span>
    </div>
  {:else if albums}
    <ul class="grid">
      {#each albums as a (a.id)}
        <li>
          <a href="/albums/{a.id}" aria-label="{a.name}, {a.count} {a.count === 1 ? 'item' : 'items'}{a.shared ? ', shared' : ''}">
            <span class="cover">{#if a.cover}<img src={thumbURL(a.cover, 's')} alt="" loading="lazy" />{:else}<BookImage aria-hidden="true" />{/if}</span>
            <span class="name">{a.name}</span>
            <span class="muted">{a.count} {a.count === 1 ? 'item' : 'items'}{#if a.shared} · <Link aria-hidden="true" class="inline" /> shared{/if}</span>
          </a>
        </li>
      {/each}
    </ul>
  {/if}
</div>

<style>
  .page { padding: var(--space-5) var(--space-4) var(--space-7); }
  header { display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); margin-bottom: var(--space-5); }
  header :global(svg) { width: 18px; height: 18px; }
  .grid { list-style: none; margin: 0; padding: 0; display: grid; gap: var(--space-5) var(--space-4); grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); }
  .grid a { display: grid; gap: var(--space-1); text-decoration: none; }
  .cover { aspect-ratio: 1; border-radius: var(--radius-md); overflow: hidden; background: var(--surface); display: grid; place-items: center; color: var(--text-muted); }
  .cover img { width: 100%; height: 100%; object-fit: cover; }
  .cover :global(svg) { width: 40px; height: 40px; }
  .name { font: 600 var(--text-sm) var(--font-sans); overflow-wrap: anywhere; }
  .muted :global(.inline) { width: 12px; height: 12px; vertical-align: -1px; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } .grid { grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); } }
</style>
