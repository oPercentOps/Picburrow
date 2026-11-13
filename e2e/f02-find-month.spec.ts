// F02 Find a photo from a past month (S03 scrubber).
import { test, expect, axe, SHOTS, tiles } from './helpers';

test('F02-H1 scrubber: keyboard End jumps to the oldest month, Home back to the newest', async ({ page }) => {
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  const slider = page.getByRole('slider', { name: 'Jump to month' });
  await slider.focus();
  await page.keyboard.press('End');
  await expect(slider).toHaveAttribute('aria-valuetext', 'December 2025');
  await expect(page.getByRole('heading', { level: 2, name: /Dec 31, 2025/ })).toBeInViewport(); // oldest month: the page can't scroll it to the very top
  await page.screenshot({ path: `${SHOTS}/S03-scrubbed.png` });
  await page.keyboard.press('Home');
  await expect(slider).toHaveAttribute('aria-valuetext', 'November 2026');
  await axe(page);
});

test('F02-E1 dragging the scrubber moves through the months', async ({ page }) => {
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  const slider = page.getByRole('slider', { name: 'Jump to month' });
  const box = (await slider.boundingBox())!;
  await page.mouse.move(box.x + box.width / 2, box.y + 4);
  await page.mouse.down();
  await page.mouse.move(box.x + box.width / 2, box.y + box.height * 0.6, { steps: 6 });
  await page.mouse.up();
  await expect(slider).not.toHaveAttribute('aria-valuetext', 'November 2026');
});

test.describe('F02-E2 time zones', () => {
  test.use({ timezoneId: 'Pacific/Auckland' });
  test('a photo taken 23:30 on 31 Dec (+01:00) stays on 31 Dec for a viewer in New Zealand', async ({ page }) => {
    await page.goto('/');
    const slider = page.getByRole('slider', { name: 'Jump to month' });
    await expect(tiles(page).first()).toBeVisible();
    await slider.focus();
    await page.keyboard.press('End');
    await expect(page.getByRole('heading', { level: 2, name: /Dec 31, 2025/ })).toBeVisible();
    await expect(page.getByRole('heading', { level: 2, name: /Jan 1, 2026/ })).toHaveCount(0);
  });
});

test('F02-E3 dateless videos land on their file date (today)', async ({ page }) => {
  const today = new Date().toLocaleDateString('en-US', { weekday: 'short', month: 'short', day: 'numeric', year: 'numeric' });
  await page.goto('/');
  await expect(page.getByRole('heading', { level: 2, name: today })).toBeVisible();
  await expect(page.getByRole('link', { name: /^Video from/ }).first()).toBeVisible();
});
