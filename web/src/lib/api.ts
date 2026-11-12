import { goto } from '$app/navigation';

export type User = { id: number; username: string; is_admin: boolean };
export type Tile = {
  id: number;
  w: number | null;
  h: number | null;
  type: 'image' | 'video';
  taken: string;
  fav: boolean;
  duration_ms?: number;
  v: number | null;
  err?: boolean;
};
export type Asset = Tile & {
  path: string;
  dir: string;
  name: string;
  size: number;
  mime: string;
  tz_offset_min: number | null;
  camera_make: string | null;
  camera_model: string | null;
  lens: string | null;
  lat: number | null;
  lon: number | null;
  prev_id: number | null;
  next_id: number | null;
};
export type Bucket = { month: string; count: number };
export type Jobs = {
  scanning: boolean;
  scan_found: number;
  scan_error: string;
  thumbs_done: number;
  thumbs_todo: number;
  thumb_errors: number;
};

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

export async function api<T>(path: string, init?: { method?: string; body?: unknown; signal?: AbortSignal }): Promise<T> {
  const res = await fetch(path, {
    method: init?.method ?? 'GET',
    signal: init?.signal,
    headers: init?.body !== undefined ? { 'Content-Type': 'application/json' } : undefined,
    body: init?.body !== undefined ? JSON.stringify(init.body) : undefined
  });
  if (res.status === 401 && !path.startsWith('/api/login')) {
    goto('/login');
  }
  if (!res.ok) {
    let msg = `Request failed (${res.status})`;
    try {
      msg = (await res.json()).error ?? msg;
    } catch {}
    throw new ApiError(res.status, msg);
  }
  return res.status === 204 || res.status === 202 ? (undefined as T) : res.json();
}

export const thumbURL = (t: { id: number; v: number | null }, size: 's' | 'l') =>
  `/api/assets/${t.id}/thumb/${size}` + (t.v ? `?v=${t.v}` : '');

// Browsers can show these originals directly; anything else (HEIC, RAW, TIFF) uses the large preview.
export const browserCanShow = (mime: string) =>
  /^image\/(jpeg|png|gif|webp|avif)$/.test(mime) || /^video\/(mp4|webm|quicktime)$/.test(mime);

export function formatDay(taken: string) {
  const d = new Date(taken.slice(0, 10) + 'T12:00:00');
  return d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' });
}

export function formatMonth(month: string) {
  return new Date(month + '-15T12:00:00').toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
}

export function formatDuration(ms?: number) {
  if (!ms) return '';
  const s = Math.round(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}
