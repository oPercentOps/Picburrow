<script lang="ts">
  import { goto } from '$app/navigation';
  import { api } from '$lib/api';

  let username = $state('');
  let password = $state('');
  let error = $state('');
  let busy = $state(false);

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = '';
    try {
      await api('/api/login', { method: 'POST', body: { username, password } });
      goto('/');
    } catch (err) {
      error = (err as Error).message;
    } finally {
      busy = false;
    }
  }
</script>

<svelte:head><title>Sign in · Homeroll</title></svelte:head>

<div class="center">
  <form class="card" onsubmit={submit}>
    <h1>Sign in</h1>
    <div class="field">
      <label for="u">Username</label>
      <input id="u" autocomplete="username" bind:value={username} required />
    </div>
    <div class="field">
      <label for="p">Password</label>
      <input id="p" type="password" autocomplete="current-password" bind:value={password} required
        aria-invalid={error ? 'true' : undefined} aria-describedby={error ? 'err' : undefined} />
    </div>
    {#if error}<p class="error-msg" id="err" role="alert">{error}</p>{/if}
    <button class="btn primary" disabled={busy} aria-busy={busy}>{busy ? 'Signing in…' : 'Sign in'}</button>
  </form>
</div>

<style>
  .center { min-height: 100dvh; display: grid; place-items: center; padding: var(--space-5); }
  .card { display: grid; gap: var(--space-5); width: min(100%, 380px); }
</style>
