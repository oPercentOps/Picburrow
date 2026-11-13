<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, formatWhen, type Album } from '$lib/api';
  import { say } from '$lib/state.svelte';
  import { Copy, Trash2 } from '@lucide/svelte';

  let { album, onchange, onclose }: { album: Album; onchange: (a: Album) => void; onclose: () => void } = $props();

  let dialog: HTMLDialogElement;
  let password = $state('');
  let expires = $state('never');
  let allowDownload = $state(true);
  let error = $state('');
  let busy = $state(false);
  $effect(() => dialog.showModal());

  const url = (token: string) => `${location.origin}/s/${token}`;
  const days: Record<string, number> = { day: 1, week: 7, month: 30 };

  async function create(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = '';
    try {
      const expires_at = expires === 'never' ? null : Math.floor(Date.now() / 1000) + days[expires] * 86400;
      onchange(await api<Album>(`/api/albums/${album.id}/links`, { method: 'POST', body: { password, allow_download: allowDownload, expires_at } }));
      password = '';
    } catch (err) {
      error = (err as Error).message;
    } finally {
      busy = false;
    }
  }
  async function copy(token: string) {
    try {
      await navigator.clipboard.writeText(url(token));
      say(t('Link copied'));
    } catch {
      say(t('Copy the link from the text box')); // clipboard needs https or localhost
    }
  }
  async function revoke(id: number) {
    if (!confirm(t('Turn this link off? Anyone using it loses access.'))) return;
    await api(`/api/links/${id}`, { method: 'DELETE' });
    onchange(await api<Album>(`/api/albums/${album.id}`));
    say(t('Link turned off'));
  }
</script>

<dialog bind:this={dialog} aria-labelledby="share-title" onclose={onclose}>
  <h2 id="share-title">{t('Share "{name}"', { name: album.name })}</h2>
  <p class="muted">{t('Anyone with a link can see the photos in this album, without an account. Nothing else in your library.')}</p>

  {#if album.links?.length}
    <ul class="links">
      {#each album.links as l (l.id)}
        <li>
          <input readonly value={url(l.token)} aria-label={t('Share link')} onfocus={(e) => e.currentTarget.select()} />
          <span class="muted">{t(l.has_password ? 'Password' : 'No password')} · {t(l.allow_download ? 'downloads on' : 'view only')} · {l.expires_at ? t('ends {date}', { date: formatWhen(l.expires_at) }) : t('no end date')}</span>
          <div class="row">
            <button class="btn secondary" onclick={() => copy(l.token)}><Copy aria-hidden="true" />{t('Copy')}</button>
            <button class="btn ghost" onclick={() => revoke(l.id)}><Trash2 aria-hidden="true" />{t('Turn off')}</button>
          </div>
        </li>
      {/each}
    </ul>
  {/if}

  <form onsubmit={create}>
    <h3>{t('New link')}</h3>
    <div class="field">
      <label for="share-pw">{t('Password (optional)')}</label>
      <input id="share-pw" type="text" bind:value={password} autocomplete="off" aria-describedby="share-pw-h" />
      <span class="hint" id="share-pw-h">{t('Visitors type it once per browser. Leave empty for no password.')}</span>
    </div>
    <div class="field">
      <label for="share-exp">{t('Stops working')}</label>
      <select id="share-exp" bind:value={expires}>
        <option value="never">{t('Never')}</option><option value="day">{t('After a day')}</option><option value="week">{t('After a week')}</option><option value="month">{t('After a month')}</option>
      </select>
    </div>
    <label class="check"><input type="checkbox" bind:checked={allowDownload} /> {t('Visitors can download originals and play videos')}</label>
    {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
    <div class="row end">
      <button type="button" class="btn ghost" onclick={() => dialog.close()}>{t('Done')}</button>
      <button class="btn primary" disabled={busy}>{t('Create link')}</button>
    </div>
  </form>
</dialog>

<style>
  dialog { width: min(100% - 2 * var(--space-5), 520px); max-height: 85dvh; padding: var(--space-6); border: 0; border-radius: var(--radius-lg);
    background: var(--bg); color: var(--text); box-shadow: var(--shadow-pop); }
  dialog::backdrop { background: rgba(0, 0, 0, .5); }
  h2 { margin: 0 0 var(--space-3); overflow-wrap: anywhere; }
  h3 { font: 600 var(--text-sm) var(--font-sans); margin: var(--space-6) 0 var(--space-4); }
  form { display: grid; gap: var(--space-4); }
  .links { list-style: none; padding: 0; margin: var(--space-5) 0 0; display: grid; gap: var(--space-5); }
  .links li { display: grid; gap: var(--space-2); }
  .links input, select { height: 40px; padding: 0 var(--space-4); border: 1px solid var(--border-input); border-radius: var(--radius-sm);
    background: var(--bg); color: var(--text); font: var(--text-sm) var(--font-mono); width: 100%; }
  select { font-family: var(--font-sans); }
  .row { display: flex; gap: var(--space-2); flex-wrap: wrap; }
  .row.end { justify-content: flex-end; }
  .row :global(svg) { width: 16px; height: 16px; }
  .check { display: flex; gap: var(--space-3); align-items: center; font: var(--text-sm) var(--font-sans); min-height: var(--touch-target); }
  .check input { width: 20px; height: 20px; accent-color: var(--accent); }
</style>
