<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, formatWhen } from '$lib/api';
  import { say } from '$lib/state.svelte';
  import { KeyRound, Trash2, Copy } from '@lucide/svelte';

  type Key = { id: number; name: string; created_at: number; last_used_at: number | null; key?: string };
  let keys = $state<Key[] | null>(null);
  let error = $state('');
  let name = $state('');
  let made = $state<Key | null>(null); // the key just made: its secret is shown this once
  let formError = $state('');
  let busy = $state(false);

  const load = () => api<Key[]>('/api/keys').then((k) => (keys = k)).catch((e) => (error = e.message));
  load();

  async function add(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    formError = '';
    try {
      made = await api<Key>('/api/keys', { method: 'POST', body: { name } });
      name = '';
      load();
    } catch (err) {
      formError = (err as Error).message;
    } finally {
      busy = false;
    }
  }
  async function copy() {
    try {
      await navigator.clipboard.writeText(made!.key!);
      say(t('Key copied'));
    } catch {
      say(t('Select the key and copy it'));
    }
  }
  async function remove(k: Key) {
    if (!confirm(t('Delete the key {name}? What uses it stops working.', { name: k.name }))) return;
    try {
      await api(`/api/keys/${k.id}`, { method: 'DELETE' });
      if (made?.id === k.id) made = null;
      say(t('Deleted {name}', { name: k.name }));
      load();
    } catch (err) {
      say((err as Error).message);
    }
  }
</script>

<svelte:head><title>{t('API keys')} · Picburrow</title></svelte:head>

<div class="page">
  <h1>{t('API keys')}</h1>
  <p class="muted intro">{t('A key lets an iPhone Shortcut ask which of your phone photos are not here yet. It can do nothing else: it does not show or change photos.')}</p>
  {#if error}<p class="error-msg" role="alert">{error}</p>{/if}

  {#if made?.key}
    <div class="card new" role="status">
      <strong>{t('Copy this key now. It will not be shown again.')}</strong>
      <code>{made.key}</code>
      <button class="btn secondary" onclick={copy}><Copy aria-hidden="true" />{t('Copy')}</button>
      <span class="hint">{t('In the Shortcut, send it in the header Authorization: Bearer followed by the key.')}</span>
    </div>
  {/if}

  {#if keys?.length}
    <ul class="keys">
      {#each keys as k (k.id)}
        <li>
          <div><strong>{k.name}</strong>
            <div class="muted">{t('Made {date}', { date: formatWhen(k.created_at) })} · {k.last_used_at ? t('last used {date}', { date: formatWhen(k.last_used_at) }) : t('never used')}</div></div>
          <button class="btn ghost" onclick={() => remove(k)} aria-label={t('Delete the key {name}', { name: k.name })}><Trash2 aria-hidden="true" /></button>
        </li>
      {/each}
    </ul>
  {:else if keys}
    <p class="muted empty">{t('No keys yet.')}</p>
  {/if}

  <form class="card" onsubmit={add}>
    <h2>{t('New key')}</h2>
    <div class="field"><label for="kn">{t('Name')}</label><input id="kn" bind:value={name} required maxlength="100" placeholder={t('iPhone Shortcut')} autocomplete="off" /></div>
    {#if formError}<p class="error-msg" role="alert">{formError}</p>{/if}
    <button class="btn primary" disabled={busy}><KeyRound aria-hidden="true" />{t('Make a key')}</button>
  </form>
</div>

<style>
  .page { padding: var(--space-5) var(--space-4) var(--space-7); max-width: 760px; }
  .intro { margin: var(--space-3) 0 var(--space-6); }
  .keys { list-style: none; padding: 0; margin: 0 0 var(--space-7); }
  .keys li { display: flex; gap: var(--space-3); justify-content: space-between; align-items: center; padding: var(--space-4) 0; border-bottom: 1px solid var(--border); }
  .empty { margin: 0 0 var(--space-6); }
  .card { display: grid; gap: var(--space-5); padding: var(--space-5); border: 1px solid var(--border); border-radius: var(--radius-lg); }
  .card.new { gap: var(--space-3); margin-bottom: var(--space-6); border-color: var(--accent); }
  code { font-family: var(--font-mono); overflow-wrap: anywhere; user-select: all; padding: var(--space-3); background: var(--surface); border-radius: var(--radius-sm); }
  .card .btn { justify-self: start; }
  .btn :global(svg) { width: 18px; height: 18px; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } }
</style>
