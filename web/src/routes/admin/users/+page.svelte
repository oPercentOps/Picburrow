<script lang="ts">
  import { t } from '$lib/i18n';
  import { api, formatWhen, type User } from '$lib/api';
  import { say } from '$lib/state.svelte';
  import { UserPlus, KeyRound, Trash2, Users } from '@lucide/svelte';

  type Row = User & { created_at: number; assets: number; libraries: number[] };
  let users = $state<Row[] | null>(null);
  let me = $state<User | null>(null);
  let error = $state('');
  const empty = () => ({ username: '', password: '', path: '', is_admin: false, library_id: 0 });
  let form = $state(empty());
  let libs = $state<{ id: number; name: string }[]>([]); // the admin's own libraries, which a new person can share
  let formError = $state('');
  let busy = $state(false);

  const load = () => api<Row[]>('/api/users').then((u) => (users = u)).catch((e) => (error = e.message));
  load();
  api<User>('/api/me').then((u) => (me = u));
  api<{ id: number; name: string }[]>('/api/libraries').then((l) => (libs = l)).catch(() => {});

  async function add(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    formError = '';
    try {
      users = await api<Row[]>('/api/users', { method: 'POST', body: form });
      say(form.library_id ? t('Added {name}. They see the same photos as you.', { name: form.username })
        : t('Added {name}. Their photos are being indexed.', { name: form.username }));
      form = empty();
    } catch (err) {
      formError = (err as Error).message;
    } finally {
      busy = false;
    }
  }
  async function reset(u: Row) {
    const pw = prompt(t("New password for {name} (at least 8 characters). They'll be signed out everywhere.", { name: u.username }));
    if (!pw) return;
    try {
      await api(`/api/users/${u.id}`, { method: 'PATCH', body: { password: pw } });
      say(t('New password set for {name}', { name: u.username }));
    } catch (err) {
      say((err as Error).message);
    }
  }
  // let someone who already has an account see one of my libraries too, or stop
  async function share(u: Row, lib: { id: number; name: string }) {
    const shared = !u.libraries.includes(lib.id);
    if (!shared && !confirm(t('Stop sharing {lib} with {name}? Its photos also leave their albums.', { lib: lib.name, name: u.username }))) return;
    try {
      await api(`/api/users/${u.id}`, { method: 'PATCH', body: { library_id: lib.id, shared } });
      say(shared ? t('{name} now sees {lib} too', { name: u.username, lib: lib.name }) : t('{name} no longer sees {lib}', { name: u.username, lib: lib.name }));
      load();
    } catch (err) {
      say((err as Error).message);
    }
  }
  async function remove(u: Row) {
    if (!confirm(t('Delete {name}? Their albums and share links go too. Their photo files stay on disk.', { name: u.username }))) return;
    await api(`/api/users/${u.id}`, { method: 'DELETE' });
    say(t('Deleted {name}', { name: u.username }));
    load();
  }
</script>

<svelte:head><title>{t('People')} · Picburrow</title></svelte:head>

<div class="page">
  <h1>{t('People')}</h1>
  <p class="muted intro">{t("Everyone gets their own sign-in. People see their own photo folder, plus any library you share with them here.")}</p>
  {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
  {#if users}
    <ul class="people">
      {#each users as u (u.id)}
        <li>
          <div><strong>{u.username}</strong>{#if u.is_admin}{' '}<span class="tag">{t('admin')}</span>{/if}
            <div class="muted">{u.assets.toLocaleString()} photos · since {formatWhen(u.created_at)}</div></div>
          <div class="actions">
            {#if u.id !== me?.id}
              {#each libs as l (l.id)}
                <button class="btn ghost" onclick={() => share(u, l)} aria-pressed={u.libraries.includes(l.id)}><Users aria-hidden="true" />{u.libraries.includes(l.id) ? t('Stop sharing ({name})', { name: l.name }) : t('Share my photos ({name})', { name: l.name })}</button>
              {/each}
            {/if}
            <button class="btn ghost" onclick={() => reset(u)}><KeyRound aria-hidden="true" />{t('New password')}</button>
            {#if u.id !== me?.id}<button class="btn ghost" onclick={() => remove(u)} aria-label={t('Delete {name}', { name: u.username })}><Trash2 aria-hidden="true" /></button>{/if}
          </div>
        </li>
      {/each}
    </ul>
  {/if}

  <form class="card" onsubmit={add}>
    <h2>{t('Add someone')}</h2>
    <div class="field"><label for="nu">{t('Username')}</label><input id="nu" bind:value={form.username} required autocomplete="off" /></div>
    <div class="field"><label for="np">{t('Password')}</label><input id="np" type="password" bind:value={form.password} required minlength="8" autocomplete="new-password" aria-describedby="nph" />
      <span class="hint" id="nph">{t('At least 8 characters. Tell them in person.')}</span></div>
    {#if libs.length}
      <div class="field"><label for="nl">{t('Their photos')}</label>
        <select id="nl" bind:value={form.library_id} aria-describedby="nlh">
          <option value={0}>{t('A folder of their own')}</option>
          {#each libs as l (l.id)}<option value={l.id}>{t('Share my photos ({name})', { name: l.name })}</option>{/each}
        </select>
        <span class="hint" id="nlh">{t('Sharing: the same photos, favourites, ratings and archive, with nothing indexed twice. Albums stay personal.')}</span></div>
    {/if}
    {#if !form.library_id}
      <div class="field"><label for="nf">{t('Their photo folder')}</label><input id="nf" bind:value={form.path} required placeholder="/mnt/nas/backup-ann" autocapitalize="off" spellcheck="false" /></div>
    {/if}
    <label class="check"><input type="checkbox" bind:checked={form.is_admin} /> {t('Can manage libraries and people')}</label>
    {#if formError}<p class="error-msg" role="alert">{formError}</p>{/if}
    <button class="btn primary" disabled={busy}><UserPlus aria-hidden="true" />{t('Add')}</button>
  </form>
</div>

<style>
  .page { padding: var(--space-5) var(--space-4) var(--space-7); max-width: 760px; }
  .intro { margin: var(--space-3) 0 var(--space-6); }
  .people { list-style: none; padding: 0; margin: 0 0 var(--space-7); }
  .people li { display: flex; flex-wrap: wrap; gap: var(--space-3); justify-content: space-between; align-items: center; padding: var(--space-4) 0; border-bottom: 1px solid var(--border); }
  .tag { font: 600 var(--text-xs) var(--font-sans); color: var(--accent); }
  .actions { display: flex; flex-wrap: wrap; gap: var(--space-1); }
  .actions :global(svg), form :global(svg) { width: 18px; height: 18px; }
  .card { display: grid; gap: var(--space-5); padding: var(--space-5); border: 1px solid var(--border); border-radius: var(--radius-lg); }
  .check { display: flex; gap: var(--space-3); align-items: center; font: var(--text-sm) var(--font-sans); min-height: var(--touch-target); }
  .check input { width: 20px; height: 20px; accent-color: var(--accent); }
  .card .btn { justify-self: start; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } }
</style>
