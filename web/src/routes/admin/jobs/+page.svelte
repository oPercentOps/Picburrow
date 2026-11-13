<script lang="ts">
  import { t, tServer } from '$lib/i18n';
  import { api, type Jobs } from '$lib/api';

  let jobs = $state<Jobs | null>(null);
  let error = $state('');

  $effect(() => {
    let timer: ReturnType<typeof setTimeout>;
    const poll = async () => {
      try {
        jobs = await api<Jobs>('/api/jobs');
        error = '';
      } catch (e) {
        error = (e as Error).message;
      }
      timer = setTimeout(poll, jobs && (jobs.scanning || jobs.thumbs_todo > 0) ? 2000 : 10000);
    };
    poll();
    return () => clearTimeout(timer);
  });
  const total = $derived(jobs ? jobs.thumbs_done + jobs.thumbs_todo : 0);
</script>

<svelte:head><title>{t('Indexing')} · Picburrow</title></svelte:head>

<div class="page">
  <h1>{t('Indexing')}</h1>
  {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
  {#if jobs?.storage_warning}<p class="warn" role="alert">{tServer(jobs.storage_warning)}</p>{/if}
  {#if jobs}
    <section aria-labelledby="scan-h">
      <h2 id="scan-h">{t('Folder scan')}</h2>
      <p role="status">{jobs.scanning ? t('Scanning: {n} new or changed files so far', { n: jobs.scan_found.toLocaleString() }) : t('Idle. Folders are scanned at start-up and every night at 03:00.')}</p>
      {#if jobs.scan_error}<p class="error-msg">{jobs.scan_error}</p>{/if}
    </section>
    <section aria-labelledby="th-h">
      <h2 id="th-h">{t('Previews')}</h2>
      <p>{t('{done} of {total} made', { done: jobs.thumbs_done.toLocaleString(), total: total.toLocaleString() })}{jobs.thumbs_todo ? ', ' + t('{n} to go', { n: jobs.thumbs_todo.toLocaleString() }) : ''}.</p>
      <div class="progress" role="progressbar" aria-label={t('Previews made')} aria-valuenow={jobs.thumbs_done} aria-valuemax={total}>
        <div style:width="{total ? (100 * jobs.thumbs_done) / total : 100}%"></div>
      </div>
    </section>
    <section aria-labelledby="err-h">
      <h2 id="err-h">{t('Files without a preview ({n})', { n: jobs.thumb_errors.toLocaleString() })}</h2>
      {#if jobs.errors?.length}
        <ul class="errs">
          {#each jobs.errors as e}<li><span class="path">{e.path}</span><span class="muted">{e.error}</span></li>{/each}
        </ul>
        {#if jobs.thumb_errors > jobs.errors.length}<p class="muted">{t('Showing the latest {n}.', { n: jobs.errors.length })}</p>{/if}
      {:else}
        <p class="muted">{t('None. Every file could be previewed.')}</p>
      {/if}
    </section>
  {/if}
</div>

<style>
  .page { padding: var(--space-5) var(--space-4) var(--space-7); max-width: 900px; }
  section { margin-top: var(--space-6); }
  h2 { margin-bottom: var(--space-3); }
  p { margin: 0 0 var(--space-3); }
  .progress { height: 6px; background: var(--border); border-radius: var(--radius-pill); overflow: hidden; }
  .progress div { height: 100%; background: var(--accent); transition: width var(--motion-base) var(--motion-ease); }
  .errs { list-style: none; padding: 0; margin: 0; display: grid; gap: var(--space-3); }
  .errs li { display: grid; gap: var(--space-1); padding: var(--space-3) 0; border-bottom: 1px solid var(--border); }
  .path { font: var(--text-sm) var(--font-mono); overflow-wrap: anywhere; }
  .warn { margin-top: var(--space-5); padding: var(--space-4) var(--space-5); border-left: 4px solid var(--warning); background: var(--surface); }
  .errs .muted { overflow-wrap: anywhere; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } }
</style>
