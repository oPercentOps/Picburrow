// F05 (albums) and F06 (share an album with family): S07, S08, S12, multi-select.
import { test, expect, axe, SHOTS, tiles, viewer, api, answerPrompt } from './helpers';

test('F05-A1 pick photos, make an album, remove one, rename; F06-H1 share it with a password, view only', async ({ page, browser }) => {
  test.setTimeout(60_000);
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  // pick 3 photos with their select circles (keyboard-reachable buttons)
  for (const i of [0, 1, 2]) await page.getByRole('button', { name: /^Select (Photo|Video) from/ }).nth(i).click();
  const bar = page.getByRole('toolbar', { name: 'Selected photos' });
  await expect(bar).toContainText('3 selected');
  await page.screenshot({ path: `${SHOTS}/S03-selecting.png` });
  await axe(page);
  await bar.getByRole('button', { name: 'Add to album' }).click();
  const picker = page.getByRole('dialog', { name: /Add 3 to an album/ });
  await picker.getByLabel('New album').fill('Été 2026 🌞');
  await picker.getByRole('button', { name: 'Create' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Made the album Été 2026 🌞 with 3 photos' })).toBeVisible();
  await expect(bar).toBeHidden();

  await page.goto('/albums');
  const card = page.getByRole('link', { name: /^Été 2026 🌞, 3 items$/ });
  await expect(card).toBeVisible();
  await page.screenshot({ path: `${SHOTS}/S07-albums.png` });
  await axe(page);
  await card.click();
  await expect(page.getByRole('heading', { level: 1, name: 'Été 2026 🌞' })).toBeVisible();
  await expect(tiles(page)).toHaveCount(3);
  // remove one from the album (it stays in the library)
  await page.getByRole('button', { name: /^Select / }).first().click();
  await page.getByRole('toolbar', { name: 'Selected photos' }).getByRole('button', { name: 'Remove from album' }).click();
  await expect(tiles(page)).toHaveCount(2);
  answerPrompt(page, 'Summer');
  await page.getByRole('button', { name: 'Rename album' }).click();
  await expect(page.getByRole('heading', { level: 1, name: 'Summer' })).toBeVisible();
  await page.screenshot({ path: `${SHOTS}/S08-album.png` });
  await axe(page);

  // share: password, view only
  await page.getByRole('button', { name: /^Share/ }).click();
  const share = page.getByRole('dialog', { name: 'Share "Summer"' });
  await share.getByLabel('Password (optional)').fill('sunny');
  await share.getByLabel('Visitors can download originals and play videos').uncheck();
  await share.getByRole('button', { name: 'Create link' }).click();
  const linkBox = share.getByRole('textbox', { name: 'Share link' });
  await expect(linkBox).toHaveValue(/\/s\/[\w-]{40,}$/);
  await expect(share).toContainText('Password · view only · no end date');
  await page.screenshot({ path: `${SHOTS}/S08-share.png` });
  await axe(page);
  const url = await linkBox.inputValue();

  // a visitor: no account
  const guestCtx = await browser.newContext({ viewport: { width: 1440, height: 900 } });
  const guest = await guestCtx.newPage();
  const guestErrors: string[] = [];
  guest.on('pageerror', (e) => guestErrors.push(e.message));
  await guest.goto(url);
  await expect(guest.getByRole('heading', { level: 1, name: 'Summer' })).toBeVisible();
  await guest.getByLabel('Password').fill('rainy');
  await guest.getByRole('button', { name: 'Open' }).click();
  await expect(guest.getByRole('alert')).toHaveText('Wrong password');
  await guest.screenshot({ path: `${SHOTS}/S12-password.png` });
  await axe(guest);
  await guest.getByLabel('Password').fill('sunny');
  await guest.getByRole('button', { name: 'Open' }).click();
  await expect(tiles(guest)).toHaveCount(2);
  await expect(guest.getByRole('button', { name: /^Select / })).toHaveCount(0); // visitors can't pick
  await expect(guest.getByRole('link', { name: 'Photos' })).toHaveCount(0); // no app navigation for visitors
  await expect(guest.getByRole('link', { name: 'Source code' })).toHaveAttribute('href', 'https://github.com/oPercentOps/Picburrow'); // AGPL section 13
  await guest.screenshot({ path: `${SHOTS}/S12-shared.png` });
  await axe(guest);
  await tiles(guest).first().click();
  await expect(viewer(guest)).toBeVisible();
  await expect(viewer(guest).getByRole('link', { name: 'Download original' })).toHaveCount(0);
  await expect(viewer(guest).getByRole('button', { name: 'Favourite' })).toHaveCount(0);
  await guest.keyboard.press('Escape');
  // the owner turns the link off
  await share.getByRole('button', { name: 'Turn off' }).click();
  await expect(linkBox).toHaveCount(0);
  await guest.reload();
  await expect(guest.getByRole('heading', { name: 'This link no longer works' })).toBeVisible();
  expect(guestErrors).toEqual([]);
  await guestCtx.close();

  // delete the album: the photos stay
  await share.getByRole('button', { name: 'Done' }).click();
  const before = (await api(page, '/api/libraries'))[0].assets;
  await page.getByRole('button', { name: 'Delete album' }).click();
  await expect(page).toHaveURL(/\/albums$/);
  await expect(page.getByText('No albums yet')).toBeVisible();
  expect((await api(page, '/api/libraries'))[0].assets).toBe(before);
});

test('F05-A2 keyboard: Space picks a photo, Escape clears; archive from the bar', async ({ page }) => {
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  const circle = page.getByRole('button', { name: /^Select / }).nth(3);
  await circle.focus();
  await page.keyboard.press('Space');
  await expect(circle).toHaveAttribute('aria-pressed', 'true');
  await page.keyboard.press('Escape');
  await expect(page.getByRole('toolbar', { name: 'Selected photos' })).toBeHidden();
  // archive two from the bar: they leave the timeline; bring them back from Archive
  const count = async () => (await api<any[]>(page, '/api/timeline/buckets')).reduce((s, b) => s + b.count, 0);
  const before = await count();
  await page.getByRole('button', { name: /^Select / }).nth(4).click();
  await page.getByRole('button', { name: /^Select / }).nth(5).click();
  await page.getByRole('toolbar', { name: 'Selected photos' }).getByRole('button', { name: 'Archive' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Archived 2.' })).toBeVisible();
  expect(await count()).toBe(before - 2);
  await page.goto('/archive');
  await expect(tiles(page)).toHaveCount(2);
  // BUG-008: unarchiving moves on to the next archived photo, and closes after the last one
  await tiles(page).first().click();
  const first = page.url();
  await viewer(page).getByRole('button', { name: 'Unarchive' }).click();
  await expect(page).not.toHaveURL(first);
  await expect(viewer(page).getByRole('button', { name: 'Unarchive' })).toBeEnabled();
  await viewer(page).getByRole('button', { name: 'Unarchive' }).click();
  await expect(viewer(page)).toBeHidden();
  await expect(page.getByText('Nothing archived')).toBeVisible();
  expect(await count()).toBe(before);
});

test('F05-A3 long-press picks a photo on a phone @mobile', async ({ page }) => {
  test.skip(page.viewportSize()!.width >= 768, 'touch only');
  await page.goto('/');
  await expect(tiles(page).first()).toBeVisible();
  const box = (await tiles(page).nth(1).boundingBox())!;
  const cdp = await page.context().newCDPSession(page);
  const pt = [{ x: box.x + box.width / 2, y: box.y + box.height / 2, id: 1 }];
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchStart', touchPoints: pt });
  await page.waitForTimeout(700);
  await cdp.send('Input.dispatchTouchEvent', { type: 'touchEnd', touchPoints: [] });
  const bar = page.getByRole('toolbar', { name: 'Selected photos' });
  await expect(bar).toContainText('1 selected');
  await expect(viewer(page)).toBeHidden(); // the long-press did not also open the photo
  await page.waitForTimeout(600);
  await tiles(page).nth(2).tap(); // while picking, a tap picks
  await expect(bar).toContainText('2 selected');
  await page.screenshot({ path: `${SHOTS}/S03-selecting-mobile.png` });
  await bar.getByRole('button', { name: 'Clear selection' }).tap();
  await expect(bar).toBeHidden();
});

test('F05-A4 add to album from the viewer', async ({ page }) => {
  const sept = (await api<any[]>(page, '/api/timeline?month=2026-09')).filter((t) => t.type === 'image');
  await page.goto(`/photos/${sept[0].id}`);
  await viewer(page).getByRole('button', { name: 'Add to album' }).click();
  const picker = page.getByRole('dialog', { name: /Add 1 to an album/ });
  await picker.getByLabel('New album').fill('Viewer pick');
  await picker.getByRole('button', { name: 'Create' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Made the album Viewer pick with 1 photo' })).toBeVisible();
  const albums = await api<any[]>(page, '/api/albums');
  const a = albums.find((x) => x.name === 'Viewer pick');
  expect(a.count).toBe(1);
  await page.request.delete(`/api/albums/${a.id}`);
});
