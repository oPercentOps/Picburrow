// Live photos, star ratings, Trash (S15), People (admin).
import { test, expect, axe, SHOTS, LIB, tiles, viewer, api, answerPrompt, waitIndexed, USER } from './helpers';
import { renameSync } from 'node:fs';

test('L-1 a live photo shows as one item and plays its clip', async ({ page }) => {
  const aug = await api<any[]>(page, '/api/timeline?month=2026-08');
  expect(aug.length, 'the clip is not a separate item').toBe(1);
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  const live = page.getByRole('link', { name: /^Photo from .*, Live$/ });
  // August is further down: months load near the viewport
  await page.evaluate(() => document.getElementById('m-2026-08')!.scrollIntoView());
  await expect(live).toHaveCount(1);
  await page.goto(`/photos/${aug[0].id}`);
  const play = viewer(page).getByRole('button', { name: 'Play live photo' });
  await play.click();
  await expect(play).toHaveAttribute('aria-pressed', 'true');
  await expect(viewer(page).locator('video')).toHaveAttribute('src', new RegExp(`/assets/${aug[0].live}/original$`));
  await page.screenshot({ path: `${SHOTS}/S04-live.png` });
  await expect(play).toHaveAttribute('aria-pressed', 'false', { timeout: 6_000 }); // back to the still when the clip ends
  await expect(viewer(page).locator('video')).toHaveCount(0);
});

test('R-1 rate with stars or keys, search by stars, clear', async ({ page }) => {
  const [p] = (await api<any[]>(page, '/api/timeline?month=2026-07')).filter((t) => t.type === 'image');
  await page.goto(`/photos/${p.id}`);
  await expect(viewer(page).getByRole('img')).toBeVisible();
  await page.keyboard.press('i');
  const stars = page.getByRole('radiogroup', { name: 'Rating' });
  await stars.getByRole('radio', { name: '4 stars' }).click();
  await expect(stars.getByRole('radio', { name: '4 stars' })).toHaveAttribute('aria-checked', 'true');
  await page.screenshot({ path: `${SHOTS}/S05-rating.png` });
  await axe(page);
  await page.keyboard.press('2'); // keys 1-5 set, 0 clears
  await expect(stars.getByRole('radio', { name: '2 stars' })).toHaveAttribute('aria-checked', 'true');
  expect((await api(page, `/api/assets/${p.id}`)).rating).toBe(2);
  const found = await api<any[]>(page, '/api/search?rating=2');
  expect(found.map((t) => t.id).sort()).toEqual([p.id, (await api<any[]>(page, '/api/search?rating=5'))[0].id].sort()); // + the fixture rated 5 in its file
  await page.goto('/search?rating=5');
  await expect(page.getByRole('status').filter({ hasText: /found$/ })).toHaveText('1 found');
  await page.goto(`/photos/${p.id}`);
  await expect(viewer(page).getByRole('img')).toBeVisible();
  await page.keyboard.press('0');
  await expect.poll(async () => (await api(page, `/api/assets/${p.id}`)).rating).toBe(null);
});

test('T-1 a file that leaves the disk goes to Trash and comes back with its favourite; Forget empties it @mobile', async ({ page }) => {
  const mobile = page.viewportSize()!.width < 768;
  const before = (await api<any[]>(page, '/api/trash')).length; // F07-H2 already left one there
  await page.goto('/trash');
  await expect(page.getByRole('heading', { level: 1, name: 'Trash' })).toBeVisible();
  await page.screenshot({ path: `${SHOTS}/S15-trash${mobile ? '-mobile' : ''}.png` });
  await axe(page);
  if (mobile) return;
  const file = `${LIB}/DCIM/Camera/IMG_20260403_1.jpg`;
  const [p] = await api<any[]>(page, '/api/search?q=IMG_20260403_1');
  await page.request.patch('/api/assets', { data: { ids: [p.id], favorite: true } });
  const lib = (await api<any[]>(page, '/api/libraries'))[0];
  renameSync(file, file + '.away');
  try {
    await page.request.post(`/api/libraries/${lib.id}/scan`, { data: {} });
    await waitIndexed(page);
    await page.reload();
    await expect(tiles(page)).toHaveCount(before + 1);
  } finally {
    renameSync(file + '.away', file);
  }
  await page.request.post(`/api/libraries/${lib.id}/scan`, { data: {} });
  await waitIndexed(page);
  await page.reload();
  await expect(tiles(page)).toHaveCount(before);
  expect((await api(page, `/api/assets/${p.id}`)).fav, 'favourite kept').toBe(true);
  await page.request.patch('/api/assets', { data: { ids: [p.id], favorite: false } });
  if (before > 0) {
    await page.getByRole('button', { name: 'Forget them now' }).click(); // confirm accepted by the guard
    await expect(page.getByText('Trash is empty')).toBeVisible();
  }
});

