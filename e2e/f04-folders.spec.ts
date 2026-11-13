// F04 Browse by folder (S06).
import { test, expect, axe, SHOTS, tiles, viewer } from './helpers';

test('F04-H1 library root -> DCIM -> Camera, breadcrumb back, viewer stays in the folder @mobile', async ({ page }) => {
  const mobile = page.viewportSize()!.width < 768;
  await page.goto('/folders');
  const folders = page.getByRole('list').getByRole('link');
  await expect(folders).toHaveCount(1);
  await expect(folders.first()).toHaveAccessibleName(/^\.fixtures, 137 items$/) // 139 files minus the skipped @eaDir one and the hidden live clip;
  await page.screenshot({ path: `${SHOTS}/S06-folders-root${mobile ? '-mobile' : ''}.png` });
  await axe(page);
  await folders.first().click();
  // F04-E3 the Synology @eaDir folder is skipped; the emoji folder is listed
  await expect(page.getByRole('link', { name: /^@eaDir/ })).toHaveCount(0);
  await expect(page.getByRole('link', { name: /^Vacances été 🏖️, 3 items$/ })).toBeVisible();
  await page.getByRole('link', { name: /^DCIM/ }).click();
  await page.getByRole('link', { name: /^Camera/ }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Camera' })).toBeVisible();
  await expect(tiles(page).first()).toBeVisible();
  await expect(page.getByRole('heading', { level: 2 })).toHaveCount(0); // plain grid, no day headers
  expect(await page.evaluate(() => document.documentElement.scrollWidth - innerWidth)).toBeLessThanOrEqual(0);
  if (!mobile) await page.screenshot({ path: `${SHOTS}/S06-folder.png` });
  await axe(page);

  await tiles(page).first().click();
  await expect(viewer(page)).toBeVisible();
  if (!mobile) {
    const u = page.url();
    await page.keyboard.press('ArrowRight');
    await expect(page).not.toHaveURL(u);
    await page.keyboard.press('Escape');
  } else {
    await page.goBack();
  }
  await expect(viewer(page)).toBeHidden();

  // F04-E1 breadcrumb goes back up
  await page.getByRole('navigation', { name: 'Breadcrumb' }).getByRole('link', { name: 'DCIM' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'DCIM' })).toBeVisible();
});

test('F04-E2 folders outside the library are refused', async ({ page }) => {
  for (const p of ['/folders/etc', '/folders/root', '/folders/proc/self']) {
    await page.goto(p);
    await expect(page.getByRole('alert')).toHaveText("That folder isn't in your libraries");
  }
  const r = await page.request.get('/api/folders?path=' + encodeURIComponent(process.cwd() + '/.fixtures/../'));
  expect(r.status()).toBe(404);
});
