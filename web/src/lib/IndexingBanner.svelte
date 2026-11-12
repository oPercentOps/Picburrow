<script lang="ts">
  import { api, type Jobs } from '$lib/api';

  let { onscandone }: { onscandone: () => void } = $props();
  let jobs = $state<Jobs | null>(null);

  $effect(() => {
    let wasScanning = false;
    let timer: ReturnType<typeof setTimeout>;
    let stopped = false;
    const poll = async () => {
      try {
        jobs = await api<Jobs>('/api/jobs');
        if (wasScanning && !jobs.scanning) onscandone();
        wasScanning = jobs.scanning;
      } catch {}
      // poll fast while busy, rarely when idle
      if (!stopped) timer = setTimeout(poll, jobs && (jobs.scanning || jobs.thumbs_todo > 0) ? 3000 : 60000);
    };
    poll();
    return () => { stopped = true; clearTimeout(timer); };
  });

  const total = $derived(jobs ? jobs.thumbs_done + jobs.thumbs_todo : 0);
</script>

{#if jobs && (jobs.scanning || jobs.thumbs_todo > 0)}
  <div class="banner" role="status">
    <div>
      {#if jobs.scanning}
        <div>Looking for photos: {jobs.scan_found.toLocaleString()} new so far</div>
      {:else}
        <div>Still making previews. Photos appear as they're ready.</div>
      {/if}
      <div class="muted">Previews {jobs.thumbs_done.toLocaleString()} of {total.toLocaleString()}</div>
    </div>
    <div class="progress" role="progressbar" aria-label="Previews made" aria-valuenow={jobs.thumbs_done} aria-valuemax={total}>
      <div style:width="{total ? (100 * jobs.thumbs_done) / total : 0}%"></div>
    </div>
  </div>
{/if}

<style>
  .banner { display: flex; flex-wrap: wrap; gap: var(--space-4); align-items: center; justify-content: space-between;
    padding: var(--space-4) var(--space-5); background: var(--surface); border-radius: var(--radius-md); margin-top: var(--space-4); }
  .progress { height: 4px; background: var(--border); border-radius: var(--radius-pill); overflow: hidden; flex: 1 1 200px; max-width: 320px; }
  .progress div { height: 100%; background: var(--accent); transition: width var(--motion-base) var(--motion-ease); }
</style>
