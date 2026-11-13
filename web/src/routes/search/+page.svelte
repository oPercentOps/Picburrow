<script lang="ts">
  import { t } from '$lib/i18n';
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import { api, type Tile } from '$lib/api';
  import { overrides } from '$lib/state.svelte';
  import TileGrid from '$lib/TileGrid.svelte';
  import { Search } from '@lucide/svelte';

  // the query lives in the URL, so Back and reload keep it
  const params = $derived(page.url.searchParams);
  let q = $state(page.url.searchParams.get('q') ?? '');
  let from = $state(page.url.searchParams.get('from') ?? '');
  let to = $state(page.url.searchParams.get('to') ?? '');
  let type = $state(page.url.searchParams.get('type') ?? '');
  let favorite = $state(page.url.searchParams.get('favorite') === '1');
  let rating = $state(page.url.searchParams.get('rating') ?? '');
  let tiles = $state<Tile[] | null>(null);
  let error = $state('');
  let width = $state(0);
  let wide = $state(false);

  const hasQuery = $derived([...params.keys()].length > 0);
  $effect(() => {
    wide = matchMedia('(min-width: 768px)').matches;
    if (!hasQuery) { tiles = null; return; }
    error = '';
    api<Tile[]>('/api/search?' + params.toString()).then((t) => (tiles = t)).catch((e) => (error = e.message));
  });

  function submit(e: SubmitEvent) {
    e.preventDefault();
    const p = new URLSearchParams();
    if (q.trim()) p.set('q', q.trim());
    if (from) p.set('from', from);
    if (to) p.set('to', to);
    if (type) p.set('type', type);
    if (favorite) p.set('favorite', '1');
    if (rating) p.set('rating', rating);
    goto('/search?' + p.toString(), { keepFocus: true });
  }
  const shown = $derived(tiles?.filter((t) => !overrides[t.id]?.archived) ?? []);
</script>

<svelte:head><title>{t('Search')} · Picburrow</title></svelte:head>

<div class="page">
  <h1>{t('Search')}</h1>
  <form onsubmit={submit} role="search" aria-label={t('Photos')}>
    <div class="field grow">
      <label for="sq">{t('Words in the file name, folder or camera')}</label>
      <input id="sq" type="search" bind:value={q} placeholder={t('beach, 2023, Pixel 8')} />
    </div>
    <div class="field"><label for="sf">{t('From')}</label><input id="sf" type="date" bind:value={from} /></div>
    <div class="field"><label for="st">{t('To')}</label><input id="st" type="date" bind:value={to} /></div>
    <div class="field">
      <label for="sy">{t('Kind')}</label>
      <select id="sy" bind:value={type}><option value="">{t('Photos and videos')}</option><option value="image">{t('Photos')}</option><option value="video">{t('Videos')}</option></select>
    </div>
    <div class="field">
      <label for="sr">{t('Stars')}</label>
      <select id="sr" bind:value={rating}><option value="">{t('Any')}</option>{#each [1, 2, 3, 4, 5] as n}<option value={String(n)}>{t(n === 1 ? '1+ star' : '{n}+ stars', { n })}</option>{/each}</select>
    </div>
    <label class="check"><input type="checkbox" bind:checked={favorite} /> {t('Favourites only')}</label>
    <button class="btn primary"><Search aria-hidden="true" />{t('Search')}</button>
  </form>

  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else if tiles}
    <p class="muted" role="status">{shown.length === 2000 ? t('Showing the first 2,000 results. Narrow the search to see the rest.') : t('{n} found', { n: shown.length.toLocaleString() })}</p>
    {#if shown.length === 0}
      <div class="empty"><Search aria-hidden="true" /><strong>{t('Nothing matches')}</strong><span class="muted">{t('Try fewer words or a wider date range.')}</span></div>
    {:else}
      <div bind:clientWidth={width}><TileGrid tiles={shown} {width} rowHeight={wide ? 220 : 120} seq /></div>
    {/if}
  {/if}
</div>

<style>
  .page { padding: var(--space-5) var(--space-1) var(--space-7); }
  h1, p { margin: 0 var(--space-3) var(--space-4); }
  form { display: flex; flex-wrap: wrap; gap: var(--space-4); align-items: end; padding: 0 var(--space-3) var(--space-5); }
  .grow { flex: 1 1 240px; }
  .field input, select { height: 40px; padding: 0 var(--space-4); border: 1px solid var(--border-input); border-radius: var(--radius-sm);
    background: var(--bg); color: var(--text); font: var(--text-base) var(--font-sans); }
  select:focus-visible { border-color: var(--accent); }
  .check { display: flex; gap: var(--space-3); align-items: center; font: var(--text-sm) var(--font-sans); min-height: 40px; }
  .check input { width: 20px; height: 20px; accent-color: var(--accent); }
  form :global(svg) { width: 18px; height: 18px; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } h1, p, form { margin-left: 0; margin-right: 0; padding-left: 0; padding-right: 0; } }
</style>
