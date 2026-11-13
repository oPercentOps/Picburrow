import { pushState } from '$app/navigation';
import { SvelteSet } from 'svelte/reactivity';

// Changes made in this session (viewer favourite/archive), applied over tiles loaded before the change.
export const overrides = $state<Record<number, { fav?: boolean; archived?: boolean }>>({});

// Multi-select: ids picked in any grid. While it is not empty, clicking a photo picks it instead of opening it.
export const selection = new SvelteSet<number>();

// One-line message at the bottom of the screen ("Added 3 photos to Summer").
export const toast = $state({ msg: '' });
let toastTimer: ReturnType<typeof setTimeout>;
export function say(msg: string) {
  toast.msg = msg;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (toast.msg = ''), 4000);
}

// Bumped after changes that a page's own list doesn't know about (photos removed from an album).
export const refresh = $state({ n: 0 });

// Album picker: open it with the ids to add.
export const picker = $state<{ ids: number[] | null }>({ ids: null });

// Opens the viewer as an overlay (shallow route), so the page behind keeps its scroll position.
// seq = the ids to step through (folder, favourites); without it the viewer follows the timeline.
export function openViewer(id: number, e: MouseEvent, seq?: number[]) {
  if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return; // let the browser open a new tab
  e.preventDefault();
  pushState(`/photos/${id}`, { viewer: id, seq });
}
