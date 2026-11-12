<script lang="ts">
  import ListPage from '$lib/ListPage.svelte';
  import { overrides } from '$lib/state.svelte';
  import { Heart } from '@lucide/svelte';
</script>

<ListPage title="Favourites" query="/api/assets?favorite=1" icon={Heart} empty="No favourites yet"
  hint="Tap the heart on any photo to keep it here." keep={(t) => (overrides[t.id]?.fav ?? true) && !overrides[t.id]?.archived} />