test('P-1 an admin adds a family member with their own folder; they see only their photos', async ({ page, browser }) => {
  test.setTimeout(90_000);
  await page.goto('/admin/users');
  await expect(page.getByRole('listitem').filter({ hasText: USER.name })).toBeVisible();
  await axe(page);
  await page.getByLabel('Username').fill('lea');
  await page.getByLabel('Password').fill('lea password');
  await page.getByLabel('Their photo folder').fill('/does/not/exist');
  await page.getByRole('button', { name: 'Add' }).click();
  await expect(page.getByRole('alert')).toContainText("doesn't exist");
  await page.getByLabel('Their photo folder').fill(`${LIB}/Vacances été 🏖️`);
  await page.getByRole('button', { name: 'Add' }).click();
  await expect(page.getByRole('listitem').filter({ hasText: 'lea' })).toBeVisible();
  await waitIndexed(page);
  await page.reload();
  await expect(page.getByRole('listitem').filter({ hasText: 'lea' })).toContainText('3 photos');
  await page.screenshot({ path: `${SHOTS}/S16-people.png` });

  const lea = await (await browser.newContext({ viewport: { width: 1440, height: 900 } })).newPage();
  await lea.goto('/login');
  await lea.getByLabel('Username').fill('lea');
  await lea.getByLabel('Password').fill('lea password');
  await lea.getByRole('button', { name: 'Sign in' }).click();
  await expect(tiles(lea)).toHaveCount(3);
  await expect(lea.getByRole('link', { name: 'People' })).toHaveCount(0); // not an admin
  expect((await lea.request.get('/api/users')).status()).toBe(403);

  answerPrompt(page, 'another password');
  await page.getByRole('listitem').filter({ hasText: 'lea' }).getByRole('button', { name: 'New password' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'New password set for lea' })).toBeVisible();
  await lea.reload();
  await expect(lea).toHaveURL(/\/login$/); // signed out by the reset
  await lea.context().close();
  await page.getByRole('button', { name: 'Delete lea' }).click();
  await expect(page.getByRole('listitem').filter({ hasText: 'lea' })).toHaveCount(0);
});

test('N-1 BUG-011: on a phone, More reaches Favourites, Archive, Trash and the admin pages @mobile', async ({ page }) => {
  test.skip(page.viewportSize()!.width >= 768, 'phone layout');
  await page.goto('/');
  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'More' }).tap();
  await expect(page.getByRole('heading', { level: 1, name: 'More' })).toBeVisible();
  await page.screenshot({ path: `${SHOTS}/S17-more-mobile.png` });
  await axe(page);
  await expect(page.getByRole('link', { name: 'Source code' })).toHaveAttribute('href', 'https://github.com/oPercentOps/Picburrow'); // AGPL section 13
  const more = page.getByRole('navigation', { name: 'More' });
  for (const [name, heading] of [['Favourites', 'Favourites'], ['Archive', 'Archive'], ['Trash', 'Trash'], ['People', 'People']]) {
    await more.getByRole('link', { name }).tap();
    await expect(page.getByRole('heading', { level: 1, name: heading })).toBeVisible();
    await expect(page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'More' })).toHaveAttribute('aria-current', 'page');
    await page.goBack();
  }
});

test('R-2 rate several photos at once from the selection bar', async ({ page }) => {
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  const circles = page.getByRole('button', { name: /^Select / });
  await circles.nth(1).click();
  await circles.nth(2).click();
  const bar = page.getByRole('toolbar', { name: 'Selected photos' });
  await bar.getByRole('button', { name: 'Rate' }).click();
  await expect(bar.getByRole('button', { name: 'Rate' })).toHaveAttribute('aria-expanded', 'true');
  await axe(page);
  await page.getByRole('group', { name: 'Rate the selected photos' }).getByRole('button', { name: /3 stars/ }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Rated 2 with 3 stars' })).toBeVisible();
  const rated = await api<any[]>(page, '/api/search?rating=3');
  expect(rated.length).toBeGreaterThanOrEqual(2);
  // clear them again
  for (const t of rated) await page.request.patch('/api/assets', { data: { ids: [t.id], rating: 0 } });
});

test('L-1 AGPL: the sidebar offers the source code on desktop', async ({ page }) => {
  test.skip(page.viewportSize()!.width < 768, 'desktop layout');
  await page.goto('/');
  await expect(page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Source code · AGPL-3.0' })).toHaveAttribute('href', 'https://github.com/oPercentOps/Picburrow');
});
