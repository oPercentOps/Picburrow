<script lang="ts">
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';

  let username = $state('');
  let password = $state('');
  let path = $state('');
  let error = $state('');
  let busy = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = '';
    try {
      await api('/api/setup', { method: 'POST', body: { username, password, path } });
      goto('/');
    } catch (err) {
      error = (err as Error).message;
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>Set up · Photos</title></svelte:head>

<div class="center">
  <form class="card" onsubmit={submit}>
    <div>
      <h1>Welcome</h1>
      <p class="muted">Create the admin account and point the gallery at the folder where your phone photos land. Files are only read, never moved or changed.</p>
    </div>
    <div class="field">
      <label for="u">Admin username</label>
      <input id="u" autocomplete="username" bind:value={username} required />
    </div>
    <div class="field">
      <label for="p">Password</label>
      <input id="p" type="password" autocomplete="new-password" minlength="8" bind:value={password} required aria-describedby="ph" />
      <span class="hint" id="ph">At least 8 characters.</span>
    </div>
    <div class="field">
      <label for="f">Photo folder on this device</label>
      <input id="f" placeholder="/mnt/nas/phone-backup" bind:value={path} required autocapitalize="off" spellcheck="false"
        aria-invalid={error ? 'true' : undefined} aria-describedby="fh" />
      <span class="hint" id="fh">Full path, for example /mnt/nas/phone-backup. You can add more folders later.</span>
    </div>
    {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
    <button class="btn primary" disabled={busy} aria-busy={busy}>{busy ? 'Setting up…' : 'Start indexing'}</button>
  </form>
</div>

<style>
  .center { min-height: 100dvh; display: grid; place-items: center; padding: var(--space-5); }
  .card { display: grid; gap: var(--space-5); width: min(100%, 440px); }
  .card p { margin: var(--space-3) 0 0; }
</style>
