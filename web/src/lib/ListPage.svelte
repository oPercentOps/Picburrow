<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, type Tile } from '$lib/api';
  import TileGrid from '$lib/TileGrid.svelte';
  import type { Component } from 'svelte';

  // A page that shows one flat list of photos (favourites, archive). keep() drops tiles changed away in this session.
  let { title, query, icon: Icon, empty, hint, keep }: {
    title: string; query: string; icon: Component; empty: string; hint: string; keep: (t: Tile) => boolean;
  } = $props();

  let tiles = $state<Tile[] | null>(null);
  let error = $state('');
  let width = $state(0);
  let wide = $state(false);

  $effect(() => {
    api<Tile[]>(query).then((t) => (tiles = t)).catch((e) => (error = e.message));
    const mq = matchMedia('(min-width: 768px)');
    wide = mq.matches;
  });
  const shown = $derived(tiles?.filter(keep) ?? []);
</script>

<svelte:head><title>{title} · Picburrow</title></svelte:head>

<div class="page">
  <header><h1>{title}</h1>{#if tiles}<span class="muted">{t('{n} items', { n: shown.length.toLocaleString() })}</span>{/if}</header>
  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else if tiles && shown.length === 0}
    <div class="empty"><Icon aria-hidden="true" /><strong>{empty}</strong><span class="muted">{hint}</span></div>
  {:else if tiles}
    <div bind:clientWidth={width}><TileGrid tiles={shown} {width} rowHeight={wide ? 220 : 120} seq /></div>
  {/if}
</div>

<style>
  .page { padding: 0 var(--space-1) var(--space-6); }
  header { display: flex; align-items: baseline; gap: var(--space-4); padding: var(--space-5) var(--space-4) 0; }
  @media (min-width: 768px) {
    .page { padding: 0 var(--space-6) var(--space-7); }
    header { padding: var(--space-6) 0 0; }
  }
</style>
