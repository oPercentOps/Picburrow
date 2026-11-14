<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, formatMonth, today, type Bucket, type Tile } from '$lib/api';
  import MonthSection from '$lib/MonthSection.svelte';
  import IndexingBanner from '$lib/IndexingBanner.svelte';
  import Scrubber from '$lib/Scrubber.svelte';
  import { History, Images, MonitorPlay } from '@lucide/svelte';
  import { SOURCE_URL } from '$lib/source';

  let buckets = $state<Bucket[] | null>(null);
  let error = $state('');
  let width = $state(0);
  let wide = $state(false);
  let generation = $state(0); // bumps after a scan so loaded months refetch

  function load() {
    api<Bucket[]>('/api/timeline/buckets')
      .then((b) => { buckets = b; generation++; })
      .catch((e) => (error = e.message));
  }
  load();

  let memories = $state(0); // photos from this day in earlier years
  api<Tile[]>(`/api/assets?onthisday=${today()}`).then((t) => (memories = t.length)).catch(() => {});

  $effect(() => {
    const mq = matchMedia('(min-width: 768px)');
    wide = mq.matches;
    const on = () => (wide = mq.matches);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  });
</script>

<svelte:head><title>{t('Photos')} · Picburrow</title></svelte:head>

<div class="page">
  <header>
    <h1>{t('Photos')}</h1>
    {#if buckets?.length}
      <div class="actions">
        {#if memories}<a class="btn secondary" href="/onthisday"><History aria-hidden="true" />{t('On this day')}</a>{/if}
        <a class="btn secondary" href="/slideshow"><MonitorPlay aria-hidden="true" />{t('Slideshow')}</a>
      </div>
    {/if}
  </header>
  <IndexingBanner onscandone={load} />
  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else if buckets && buckets.length === 0}
    <div class="empty">
      <Images aria-hidden="true" />
      <strong>{t('No photos yet')}</strong>
      <span class="muted">{t('They appear here as soon as the first scan finds them. If nothing shows up, check the folder under Libraries.')}</span>
      <a href="{SOURCE_URL}/blob/main/docs/phone-to-folder.md" rel="noreferrer">{t('How to get photos from a phone into the folder')}</a>
    </div>
  {:else if buckets}
    <div bind:clientWidth={width}>
      {#each buckets as b (b.month + generation)}
        <h2 class="sr-only">{formatMonth(b.month)}</h2>
        <MonthSection month={b.month} count={b.count} {width} rowHeight={wide ? 220 : 120} />
      {/each}
    </div>
    {#if buckets.length > 1}<Scrubber {buckets} />{/if}
  {/if}
</div>

<style>
  .page { padding: 0 var(--space-1) var(--space-6); }
  header { display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: var(--space-4); padding: var(--space-5) var(--space-4) 0; }
  .actions { display: flex; flex-wrap: wrap; gap: var(--space-2); }
  header a :global(svg) { width: 20px; height: 20px; }
  header a { text-decoration: none; }
  @media (min-width: 768px) {
    .page { padding: 0 var(--space-8) var(--space-7) var(--space-6); } /* right padding = the scrubber's column */
    header { padding: var(--space-6) 0 0; }
  }
</style>
