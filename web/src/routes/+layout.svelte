<script lang="ts">
  import { t } from '$lib/i18n';
  import '../../../replica/design/tokens.css';
  import '$lib/app.css';
  import { page } from '$app/state';
  import { goto, replaceState } from '$app/navigation';
  import Viewer from '$lib/Viewer.svelte';
  import SelectionBar from '$lib/SelectionBar.svelte';
  import AlbumPicker from '$lib/AlbumPicker.svelte';
  import { selection, toast } from '$lib/state.svelte';
  import { api, type User } from '$lib/api';
  import { SOURCE_URL } from '$lib/source';
  import { sleepSeconds } from '$lib/sleep';
  import { Image, Folder, BookImage, Heart, Archive, Search, Trash2, HardDrive, Activity, LogOut, Users, Menu } from '@lucide/svelte';

  let { children } = $props();
  let user = $state<User | null>(null);
  let ready = $state(false);

  const publicRoute = $derived(['/login', '/setup'].includes(page.url.pathname) || page.url.pathname.startsWith('/s/'));

  // picking photos belongs to one page
  let lastPath = '';
  $effect(() => {
    if (page.url.pathname !== lastPath && !page.state.viewer) selection.clear();
    lastPath = page.url.pathname;
  });

  $effect(() => {
    page.url.pathname; // re-check on every navigation
    api<{ needs_setup: boolean; user?: User }>('/api/status').then((s) => {
      user = s.user ?? null;
      ready = true;
      if (s.needs_setup && page.url.pathname !== '/setup') goto('/setup');
      else if (!s.needs_setup && !user && !publicRoute) goto('/login');
    });
  });

  const nav = [
    { href: '/', label: t('Photos'), icon: Image },
    { href: '/folders', label: t('Folders'), icon: Folder },
    { href: '/albums', label: t('Albums'), icon: BookImage },
    { href: '/favorites', label: t('Favourites'), icon: Heart, desktop: true },
    { href: '/archive', label: t('Archive'), icon: Archive, desktop: true },
    { href: '/search', label: t('Search'), icon: Search },
    { href: '/trash', label: t('Trash'), icon: Trash2, desktop: true }
  ];
  const admin = [
    { href: '/admin/libraries', label: t('Libraries'), icon: HardDrive },
    { href: '/admin/jobs', label: t('Indexing'), icon: Activity },
    { href: '/admin/users', label: t('People'), icon: Users }
  ];
  const current = (href: string) => (href === '/' ? page.url.pathname === '/' || page.url.pathname.startsWith('/photos/') : page.url.pathname.startsWith(href));

  // sleep mode: after the chosen time without input, the slideshow starts by itself (like a tablet's photo frame)
  $effect(() => {
    if (!user || publicRoute) return;
    let last = Date.now();
    const active = () => (last = Date.now());
    const events = ['pointerdown', 'pointermove', 'keydown', 'wheel', 'touchstart', 'scroll'] as const;
    for (const e of events) window.addEventListener(e, active, { capture: true, passive: true });
    const tick = setInterval(() => {
      const n = sleepSeconds();
      const watching = [...document.querySelectorAll('video')].some((v) => !v.paused && !v.ended);
      if (page.url.pathname === '/slideshow' || watching || document.visibilityState !== 'visible') return active();
      if (n > 0 && Date.now() - last >= n * 1000) { active(); goto('/slideshow?idle=1'); }
    }, 1000);
    return () => { clearInterval(tick); for (const e of events) window.removeEventListener(e, active, { capture: true }); };
  });

  async function logout() {
    await api('/api/logout', { method: 'POST', body: {} });
    user = null;
    goto('/login');
  }
</script>

