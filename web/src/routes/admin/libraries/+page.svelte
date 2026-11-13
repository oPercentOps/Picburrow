<script lang="ts">
  import { api, formatWhen, type Library } from '$lib/api';

  let libs = $state<Library[] | null>(null);
  let error = $state('');
  // one editable form per library: folders and exclusion patterns as one-per-line text
  let forms = $state<Record<number, { name: string; paths: string; globs: string; error: string; saved: boolean; busy: boolean }>>({});

  function load() {
    api<Library[]>('/api/libraries').then(fill).catch((e) => (error = e.message));
  }
  function fill(ls: Library[]) {
    libs = ls;
    for (const l of ls) forms[l.id] = { name: l.name, paths: l.import_paths.join('\n'), globs: l.exclusion_globs.join('\n'), error: '', saved: false, busy: false };
  }
  load();

  const lines = (s: string) => s.split('\n').map((x) => x.trim()).filter(Boolean);

  async function save(l: Library, e: SubmitEvent) {
    e.preventDefault();
    const f = forms[l.id];
    const removed = l.import_paths.filter((p) => !lines(f.paths).includes(p));
    if (removed.length && !confirm(`Stop showing photos from:\n${removed.join('\n')}\n\nThe files stay on disk. Their photos move to Trash on the next scan.`)) return;
    f.busy = true;
    f.error = '';
    try {
      fill(await api<Library[]>(`/api/libraries/${l.id}`, { method: 'PATCH', body: { name: f.name, import_paths: lines(f.paths), exclusion_globs: lines(f.globs) } }));
      forms[l.id].saved = true;
    } catch (err) {
      forms[l.id].error = (err as Error).message;
      forms[l.id].busy = false;
    }
  }

  let scanning = $state(false);
  async function rescan(id: number) {
    scanning = true;
    await api(`/api/libraries/${id}/scan`, { method: 'POST', body: {} }).catch((e) => (error = e.message));
    setTimeout(() => (scanning = false), 1500);
  }
</script>

<svelte:head><title>Libraries · Homeroll</title></svelte:head>

<div class="page">
  <h1>Libraries</h1>
  <p class="muted intro">The folders the gallery reads photos from. Files are only read, never moved or changed.</p>
  {#if error}<p class="error-msg" role="alert">{error}</p>{/if}
  {#each libs ?? [] as l (l.id)}
    {@const f = forms[l.id]}
    <form class="card" onsubmit={(e) => save(l, e)}>
      <div class="status">
        <span><strong>{l.assets.toLocaleString()}</strong> photos and videos{#if l.missing}, <a href="/trash">{l.missing.toLocaleString()} missing from disk</a>{/if}</span>
        <span class="muted">Last scan: {formatWhen(l.last_scan_at)}</span>
        {#if l.last_scan_error}<span class="warn" role="status">{l.last_scan_error}</span>{/if}
      </div>
      <div class="field">
        <label for="n{l.id}">Name</label>
        <input id="n{l.id}" bind:value={f.name} required maxlength="100" />
      </div>
      <div class="field">
        <label for="p{l.id}">Folders</label>
        <textarea id="p{l.id}" bind:value={f.paths} rows="3" spellcheck="false" autocapitalize="off" aria-describedby="ph{l.id}"></textarea>
        <span class="hint" id="ph{l.id}">One full path per line, for example /mnt/nas/phone-backup. Subfolders are included.</span>
      </div>
      <div class="field">
        <label for="g{l.id}">Skip files matching</label>
        <textarea id="g{l.id}" class="mono" bind:value={f.globs} rows="3" spellcheck="false" autocapitalize="off" aria-describedby="gh{l.id}"></textarea>
        <span class="hint" id="gh{l.id}">One pattern per line. <code>**/@eaDir/**</code> skips Synology thumbnail folders, <code>**/*.png</code> skips screenshots.</span>
      </div>
      {#if f.error}<p class="error-msg" role="alert">{f.error}</p>{/if}
      {#if f.saved}<p class="ok" role="status">Saved. Scanning for changes now.</p>{/if}
      <div class="actions">
        <button class="btn primary" disabled={f.busy}>{f.busy ? 'Saving…' : 'Save'}</button>
        <button type="button" class="btn secondary" onclick={() => rescan(l.id)} disabled={scanning}>{scanning ? 'Scan started' : 'Scan now'}</button>
        <a class="btn ghost" href="/admin/jobs">See progress</a>
      </div>
    </form>
  {/each}
</div>

<style>
  .page { padding: var(--space-5) var(--space-4) var(--space-7); max-width: 760px; }
  .intro { margin: var(--space-3) 0 var(--space-6); }
  .card { display: grid; gap: var(--space-5); padding: var(--space-5); border: 1px solid var(--border); border-radius: var(--radius-lg); }
  .status { display: grid; gap: var(--space-2); }
  .warn { color: var(--warning); font: var(--text-sm) var(--font-sans); overflow-wrap: anywhere; }
  .ok { color: var(--success); font: var(--text-sm) var(--font-sans); margin: 0; }
  textarea { padding: var(--space-3) var(--space-4); border: 1px solid var(--border-input); border-radius: var(--radius-sm); background: var(--bg);
    color: var(--text); font: var(--text-base) var(--font-sans); resize: vertical; width: 100%; }
  textarea:focus-visible { border-color: var(--accent); }
  .mono, code { font-family: var(--font-mono); }
  .actions { display: flex; flex-wrap: wrap; gap: var(--space-3); }
  @media (min-width: 768px) { .page { padding: var(--space-6) var(--space-6) var(--space-7); } }
</style>
