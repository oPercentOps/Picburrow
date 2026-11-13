<script lang="ts">
  import { t } from '$lib/i18n';
  import { goto } from '$app/navigation';
  import { api, type User } from '$lib/api';
  import { SOURCE_URL } from '$lib/source';
  import { Heart, Archive, Trash2, HardDrive, Activity, Users, LogOut, ChevronRight } from '@lucide/svelte';

  // Phones: the bottom bar holds four places; the rest lives here.
  let me = $state<User | null>(null);
  api<User>('/api/me').then((u) => (me = u));
  const links = [
    { href: '/favorites', label: t('Favourites'), icon: Heart },
    { href: '/archive', label: t('Archive'), icon: Archive },
    { href: '/trash', label: t('Trash'), icon: Trash2 }
  ];
  const admin = [
    { href: '/admin/libraries', label: t('Libraries'), icon: HardDrive },
    { href: '/admin/jobs', label: t('Indexing'), icon: Activity },
    { href: '/admin/users', label: t('People'), icon: Users }
  ];
  async function logout() {
    await api('/api/logout', { method: 'POST', body: {} });
    goto('/login');
  }
</script>

<svelte:head><title>{t('More')} · Picburrow</title></svelte:head>

<div class="page">
  <h1>{t('More')}</h1>
  <nav aria-label={t('More')}>
    <ul>
      {#each links as l}<li><a href={l.href}><l.icon aria-hidden="true" /><span>{l.label}</span><ChevronRight aria-hidden="true" /></a></li>{/each}
    </ul>
    {#if me?.is_admin}
      <h2>{t('Admin')}</h2>
      <ul>
        {#each admin as l}<li><a href={l.href}><l.icon aria-hidden="true" /><span>{l.label}</span><ChevronRight aria-hidden="true" /></a></li>{/each}
      </ul>
    {/if}
  </nav>
  {#if me}<button class="btn secondary out" onclick={logout}><LogOut aria-hidden="true" />{t('Sign out {name}', { name: me.username })}</button>{/if}
  <p class="muted about">{t('Picburrow is free software under the AGPL-3.0.')} <a class="inline" href={SOURCE_URL} rel="noopener">{t('Source code')}</a></p>
</div>

<style>
  .page { padding: var(--space-5) var(--space-4) var(--space-7); max-width: 560px; }
  h1 { margin-bottom: var(--space-5); }
  h2 { font: 600 var(--text-sm) var(--font-sans); color: var(--text-muted); margin: var(--space-6) 0 var(--space-2); }
  ul { list-style: none; margin: 0; padding: 0; }
  a { display: flex; align-items: center; gap: var(--space-4); min-height: 52px; border-bottom: 1px solid var(--border); text-decoration: none; }
  a span { flex: 1; }
  a :global(svg), .out :global(svg) { width: 20px; height: 20px; }
  a :global(svg:last-child) { color: var(--text-muted); }
  .out { margin-top: var(--space-6); }
  .about { margin-top: var(--space-6); }
  a.inline { display: inline; min-height: 0; border: 0; text-decoration: underline; }
</style>
