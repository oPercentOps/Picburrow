<script lang="ts">
  import ListPage from '$lib/ListPage.svelte';
  import { overrides } from '$lib/state.svelte';
  import { Archive } from '@lucide/svelte';
</script>

<ListPage title="Archive" query="/api/assets?archived=1" icon={Archive} empty="Nothing archived"
  hint="Archived photos stay on disk but are hidden from the timeline. Archive one from the photo viewer."
  keep={(t) => overrides[t.id]?.archived ?? true} />
