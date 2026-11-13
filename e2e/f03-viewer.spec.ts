// F03 View, zoom, swipe (S04 viewer, S05 info).
import { test, expect, axe, SHOTS, api, photoIn, viewer } from './helpers';

test('F03-H1 arrows step through photos, info shows the details', async ({ page }) => {
  const [p] = await photoIn(page, '2026-09');
  await page.goto(`/photos/${p.id}`);
  await expect(viewer(page).getByRole('img')).toBeVisible();
  await page.screenshot({ path: `${SHOTS}/S04-viewer.png` });
  await axe(page);
  const first = page.url();
  await page.keyboard.press('ArrowRight');
  await expect(page).not.toHaveURL(first);
  await page.keyboard.press('ArrowRight'); // F03-E: a second step after a deep link (focus must survive navigation)
  await page.keyboard.press('ArrowLeft');
  await page.keyboard.press('i');
  const info = page.getByRole('complementary', { name: 'Details' });
  await expect(info).toContainText('Google Pixel 8');
  await expect(info).toContainText('UTC+01:00');
  // capture settings, read from the file on demand
  await expect(info).toContainText('f/1.8 · 1/120 s · ISO 100');
  await expect(info).toContainText('6.9 mm (26 mm equiv.)');
  await expect(info).toContainText('Off, Did not fire');
  await page.screenshot({ path: `${SHOTS}/S05-info.png` });
  await axe(page);
  // a click on the photo closes the details; the viewer stays open on the same photo
  const here = page.url();
  await viewer(page).getByRole('img').click();
  await expect(info).toHaveCount(0);
  await expect(page).toHaveURL(here);
});

test('F03-E1 double-click zooms and swaps in the original', async ({ page }) => {
  const [p] = await photoIn(page, '2026-07');
  await page.goto(`/photos/${p.id}`);
  const img = viewer(page).getByRole('img');
  await expect(img).toHaveAttribute('src', /thumb\/l/);
  await img.dblclick();
  await expect(img).toHaveAttribute('src', /\/original$/);
  await img.dblclick();
  await expect(img).toHaveAttribute('src', /thumb\/l/);
});

test('F03-E2 videos play with seeking (Range)', async ({ page }) => {
  const months = await api<any[]>(page, '/api/timeline/buckets');
  const vid = (await api<any[]>(page, `/api/timeline?month=${months[0].month}`)).find((t) => t.type === 'video');
  await page.goto(`/photos/${vid.id}`);
  const video = viewer(page).locator('video');
  await expect(video).toHaveAttribute('src', /\/play$/);
  const r = await page.request.get(`/api/assets/${vid.id}/play`, { headers: { Range: 'bytes=0-99' } });
  expect(r.status()).toBe(206);
  expect((await r.body()).length).toBe(100);
});

test('F03-E3 the corrupt file opens with a download instead of a picture', async ({ page }) => {
  const months = await api<any[]>(page, '/api/timeline/buckets');
  const broken = (await api<any[]>(page, `/api/timeline?month=${months[0].month}`)).find((t) => t.err);
  await page.goto(`/photos/${broken.id}`);
  await expect(viewer(page).getByText("This file can't be previewed here.")).toBeVisible();
  await expect(viewer(page).getByRole('link', { name: 'Download original' }).first()).toBeVisible();
});

test('F03-E4 the newest photo has no previous button, the oldest no next', async ({ page }) => {
  const months = await api<any[]>(page, '/api/timeline/buckets');
  const newest = (await api<any[]>(page, `/api/timeline?month=${months[0].month}`))[0];
  const oldestMonth = await api<any[]>(page, `/api/timeline?month=${months.at(-1).month}`);
  await page.goto(`/photos/${newest.id}`);
  await expect(viewer(page).getByRole('button', { name: 'Next' })).toBeVisible();
  await expect(viewer(page).getByRole('button', { name: 'Previous' })).toHaveCount(0);
  await page.goto(`/photos/${oldestMonth.at(-1).id}`);
  await expect(viewer(page).getByRole('button', { name: 'Previous' })).toBeVisible();
  await expect(viewer(page).getByRole('button', { name: 'Next' })).toHaveCount(0);
});

test('F03-E5 accents, emoji and HTML-looking file names are shown as text, never run', async ({ page }) => {
  const dec = await api<any[]>(page, '/api/timeline?month=2025-12');
  const names: string[] = [];
  for (const t of dec) names.push((await api(page, `/api/assets/${t.id}`)).name);
  expect(names.sort()).toEqual(['<img src=x onerror=alert(1)>.jpg', 'Plage à Nice 😎.jpg', expect.stringMatching(/^very_long_file_name_.*end\.jpg$/)]);
  for (const t of dec) {
    await page.goto(`/photos/${t.id}`);
    await expect(viewer(page).getByRole('img')).toBeVisible();
    await page.keyboard.press('i');
    await expect(page.getByRole('complementary', { name: 'Details' })).toContainText('Vacances été 🏖️');
  }
  // the guard fixture fails the test if any alert() ran
});

test('F03-E6 swipe left moves to the next photo, swipe down closes @mobile', async ({ page }) => {
  test.skip(!page.viewportSize() || page.viewportSize()!.width >= 768, 'touch only');
  const [p] = await photoIn(page, '2026-09');
  await page.goto(`/photos/${p.id}`);
  await expect(viewer(page).getByRole('img')).toBeVisible();
  await page.screenshot({ path: `${SHOTS}/S04-viewer-mobile.png` });
  // a real finger: CDP touch events make the browser produce genuine pointer events
  const cdp = await page.context().newCDPSession(page);
  const swipe = async (dx: number, dy: number) => {
    const at = (x: number, y: number) => [{ x, y, id: 1 }];
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: at(200, 400) });
    for (let i = 1; i <= 5; i++) await cdp.send('Input.dispatchTouchEvent', { type: 'touchMove', touchPoints: at(200 + (dx * i) / 5, 400 + (dy * i) / 5) });
    await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  };
  const first = page.url();
  await swipe(-150, 0);
  await expect(page).not.toHaveURL(first);
  // Chrome treats a tap within ~0.5 s of a fast swipe as "catching the fling" and sends no click: pause like a person
  await page.waitForTimeout(1500);
  await page.getByRole('button', { name: 'Details' }).tap();
  await expect(page.getByRole('complementary', { name: 'Details' })).toBeVisible();
  await page.screenshot({ path: `${SHOTS}/S05-info-mobile.png` });
  // a tap on the photo above the sheet closes it (no Escape key on a phone)
  const here = page.url();
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: [{ x: 200, y: 150, id: 1 }] });
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  await expect(page.getByRole('complementary', { name: 'Details' })).toHaveCount(0);
  await expect(page).toHaveURL(here);
  await page.waitForTimeout(600); // a second touch right away would count as a double-tap zoom
  await swipe(0, 200);
  await expect(page).toHaveURL(/\/$/);
});
