<script lang="ts">
  import { api, formatMonth, type Bucket } from '$lib/api';
  import MonthSection from '$lib/MonthSection.svelte';
  import IndexingBanner from '$lib/IndexingBanner.svelte';
  import Scrubber from '$lib/Scrubber.svelte';
  import { Images } from '@lucide/svelte';

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

  $effect(() => {
    const mq = matchMedia('(min-width: 768px)');
    wide = mq.matches;
    const on = () => (wide = mq.matches);
    mq.addEventListener('change', on);
    return () => mq.removeEventListener('change', on);
  });
</script>

<svelte:head><title>Photos</title></svelte:head>

<div class="page">
  <header><h1>Photos</h1></header>
  <IndexingBanner onscandone={load} />
  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else if buckets && buckets.length === 0}
    <div class="empty">
      <Images aria-hidden="true" />
      <strong>No photos here yet</strong>
      <span class="muted">They'll show up as soon as the first scan finds some. Check the folder under Admin › Libraries if nothing appears.</span>
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
  header { padding: var(--space-5) var(--space-4) 0; }
  @media (min-width: 768px) {
    .page { padding: 0 var(--space-8) var(--space-7) var(--space-6); } /* right padding = the scrubber's column */
    header { padding: var(--space-6) 0 0; }
  }
</style>
