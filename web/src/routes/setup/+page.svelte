<script lang="ts">
  import { t } from '$lib/i18n';
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

<svelte:head><title>{t('Set up')} · Picburrow</title></svelte:head>

<div class="center">
  <form class="card" onsubmit={submit}>
    <div>
      <h1>{t('Welcome to Picburrow')}</h1>
      <p class="muted">{t('Make your account, then tell Picburrow where your phone photos land. It only reads them: nothing gets moved, renamed or changed.')}</p>
    </div>
    <div class="field">
      <label for="u">{t('Admin username')}</label>
      <input id="u" autocomplete="username" bind:value={username} required />
    </div>
    <div class="field">
      <label for="p">{t('Password')}</label>
      <input id="p" type="password" autocomplete="new-password" minlength="8" bind:value={password} required aria-describedby="ph" />
      <span class="hint" id="ph">{t('At least 8 characters.')}</span>
    </div>
    <div class="field">
      <label for="f">{t('Photo folder on this device')}</label>
      <input id="f" placeholder="/mnt/nas/phone-backup" bind:value={path} required autocapitalize="off" spellcheck="false"
        aria-invalid={error ? 'true' : undefined} aria-describedby="fh" />
      <span class="hint" id="fh">{t('Full path, for example /mnt/nas/phone-backup. You can add more folders later.')}</span>
    </div>
    {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
    <button class="btn primary" disabled={busy} aria-busy={busy}>{busy ? t('Setting up…') : t('Show my photos')}</button>
  </form>
</div>

<style>
  .center { min-height: 100dvh; display: grid; place-items: center; padding: var(--space-5); }
  .card { display: grid; gap: var(--space-5); width: min(100%, 440px); }
  .card p { margin: var(--space-3) 0 0; }
</style>