<div class="toast" role="status" aria-live="polite">{#if toast.msg}<span>{toast.msg}</span>{/if}</div>

{#if publicRoute}
  {@render children()}
{:else if ready && user}
  <div class="shell">
    <nav class="side" aria-label={t('Main')}>
      <div class="brand"><img src="/icon.svg" alt="" width="28" height="28" />Picburrow</div>
      {#each nav as item}
        <a href={item.href} aria-current={current(item.href) ? 'page' : undefined}><item.icon aria-hidden="true" />{item.label}</a>
      {/each}
      {#if user.is_admin}
        <div class="section">{t('Admin')}</div>
        {#each admin as item}
          <a href={item.href} aria-current={current(item.href) ? 'page' : undefined}><item.icon aria-hidden="true" />{item.label}</a>
        {/each}
      {/if}
      <button class="btn ghost signout" onclick={logout}><LogOut aria-hidden="true" />{t('Sign out {name}', { name: user.username })}</button>
      <a class="source" href={SOURCE_URL} rel="noopener">{t('Source code')} · AGPL-3.0</a>
    </nav>
    <main>{@render children()}</main>
    {#if selection.size > 0}<SelectionBar />{/if}
    <AlbumPicker />
    {#if page.state.viewer}
      <!-- overlay viewer opened from any grid (openViewer); closing goes back to the same scroll position -->
      <Viewer id={page.state.viewer} seq={page.state.seq} onclose={() => history.back()}
        onnavigate={(id) => replaceState(`/photos/${id}`, { viewer: id, seq: page.state.seq })} />
    {/if}
    <nav class="bottom" aria-label={t('Main')}>
      {#each nav.filter((n) => !n.desktop) as item}
        <a href={item.href} aria-current={current(item.href) ? 'page' : undefined}><item.icon aria-hidden="true" />{item.label}</a>
      {/each}
      <!-- phones: everything that doesn't fit the bar (BUG-011: Favourites, Archive, Trash and admin had no way in) -->
      <a href="/more" aria-current={['/more', ...nav.filter((n) => n.desktop).map((n) => n.href), ...admin.map((n) => n.href)].some((h) => page.url.pathname.startsWith(h)) ? 'page' : undefined}><Menu aria-hidden="true" />{t('More')}</a>
    </nav>
  </div>
{/if}

<style>
  .shell { min-height: 100dvh; }
  main { min-width: 0; padding-bottom: calc(var(--bottom-nav) + env(safe-area-inset-bottom) + var(--space-3)); }
  .side { display: none; }
  /* glass: 90% keeps muted text >= 4.5:1 even over a pure black or white photo */
  .bottom, .toast span { background: color-mix(in srgb, var(--bg) 90%, transparent); -webkit-backdrop-filter: blur(20px) saturate(1.6);
    backdrop-filter: blur(20px) saturate(1.6); border: 1px solid color-mix(in srgb, var(--border) 70%, transparent); box-shadow: var(--shadow-pop); }
  .bottom { position: fixed; z-index: 10; left: var(--space-4); right: var(--space-4); bottom: calc(env(safe-area-inset-bottom) + var(--space-3));
    display: flex; justify-content: space-around; gap: var(--space-1); height: calc(var(--bottom-nav) - var(--space-4)); padding: var(--space-2);
    border-radius: var(--radius-pill); max-width: 480px; margin: 0 auto; }
  .bottom a { flex: 1; display: grid; justify-items: center; align-content: center; gap: 2px; min-width: var(--touch-target); border-radius: var(--radius-pill);
    color: var(--text-muted); text-decoration: none; font: var(--weight-xs) var(--text-xs) var(--font-sans);
    transition: background var(--motion-fast) var(--motion-ease), color var(--motion-fast) var(--motion-ease); }
  .bottom a[aria-current='page'] { color: var(--accent); background: var(--accent-tint); font-weight: 700; }
  .bottom a:active { transform: scale(.96); }
  @media (min-width: 768px) {
    .shell { display: grid; grid-template-columns: var(--sidebar) 1fr; }
    main { padding-bottom: 0; }
    .bottom { display: none; }
    .side { display: flex; flex-direction: column; gap: 2px; position: sticky; top: 0; height: 100dvh; padding: var(--space-6) var(--space-4) var(--space-5);
      background: var(--surface); overflow-y: auto; }
    .brand { display: flex; align-items: center; gap: var(--space-3); font: 800 var(--text-lg) var(--font-sans); letter-spacing: -.02em;
      padding: 0 var(--space-3) var(--space-6); }
    .brand img { border-radius: var(--radius-sm); }
    .section { margin-top: var(--space-6); padding: 0 var(--space-4) var(--space-3); color: var(--text-muted); font: var(--weight-xs) var(--text-xs) var(--font-sans); text-transform: uppercase; letter-spacing: .08em; }
    .side a { display: flex; align-items: center; gap: var(--space-4); height: 40px; padding: 0 var(--space-4); border-radius: var(--radius-pill);
      color: var(--text); text-decoration: none; font: 500 var(--text-sm) var(--font-sans); transition: background var(--motion-fast) var(--motion-ease); }
    .side a:hover { background: color-mix(in srgb, var(--surface) 90%, var(--text)); }
    .side a[aria-current='page'] { background: var(--bg); color: var(--accent); font-weight: 700; box-shadow: 0 1px 3px rgba(0,0,0,.08); }
    .signout { margin-top: auto; justify-content: flex-start; font-weight: 500; }
    .side a.source { height: 32px; color: var(--text-muted); font: var(--text-xs) var(--font-sans); }
  }
  nav :global(svg) { width: 20px; height: 20px; flex: none; }
  .toast { position: fixed; z-index: 60; left: 50%; translate: -50% 0; bottom: calc(var(--bottom-nav) + env(safe-area-inset-bottom) + var(--space-4));
    pointer-events: none; width: max-content; max-width: calc(100% - 2 * var(--space-5)); }
  .toast span { display: block; padding: var(--space-4) var(--space-6); border-radius: var(--radius-pill); color: var(--text);
    font: 500 var(--text-sm) var(--font-sans); }
  @media (min-width: 768px) { .toast { bottom: var(--space-6); } }
</style>
