// F05 Favourite a photo (S04, S09). Albums are not built yet (see test-plan).
import { test, expect, axe, SHOTS, photoIn, viewer, tiles } from './helpers';

test('F05-H1 favourite with the heart and F, see them in Favourites, unfavourite there', async ({ page }) => {
  const [a, b] = await photoIn(page, '2026-09');
  await page.goto(`/photos/${a.id}`);
  const heart = viewer(page).getByRole('button', { name: 'Favourite' });
  await expect(heart).toHaveAttribute('aria-pressed', 'false');
  await page.keyboard.press('f');
  await expect(heart).toHaveAttribute('aria-pressed', 'true');
  await page.goto(`/photos/${b.id}`);
  await heart.click();
  await expect(heart).toHaveAttribute('aria-pressed', 'true');

  await page.goto('/favorites');
  await expect(tiles(page)).toHaveCount(2);
  await expect(tiles(page).first()).toHaveAccessibleName(/, favourite$/);
  await page.screenshot({ path: `${SHOTS}/S09-favourites.png` });
  await axe(page);

  // viewer steps through favourites only
  await tiles(page).first().click();
  await expect(viewer(page).getByRole('button', { name: 'Next' })).toBeVisible();
  await page.keyboard.press('ArrowRight');
  await expect(viewer(page).getByRole('button', { name: 'Next' })).toHaveCount(0);
  await expect(heart).toBeEnabled(); // actions wait for the photo to load (BUG-005)
  // F05-E2 unfavourite in the viewer: it leaves the page behind
  await page.keyboard.press('f');
  await expect(heart).toHaveAttribute('aria-pressed', 'false');
  await page.keyboard.press('Escape');
  await expect(tiles(page)).toHaveCount(1);
  await tiles(page).first().click();
  await expect(heart).toBeEnabled();
  await page.keyboard.press('f');
  await expect(heart).toHaveAttribute('aria-pressed', 'false');
  await page.keyboard.press('Escape');
  await expect(page.getByText('No favourites yet')).toBeVisible();
});

test('F05-E1 a fast double click on the heart leaves the server and the button agreeing', async ({ page }) => {
  const [a] = await photoIn(page, '2026-10');
  await page.goto(`/photos/${a.id}`);
  const heart = viewer(page).getByRole('button', { name: 'Favourite' });
  await expect(heart).toBeVisible();
  await heart.dblclick();
  await page.waitForTimeout(500);
  const shown = (await heart.getAttribute('aria-pressed')) === 'true';
  const saved = (await (await page.request.get(`/api/assets/${a.id}`)).json()).fav;
  expect(shown).toBe(saved);
  if (saved) await heart.click(); // clean up
});

test('F05-E3 BUG-005: F pressed while the next photo is still loading never changes the previous photo', async ({ page }) => {
  const [a, b] = await photoIn(page, '2026-04');
  await page.goto(`/photos/${a.id}`);
  const heart = viewer(page).getByRole('button', { name: 'Favourite' });
  await expect(heart).toHaveAttribute('aria-pressed', 'false');
  // slow down loading the next photo's details, like a busy Pi
  await page.route(`**/api/assets/${b.id}`, async (r) => { await new Promise((ok) => setTimeout(ok, 1500)); await r.continue(); });
  await page.keyboard.press('ArrowRight');
  await page.keyboard.press('f');
  await page.waitForTimeout(2000);
  expect((await (await page.request.get(`/api/assets/${a.id}`)).json()).fav, 'the photo we left').toBe(false);
  // clean up whatever the key did to the new one
  if ((await (await page.request.get(`/api/assets/${b.id}`)).json()).fav) {
    await page.request.patch('/api/assets', { data: { ids: [b.id], favorite: false } });
  }
});
