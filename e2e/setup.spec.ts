// F07 (first run): an admin sets up the gallery on a fresh install.
import { test, expect, axe, LIB, USER, SHOTS, waitIndexed } from './helpers';

test('F07-H1 first-run setup indexes the folder and lands on the timeline', async ({ page, browser }) => {
  test.setTimeout(240_000); // indexes the whole fixture library
  await page.goto('/');
  await expect(page).toHaveURL(/\/setup$/);
  await expect(page.getByRole('heading', { name: 'Welcome to Picburrow' })).toBeVisible();
  await axe(page);
  await page.screenshot({ path: `${SHOTS}/S02-setup.png` });

  // F07-E1 empty form: the browser blocks submit, nothing is created
  await page.getByRole('button', { name: 'Show my photos' }).click();
  await expect(page).toHaveURL(/\/setup$/);

  // F07-N1 short password, F07-N2 folder that doesn't exist, F07-N3 relative path
  await page.getByLabel('Admin username').fill(USER.name);
  await page.getByLabel('Password').fill('short');
  await page.getByLabel('Photo folder on this device').fill(LIB);
  await page.getByLabel('Password').evaluate((e: HTMLInputElement) => e.removeAttribute('minlength')); // reach the server check
  await page.getByRole('button', { name: 'Show my photos' }).click();
  await expect(page.getByRole('alert')).toHaveText(/at least 8 characters/);
  await page.getByLabel('Password').fill(USER.password);
  await page.getByLabel('Photo folder on this device').fill('/does/not/exist');
  await page.getByRole('button', { name: 'Show my photos' }).click();
  await expect(page.getByRole('alert')).toHaveText(/doesn't exist/);
  await page.screenshot({ path: `${SHOTS}/S02-setup-error.png` });
  await page.getByLabel('Photo folder on this device').fill('photos/backup');
  await page.getByRole('button', { name: 'Show my photos' }).click();
  await expect(page.getByRole('alert')).toHaveText(/full path/);

  // F07-E2 double submit: one admin, and the user ends up on the timeline
  await page.getByLabel('Photo folder on this device').fill(LIB);
  // two clicks in the same tick, before the UI can disable the button
  await page.getByRole('button', { name: 'Show my photos' }).evaluate((b: HTMLButtonElement) => { b.click(); b.click(); });
  await expect(page).toHaveURL(/\/$/);

  // F01-E1 while indexing: the banner says so
  await expect(page.getByRole('heading', { name: 'Photos', level: 1 })).toBeVisible();
  await page.screenshot({ path: `${SHOTS}/S03-timeline-indexing.png` });

  // F07-E3 a second browser can't run setup again
  const other = await browser.newContext();
  const r = await other.request.post('/api/setup', { data: { username: 'eve', password: 'correct horse', path: LIB } });
  expect(r.status()).toBe(409);
  await other.close();

  await waitIndexed(page);
  await page.context().storageState({ path: '.auth.json' });
});
