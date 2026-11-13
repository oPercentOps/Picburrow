// Sleep mode: start the slideshow by itself after this many seconds without a touch, click or key (0 = off).
// Kept per device (a tablet used as a photo frame wants it, a phone usually doesn't).
const KEY = 'picburrow.sleep.seconds';
export const SLEEP_CHOICES = [0, 30, 60, 120, 300, 900];

export function sleepSeconds(): number {
  try { return Math.max(0, Number(localStorage.getItem(KEY)) || 0); } catch { return 0; }
}

export function setSleepSeconds(s: number) {
  try { localStorage.setItem(KEY, String(s)); } catch {}
}
