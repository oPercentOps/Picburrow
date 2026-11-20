// F08 Hide a photo from the timeline (S04, S10).
import { test, expect, axe, SHOTS, api, photoIn, viewer, tiles } from './helpers';

test('F08-H1 archive moves on to the next photo, leaves the timeline, unarchive brings it back', async ({ page }) => {
  const count = async () => (await api<any[]>(page, '/api/timeline/buckets')).find((b) => b.month === '2026-06').count;
  const before = await count();
  const [p] = await photoIn(page, '2026-06');
  await page.goto(`/photos/${p.id}`);
  await viewer(page).getByRole('button', { name: 'Archive' }).click();
  await expect(page).not.toHaveURL(new RegExp(`/photos/${p.id}$`));
  expect(await count()).toBe(before - 1);

  await page.goto('/archive');
  await expect(tiles(page)).toHaveCount(1);
  await page.screenshot({ path: `${SHOTS}/S10-archive.png` });
  await axe(page);
  await tiles(page).first().click();
  await viewer(page).getByRole('button', { name: 'Unarchive' }).click();
  await expect(viewer(page)).toBeHidden();
  await expect(page.getByText('Nothing archived')).toBeVisible();
  expect(await count()).toBe(before);
});

test('F08-D1 delete moves the selected photo to Trash; Restore there brings it back', async ({ page }) => {
  const count = async () => (await api<any[]>(page, '/api/timeline/buckets')).reduce((n, b) => n + b.count, 0);
  const before = await count();
  const inTrash = (await api<any[]>(page, '/api/trash')).length;
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  const shown = await tiles(page).count();
  await page.getByRole('button', { name: /^Select (Photo|Video) from/ }).first().click();
  const bar = page.getByRole('toolbar', { name: 'Selected photos' });
  await bar.getByRole('button', { name: 'Delete' }).click();
  await expect(page.getByRole('status').filter({ hasText: /^Moved \d+ to Trash/ })).toBeVisible();
  await expect(tiles(page)).toHaveCount(shown - 1);
  expect(await count()).toBeLessThan(before);

  await page.goto('/trash');
  await expect(tiles(page)).not.toHaveCount(inTrash);
  await page.getByRole('button', { name: /^Select (Photo|Video) from/ }).first().click();
  await bar.getByRole('button', { name: 'Restore' }).click();
  await expect(page.getByRole('status').filter({ hasText: /^Restored \d+/ })).toBeVisible();
  await expect(tiles(page)).toHaveCount(inTrash);
  expect(await count()).toBe(before);
});
