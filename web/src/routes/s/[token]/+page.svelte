<script lang="ts">
  import { t } from '$lib/i18n';
  import { page } from '$app/state';
  import { pushState, replaceState } from '$app/navigation';
  import { api, ApiError, type Tile } from '$lib/api';
  import TileGrid from '$lib/TileGrid.svelte';
  import Viewer from '$lib/Viewer.svelte';
  import { SOURCE_URL } from '$lib/source';
  import { Lock, Link as LinkIcon } from '@lucide/svelte';

  // A visitor's view of a shared album: no account, everything goes through the link token.
  const base = $derived(`/api/s/${page.params.token}`);
  let data = $state<{ name: string; locked: boolean; allow_download?: boolean; assets?: Tile[] } | null>(null);
  let gone = $state(false);
  let error = $state('');
  let password = $state('');
  let width = $state(0);
  let wide = $state(false);

  async function load() {
    try {
      data = await api(base);
    } catch (e) {
      if (e instanceof ApiError && e.status === 410) gone = true;
      else error = (e as Error).message;
    }
  }
  $effect(() => { load(); wide = matchMedia('(min-width: 768px)').matches; });

  async function unlock(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    try {
      await api(`${base}/unlock`, { method: 'POST', body: { password } });
      await load();
    } catch (err) {
      error = (err as Error).message;
    }
  }
  const ids = $derived(data?.assets?.map((t) => t.id) ?? []);
  function open(id: number, e: MouseEvent) {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
    e.preventDefault();
    pushState('', { viewer: id });
  }
</script>

<svelte:head><title>{data?.name ?? t('Shared album')}</title><meta name="robots" content="noindex" /></svelte:head>

<div class="page">
  {#if gone}
    <div class="empty"><LinkIcon aria-hidden="true" /><h1>{t('This link no longer works')}</h1>
      <span class="muted">{t('It may have expired, or the owner turned it off. Ask them for a new one.')}</span></div>
  {:else if data?.locked}
    <form class="gate" onsubmit={unlock}>
      <Lock aria-hidden="true" />
      <h1>{data.name}</h1>
      <div class="field">
        <label for="lp">{t('Password')}</label>
        <input id="lp" type="password" bind:value={password} required aria-invalid={error ? 'true' : undefined} aria-describedby={error ? 'lpe' : undefined} />
      </div>
      {#if error}<p class="error-msg" id="lpe" role="alert">{error}</p>{/if}
      <button class="btn primary">{t('Open')}</button>
    </form>
  {:else if data}
    <header><h1>{data.name}</h1><span class="muted">{t('{n} items', { n: data.assets?.length ?? 0 })} · {t('shared with you')}</span></header>
    <div bind:clientWidth={width}>
      <TileGrid tiles={data.assets ?? []} {width} rowHeight={wide ? 220 : 120} {base} onopen={open} selectable={false} />
    </div>
  {:else if error}
    <p class="error-msg" role="alert">{error}</p>
  {/if}
  <footer class="muted">{t('Shared with Picburrow, free software.')} <a href={SOURCE_URL} rel="noopener">{t('Source code')}</a></footer>
</div>

{#if page.state.viewer && data && !data.locked}
  <Viewer id={page.state.viewer} seq={ids} {base} readonly onclose={() => history.back()}
    onnavigate={(id) => replaceState('', { viewer: id })} />
{/if}

<style>
  .page { min-height: 100dvh; padding: var(--space-5) var(--space-1) var(--space-7); }
  header { padding: 0 var(--space-3) var(--space-5); }
  h1 { overflow-wrap: anywhere; }
  footer { padding: var(--space-6) var(--space-3) 0; font: var(--text-xs) var(--font-sans); text-align: center; }
  footer a { color: inherit; }
  .gate { display: grid; gap: var(--space-5); width: min(100%, 380px); margin: 15dvh auto 0; padding: 0 var(--space-4); }
  .gate > :global(svg) { width: 40px; height: 40px; color: var(--text-muted); }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } header { padding: 0 0 var(--space-5); } }
</style>
