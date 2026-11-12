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
