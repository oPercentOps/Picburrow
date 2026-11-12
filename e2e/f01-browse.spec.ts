// F01 Browse recent photos on a phone or laptop (S03 timeline, S04 viewer overlay).
import { test, expect, axe, SHOTS, tiles, viewer } from './helpers';

test('F01-H1 timeline shows days newest first, rows fit, no overflow @mobile', async ({ page }) => {
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  const days = await page.getByRole('heading', { level: 2 }).filter({ hasText: /^\w{3}, \w{3} \d+, \d{4}$/ }).allInnerTexts(); // day headers, not the hidden month headings
  const dates = days.slice(0, 5).map((d) => Date.parse(d));
  expect(dates).toEqual([...dates].sort((a, b) => b - a));
  const overflow = await page.evaluate(() => ({
    page: document.documentElement.scrollWidth - innerWidth,
    rows: Math.max(...[...document.querySelectorAll('section [style*="height"]')].map((r) => r.scrollWidth - r.parentElement!.clientWidth))
  }));
  expect(overflow.page).toBeLessThanOrEqual(0);
  expect(overflow.rows).toBeLessThanOrEqual(1);
  await page.waitForTimeout(500); // thumbnails decode
  const mobile = page.viewportSize()!.width < 768;
  await page.screenshot({ path: `${SHOTS}/S03-timeline${mobile ? '-mobile' : ''}.png` });
  await axe(page);
});

test('F01-E2 the corrupt file shows a no-preview tile', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByText('No preview').first()).toBeVisible();
});

test('F01-E3 keyboard only: Tab to a photo, Enter opens, Escape closes, focus returns', async ({ page }) => {
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  let label: string | null = null;
  for (let i = 0; i < 40 && !label; i++) {
    await page.keyboard.press('Tab');
    label = await page.evaluate(() => (document.activeElement?.matches('a[href^="/photos/"]') ? document.activeElement.getAttribute('aria-label') : null));
  }
  expect(label, 'a photo is reachable with Tab').toBeTruthy();
  await page.keyboard.press('Enter');
  await expect(viewer(page)).toBeVisible();
  await page.keyboard.press('Escape');
  await expect(viewer(page)).toBeHidden();
  await expect(page.locator(':focus')).toHaveAttribute('aria-label', label!);
});

test('F01-E4 closing the viewer keeps the scroll position; Back closes it too', async ({ page }) => {
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  await page.mouse.wheel(0, 700);
  await page.waitForTimeout(300);
  const y = await page.evaluate(() => scrollY);
  const onScreen = await page.evaluate(() =>
    [...document.querySelectorAll('a[href^="/photos/"]')].findIndex((a) => { const r = a.getBoundingClientRect(); return r.top > 0 && r.bottom < innerHeight; }));
  await tiles(page).nth(onScreen).click();
  await expect(viewer(page)).toBeVisible();
  await page.goBack();
  await expect(viewer(page)).toBeHidden();
  expect(Math.abs((await page.evaluate(() => scrollY)) - y)).toBeLessThan(5);
});

test('F01-E5 a month that fails to load can be retried', async ({ page }) => {
  let fail = true;
  await page.route('**/api/timeline?month=*', (r) => (fail ? r.fulfill({ status: 429, body: '{"error":"busy"}' }) /* 4xx: the guard fails on any 5xx */ : r.continue()));
  await page.goto('/');
  await expect(page.getByText("This month didn't load.").first()).toBeVisible();
  fail = false;
  await page.getByRole('button', { name: 'Try again' }).first().click();
  await expect(tiles(page).first()).toBeVisible();
});

test('F01-E6 BUG-004: what a tile shows (video length, "No preview") is part of its name for voice control', async ({ page }) => {
  await page.goto('/');
  await expect(page.getByRole('link', { name: /^Video from .*, 0:04$/ }).first()).toBeVisible();
  await expect(page.getByRole('link', { name: /^Photo from .*, No preview$/ })).toHaveCount(1);
});
