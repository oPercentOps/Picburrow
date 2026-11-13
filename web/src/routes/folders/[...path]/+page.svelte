<script lang="ts">
  import { t } from '$lib/i18n';
  import { page } from '$app/state';
  import { api, thumbURL, type Tile } from '$lib/api';
  import TileGrid from '$lib/TileGrid.svelte';
  import { Folder, FolderOpen, ChevronRight } from '@lucide/svelte';

  type Dir = { path: string; name: string; count: number; cover?: Tile };
  type Listing = { path: string; root?: string; dirs: Dir[]; assets: Tile[] };

  // the URL mirrors the disk: /folders/mnt/nas/phone -> /mnt/nas/phone
  const path = $derived(page.params.path ? '/' + page.params.path : '');
  let data = $state<Listing | null>(null);
  let error = $state('');
  let width = $state(0);
  let wide = $state(false);

  $effect(() => {
    const ctl = new AbortController();
    data = null;
    error = '';
    api<Listing>('/api/folders' + (path ? '?path=' + encodeURIComponent(path) : ''), { signal: ctl.signal })
      .then((d) => (data = d))
      .catch((e) => { if (e.name !== 'AbortError') error = e.message; });
    wide = matchMedia('(min-width: 768px)').matches;
    return () => ctl.abort();
  });

  const href = (p: string) => '/folders' + p.split('/').map(encodeURIComponent).join('/');
  // breadcrumb: Folders › <library root> › sub › sub
  const crumbs = $derived.by(() => {
    if (!data?.root) return [];
    const out = [{ name: data.root.split('/').at(-1) || data.root, path: data.root }];
    let acc = data.root;
    for (const seg of data.path.slice(data.root.length).split('/').filter(Boolean)) out.push({ name: seg, path: (acc += '/' + seg) });
    return out;
  });
  const title = $derived(crumbs.at(-1)?.name || t('Folders'));
</script>

<svelte:head><title>{title} · Picburrow</title></svelte:head>

<div class="page">
  <nav aria-label={t('Breadcrumb')} class="crumbs">
    <a href="/folders" aria-current={path ? undefined : 'page'}>{t('Folders')}</a>
    {#each crumbs as c, i}
      <ChevronRight aria-hidden="true" />
      <a href={href(c.path)} title={c.path} aria-current={i === crumbs.length - 1 ? 'page' : undefined}>{c.name}</a>
    {/each}
  </nav>
  <h1>{title}</h1>

  {#if error}
    <p class="error-msg" role="alert">{error}</p>
  {:else if data}
    {#if data.dirs.length}
      <ul class="dirs">
        {#each data.dirs as d (d.path)}
          <li>
            <a href={href(d.path)} title={d.path} aria-label="{d.name}, {t('{n} items', { n: d.count.toLocaleString() })}">
              <span class="cover">
                {#if d.cover}<img src={thumbURL(d.cover, 's')} alt="" loading="lazy" />{:else}<Folder aria-hidden="true" />{/if}
              </span>
              <span class="name">{d.name}</span>
              <span class="muted">{t('{n} items', { n: d.count.toLocaleString() })}</span>
            </a>
          </li>
        {/each}
      </ul>
    {/if}
    {#if data.assets.length}
      <div bind:clientWidth={width} class="files"><TileGrid tiles={data.assets} {width} rowHeight={wide ? 220 : 120} byDay={false} seq /></div>
    {:else if !data.dirs.length}
      <div class="empty"><FolderOpen aria-hidden="true" /><strong>{t('This folder is empty')}</strong><span class="muted">{t('No photos or videos were found here.')}</span></div>
    {/if}
  {/if}
</div>

<style>
  .page { padding: var(--space-5) var(--space-4) var(--space-6); }
  .crumbs { display: flex; flex-wrap: wrap; align-items: center; gap: var(--space-2); margin-bottom: var(--space-3); font: var(--text-sm) var(--font-sans); color: var(--text-muted); }
  .crumbs a { color: inherit; text-decoration: none; overflow-wrap: anywhere; }
  .crumbs a:hover { text-decoration: underline; }
  .crumbs a[aria-current='page'] { color: var(--text); font-weight: 600; }
  .crumbs :global(svg) { width: 16px; height: 16px; flex: none; }
  h1 { overflow-wrap: anywhere; margin-bottom: var(--space-5); }
  .dirs { list-style: none; margin: 0 0 var(--space-6); padding: 0; display: grid; gap: var(--space-5) var(--space-4);
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); }
  .dirs a { display: grid; gap: var(--space-1); text-decoration: none; }
  .cover { aspect-ratio: 1; border-radius: var(--radius-lg); transition: transform var(--motion-base) var(--motion-ease), box-shadow var(--motion-base) var(--motion-ease); overflow: hidden; background: var(--surface); display: grid; place-items: center; color: var(--text-muted); }
  a:hover .cover { transform: translateY(-2px); box-shadow: var(--shadow-pop); }
  .cover img { width: 100%; height: 100%; object-fit: cover; }
  .cover :global(svg) { width: 40px; height: 40px; }
  .name { font: 600 var(--text-sm) var(--font-sans); overflow: hidden; display: -webkit-box; -webkit-line-clamp: 2; line-clamp: 2; -webkit-box-orient: vertical; overflow-wrap: anywhere; }
  @media (min-width: 768px) {
    .page { padding: var(--space-6) var(--space-6) var(--space-7); }
    .dirs { grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); }
  }
</style>
