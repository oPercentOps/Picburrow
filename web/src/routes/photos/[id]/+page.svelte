<script lang="ts">
  import { page } from '$app/state';
  import { goto } from '$app/navigation';
  import Viewer from '$lib/Viewer.svelte';

  // deep link or reload of /photos/:id: the viewer on its own, closing returns to the timeline.
  // When opened as an overlay (page.state.viewer), the layout renders the viewer instead.
  const id = $derived(Number(page.params.id));
</script>

{#if !page.state.viewer}
  <Viewer {id} onnavigate={(to) => goto(`/photos/${to}`, { replaceState: true, keepFocus: true, noScroll: true })} onclose={() => goto('/')} />
{/if}
