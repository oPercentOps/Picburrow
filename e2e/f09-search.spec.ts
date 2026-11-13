// S11 search, S03 placeholders, installable app.
import { test, expect, axe, SHOTS, tiles, api } from './helpers';

test('S11-H1 search by name, date range and kind; the URL keeps the search @mobile', async ({ page }) => {
  const mobile = page.viewportSize()!.width < 768;
  await page.goto('/search');
  await page.getByLabel('Words in the file name, folder or camera').fill('plage');
  await page.getByRole('button', { name: 'Search' }).click();
  await expect(page).toHaveURL(/\?q=plage$/);
  await expect(page.getByRole('status').filter({ hasText: '1 found' })).toBeVisible();
  await expect(tiles(page)).toHaveCount(1);
  await page.screenshot({ path: `${SHOTS}/S11-search${mobile ? '-mobile' : ''}.png` });
  await axe(page);
  if (mobile) return;
  await page.reload(); // the query survives a reload
  await expect(tiles(page)).toHaveCount(1);
  await expect(page.getByLabel('Words in the file name, folder or camera')).toHaveValue('plage');

  await page.goto('/search?from=2026-09-01&to=2026-09-30');
  await expect(page.getByRole('status').filter({ hasText: /found$/ })).toHaveText('16 found');
  await page.goto('/search?type=video');
  await expect(page.getByRole('status').filter({ hasText: /found$/ })).toHaveText('3 found');
  await page.goto('/search?q=no-such-thing-xyz');
  await expect(page.getByText('Nothing matches')).toBeVisible();
  await page.goto('/search?q=%25'); // a literal %, not "everything"
  await expect(page.getByText('Nothing matches')).toBeVisible();
});

test('S03-P1 every indexed photo carries a blur placeholder', async ({ page }) => {
  const sept = await api<any[]>(page, '/api/timeline?month=2026-09');
  expect(sept.every((t) => typeof t.ph === 'string' && t.ph.length > 20)).toBe(true);
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
});

test('X-4 the app can be installed (manifest, icons)', async ({ page }) => {
  const r = await page.request.get('/manifest.json');
  expect(r.headers()['content-type']).toContain('json');
  const m = await r.json();
  expect(m.display).toBe('standalone');
  for (const i of m.icons) expect((await page.request.get(i.src)).status(), i.src).toBe(200);
  await page.goto('/');
  await expect(page.locator('link[rel="manifest"]')).toHaveAttribute('href', '/manifest.json');
});
