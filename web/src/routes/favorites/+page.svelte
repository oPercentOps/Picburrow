<script lang="ts">
  import { t } from '$lib/i18n';
  import ListPage from '$lib/ListPage.svelte';
  import { overrides } from '$lib/state.svelte';
  import { Heart } from '@lucide/svelte';
</script>

<ListPage title={t('Favourites')} query="/api/assets?favorite=1" icon={Heart} empty={t('No favourites yet')}
  hint={t('Tap the heart on any photo to keep it here.')} keep={(t) => (overrides[t.id]?.fav ?? true) && !overrides[t.id]?.archived} />
