<script lang="ts">
  import { t } from '$lib/i18n';
  import { today } from '$lib/api';
  import ListPage from '$lib/ListPage.svelte';
  import { overrides } from '$lib/state.svelte';
  import { History } from '@lucide/svelte';
</script>

<ListPage title={t('On this day')} query="/api/assets?onthisday={today()}" icon={History} empty={t('Nothing from this day in earlier years')}
  hint={t("Photos taken on today's date in past years show up here.")} keep={(t) => !overrides[t.id]?.archived} />
