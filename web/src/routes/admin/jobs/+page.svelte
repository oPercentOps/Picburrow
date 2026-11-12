<script lang="ts">
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

<svelte:head><title>Indexing · Photos</title></svelte:head>

<div class="page">
  <h1>Indexing</h1>
  {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
  {#if jobs}
    <section aria-labelledby="scan-h">
      <h2 id="scan-h">Folder scan</h2>
      <p role="status">{jobs.scanning ? `Scanning: ${jobs.scan_found.toLocaleString()} new or changed files so far` : 'Idle. Folders are scanned at start-up and every night at 03:00.'}</p>
      {#if jobs.scan_error}<p class="error-msg">{jobs.scan_error}</p>{/if}
    </section>
    <section aria-labelledby="th-h">
      <h2 id="th-h">Previews</h2>
      <p>{jobs.thumbs_done.toLocaleString()} of {total.toLocaleString()} made{jobs.thumbs_todo ? `, ${jobs.thumbs_todo.toLocaleString()} to go` : ''}.</p>
      <div class="progress" role="progressbar" aria-label="Previews made" aria-valuenow={jobs.thumbs_done} aria-valuemax={total}>
        <div style:width="{total ? (100 * jobs.thumbs_done) / total : 100}%"></div>
      </div>
    </section>
    <section aria-labelledby="err-h">
      <h2 id="err-h">Files without a preview ({jobs.thumb_errors.toLocaleString()})</h2>
      {#if jobs.errors?.length}
        <ul class="errs">
          {#each jobs.errors as e}<li><span class="path">{e.path}</span><span class="muted">{e.error}</span></li>{/each}
        </ul>
        {#if jobs.thumb_errors > jobs.errors.length}<p class="muted">Showing the latest {jobs.errors.length}.</p>{/if}
      {:else}
        <p class="muted">None. Every file could be previewed.</p>
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
  .errs .muted { overflow-wrap: anywhere; }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } }
</style>
