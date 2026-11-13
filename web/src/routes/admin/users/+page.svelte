<script lang="ts">
  import { api, formatWhen, type User } from '$lib/api';
  import { say } from '$lib/state.svelte';
  import { UserPlus, KeyRound, Trash2 } from '@lucide/svelte';

  type Row = User & { created_at: number; assets: number };
  let users = $state<Row[] | null>(null);
  let me = $state<User | null>(null);
  let error = $state('');
  let form = $state({ username: '', password: '', path: '', is_admin: false });
  let formError = $state('');
  let busy = $state(false);

  const load = () => api<Row[]>('/api/users').then((u) => (users = u)).catch((e) => (error = e.message));
  load();
  api<User>('/api/me').then((u) => (me = u));

  async function add(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    formError = '';
    try {
      users = await api<Row[]>('/api/users', { method: 'POST', body: form });
      say(`Added ${form.username}. Their photos are being indexed.`);
      form = { username: '', password: '', path: '', is_admin: false };
    } catch (err) {
      formError = (err as Error).message;
    } finally {
      busy = false;
    }
  }
  async function reset(u: Row) {
    const pw = prompt(`New password for ${u.username} (at least 8 characters). They'll be signed out everywhere.`);
    if (!pw) return;
    try {
      await api(`/api/users/${u.id}`, { method: 'PATCH', body: { password: pw } });
      say(`New password set for ${u.username}`);
    } catch (err) {
      say((err as Error).message);
    }
  }
  async function remove(u: Row) {
    if (!confirm(`Delete ${u.username}? Their albums and share links go too. Their photo files stay on disk.`)) return;
    await api(`/api/users/${u.id}`, { method: 'DELETE' });
    say(`Deleted ${u.username}`);
    load();
  }
</script>

<svelte:head><title>People · Homeroll</title></svelte:head>

<div class="page">
  <h1>People</h1>
  <p class="muted intro">Everyone gets their own sign-in and their own photo folder. Nobody sees another person's photos unless they share an album.</p>
  {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
  {#if users}
    <ul class="people">
      {#each users as u (u.id)}
        <li>
          <div><strong>{u.username}</strong>{#if u.is_admin}{' '}<span class="tag">admin</span>{/if}
            <div class="muted">{u.assets.toLocaleString()} photos · since {formatWhen(u.created_at)}</div></div>
          <div class="actions">
            <button class="btn ghost" onclick={() => reset(u)}><KeyRound aria-hidden="true" />New password</button>
            {#if u.id !== me?.id}<button class="btn ghost" onclick={() => remove(u)} aria-label="Delete {u.username}"><Trash2 aria-hidden="true" /></button>{/if}
          </div>
        </li>
      {/each}
    </ul>
  {/if}

  <form class="card" onsubmit={add}>
    <h2>Add someone</h2>
    <div class="field"><label for="nu">Username</label><input id="nu" bind:value={form.username} required autocomplete="off" /></div>
    <div class="field"><label for="np">Password</label><input id="np" type="password" bind:value={form.password} required minlength="8" autocomplete="new-password" aria-describedby="nph" />
      <span class="hint" id="nph">At least 8 characters. Tell them in person.</span></div>
    <div class="field"><label for="nf">Their photo folder</label><input id="nf" bind:value={form.path} required placeholder="/mnt/nas/backup-ann" autocapitalize="off" spellcheck="false" /></div>
    <label class="check"><input type="checkbox" bind:checked={form.is_admin} /> Can manage libraries and people</label>
    {#if formError}<p class="error-msg" role="alert">{formError}</p>{/if}
    <button class="btn primary" disabled={busy}><UserPlus aria-hidden="true" />Add</button>
  </form>
</div>

<style>
  .page { padding: var(--space-5) var(--space-4) var(--space-7); max-width: 760px; }
  .intro { margin: var(--space-3) 0 var(--space-6); }
  .people { list-style: none; padding: 0; margin: 0 0 var(--space-7); }
  .people li { display: flex; flex-wrap: wrap; gap: var(--space-3); justify-content: space-between; align-items: center; padding: var(--space-4) 0; border-bottom: 1px solid var(--border); }
  .tag { font: 600 var(--text-xs) var(--font-sans); color: var(--accent); }
  .actions { display: flex; gap: var(--space-1); }
  .actions :global(svg), form :global(svg) { width: 18px; height: 18px; }
  .card { display: grid; gap: var(--space-5); padding: var(--space-5); border: 1px solid var(--border); border-radius: var(--radius-lg); }
  .check { display: flex; gap: var(--space-3); align-items: center; font: var(--text-sm) var(--font-sans); min-height: var(--touch-target); }
  .check input { width: 20px; height: 20px; accent-color: var(--accent); }
  .card .btn { justify-self: start; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } }
</style>
