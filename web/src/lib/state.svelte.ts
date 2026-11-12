import { pushState } from '$app/navigation';

// Changes made in this session (viewer favourite/archive), applied over tiles loaded before the change.
export const overrides = $state<Record<number, { fav?: boolean; archived?: boolean }>>({});

// Opens the viewer as an overlay (shallow route), so the page behind keeps its scroll position.
// seq = the ids to step through (folder, favourites); without it the viewer follows the timeline.
export function openViewer(id: number, e: MouseEvent, seq?: number[]) {
  if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return; // let the browser open a new tab
  e.preventDefault();
  pushState(`/photos/${id}`, { viewer: id, seq });
}
