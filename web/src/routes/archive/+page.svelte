<script lang="ts">
  import { t } from '$lib/i18n';
  import ListPage from '$lib/ListPage.svelte';
  import { overrides } from '$lib/state.svelte';
  import { Archive } from '@lucide/svelte';
</script>

<ListPage title={t('Archive')} query="/api/assets?archived=1" icon={Archive} empty={t('Nothing archived')}
  hint={t('Archived photos stay on disk but are hidden from the timeline. Archive one from the photo viewer.')}
  keep={(t) => overrides[t.id]?.archived ?? true} />
