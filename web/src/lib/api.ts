import { goto } from '$app/navigation';
import { tServer } from './i18n';

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
  ph?: string; // tiny WebP placeholder, base64
  live?: number; // id of the live-photo clip
};
export type Asset = Tile & {
  archived: boolean;
  rotation: number; // degrees clockwise it was turned in the app; the file itself is untouched
  keywords: string | null; // from the XMP sidecar next to the file
  description: string | null;
  rating: number | null;
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
  videos_todo: number;
  video_copies: boolean; // admin switch: convert videos for smooth playback
  video_one_thread: boolean; // admin switch: convert on one CPU thread
  errors?: { path: string; error: string }[]; // admins only
  storage_warning?: string; // admins only: data on the SD card
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
  // an expired session sends you to sign in; a wrong password on a share link must not (BUG-009)
  if (res.status === 401 && !path.startsWith('/api/login') && !path.startsWith('/api/s/')) {
    goto('/login');
  }
  if (!res.ok) {
    let msg = `Request failed (${res.status})`;
    try {
      msg = (await res.json()).error ?? msg;
    } catch {}
    throw new ApiError(res.status, tServer(msg));
  }
  return res.status === 204 || res.status === 202 ? (undefined as T) : res.json();
}

// base: '/api' for your own photos, '/api/s/<token>' for a shared album
export const thumbURL = (t: { id: number; v: number | null }, size: 's' | 'l', base = '/api') =>
  `${base}/assets/${t.id}/thumb/${size}` + (t.v ? `?v=${t.v}` : '');

// Browsers can show these originals directly; anything else (HEIC, RAW, TIFF) uses the large preview.
export const browserCanShow = (mime: string) =>
  /^image\/(jpeg|png|gif|webp|avif)$/.test(mime) || /^video\/(mp4|webm|quicktime)$/.test(mime);

export function formatDay(taken: string) {
  const d = new Date(taken.slice(0, 10) + 'T12:00:00');
  return d.toLocaleDateString(undefined, { weekday: 'short', day: 'numeric', month: 'short', year: 'numeric' });
}

// today's date where the viewer is, like 2024-05-17
export function today() {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
}

export function formatMonth(month: string) {
  return new Date(month + '-15T12:00:00').toLocaleDateString(undefined, { month: 'long', year: 'numeric' });
}

export function formatDuration(ms?: number) {
  if (!ms) return '';
  const s = Math.round(ms / 1000);
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`;
}

export type Library = {
  id: number;
  name: string;
  import_paths: string[];
  exclusion_globs: string[];
  last_scan_at: number | null;
  last_scan_error: string | null;
  assets: number;
  missing: number;
};

export const formatWhen = (unix: number | null) =>
  unix ? new Date(unix * 1000).toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' }) : 'never';

export type Link = { id: number; token: string; has_password: boolean; allow_download: boolean; expires_at: number | null; created_at: number };
export type Album = { id: number; name: string; count: number; cover?: Tile; shared: boolean; updated_at: number; assets?: Tile[]; links?: Link[] };
