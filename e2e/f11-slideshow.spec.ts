// Issue #20: a random, self-advancing slideshow with smooth transitions.
import { test, expect, axe, SHOTS } from './helpers';

const current = (page: import('@playwright/test').Page) =>
  page.getByRole('region', { name: 'Slideshow' }).getByRole('img').last();

test('SS-1 start from Photos, photos advance on their own, pause, stop', async ({ page }) => {
  await page.goto('/');
  await page.getByRole('link', { name: 'Slideshow' }).click();
  const show = page.getByRole('region', { name: 'Slideshow' });
  await expect(current(page)).toBeVisible();
  await expect(current(page)).toHaveAttribute('src', /\/thumb\/l/); // the large preview, never the original
  await page.getByLabel('Change photo every').selectOption('5');
  const first = await current(page).getAttribute('src');
  await expect.poll(() => current(page).getAttribute('src'), { timeout: 12_000 }).not.toBe(first);
  await page.mouse.move(400, 300); // controls come back with the pointer
  await page.screenshot({ path: `${SHOTS}/S18-slideshow.png` });
  await axe(page);
  // pause holds the photo
  await page.getByRole('button', { name: 'Pause' }).click();
  const held = await current(page).getAttribute('src');
  await page.waitForTimeout(6500);
  expect(await current(page).getAttribute('src')).toBe(held);
  await page.mouse.move(420, 320); // the controls faded out while it sat still; moving the mouse brings them back
  await page.getByRole('button', { name: 'Play' }).click();
  // the interval is remembered
  await page.keyboard.press('Escape');
  await expect(page).toHaveURL(/\/$/);
  await expect(show).toHaveCount(0);
  await page.getByRole('link', { name: 'Slideshow' }).click();
  await expect(page.getByLabel('Change photo every')).toHaveValue('5');
  await page.getByRole('button', { name: 'Stop slideshow' }).click();
  await expect(page).toHaveURL(/\/$/);
});

test('SS-2 on a phone: tap shows the controls, the X stops it @mobile', async ({ page }) => {
  test.skip(page.viewportSize()!.width >= 768, 'phone layout');
  await page.goto('/');
  await page.getByRole('link', { name: 'Slideshow' }).tap();
  await expect(current(page)).toBeVisible();
  const stop = page.getByRole('button', { name: 'Stop slideshow' });
  await expect(stop).toBeVisible();
  await page.waitForTimeout(3500); // controls fade away for passive viewing
  await expect(page.locator('.bar')).toHaveClass(/hidden/);
  await page.touchscreen.tap(200, 400);
  await expect(page.locator('.bar')).not.toHaveClass(/hidden/);
  await page.screenshot({ path: `${SHOTS}/S18-slideshow-mobile.png` });
  await axe(page);
  await stop.tap();
  await expect(page).toHaveURL(/\/$/);
});

test('SL-1 sleep mode: idle for the chosen time starts the slideshow; moving the mouse returns', async ({ page }) => {
  test.skip(page.viewportSize()!.width < 768, 'desktop');
  await page.goto('/slideshow');
  await expect(current(page)).toBeVisible();
  await page.getByLabel('Start the slideshow by itself when the screen is left alone').selectOption({ label: 'Sleep: 30 s' });
  expect(await page.evaluate(() => localStorage.getItem('picburrow.sleep.seconds'))).toBe('30');
  await page.getByRole('button', { name: 'Stop slideshow' }).click();
  await page.evaluate(() => localStorage.setItem('picburrow.sleep.seconds', '2')); // shorter than any choice, to keep the test fast
  await page.goto('/folders');
  await expect(page.getByRole('heading', { level: 1, name: 'Folders' })).toBeVisible();
  await expect(page).toHaveURL(/\/slideshow\?idle=1$/, { timeout: 8000 });
  await expect(current(page)).toBeVisible();
  await expect(page.locator('.bar')).toHaveClass(/hidden/); // a photo frame: no controls
  await page.mouse.move(300, 300);
  await page.mouse.move(360, 340); // a real movement, not a twitch
  await expect(page).toHaveURL(/\/folders$/);
  await expect(page.getByRole('heading', { level: 1, name: 'Folders' })).toBeVisible();
});

test('SL-2 sleep mode on a phone: a tap wakes it back to the page @mobile', async ({ page }) => {
  test.skip(page.viewportSize()!.width >= 768, 'phone layout');
  await page.goto('/albums');
  await page.evaluate(() => localStorage.setItem('picburrow.sleep.seconds', '2'));
  await expect(page).toHaveURL(/\/slideshow\?idle=1$/, { timeout: 8000 });
  await expect(current(page)).toBeVisible();
  await page.touchscreen.tap(200, 400);
  await expect(page).toHaveURL(/\/albums$/);
});
