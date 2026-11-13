// Sign-in and sessions (S01).
import { test, expect, axe, USER, SHOTS } from './helpers';

test.describe('sign in', () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  test('S01-H1 sign in, S01-N1 wrong password @mobile', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveURL(/\/login$/);
    await axe(page);
    await page.getByLabel('Username').fill(USER.name);
    await page.getByLabel('Password').fill('wrong password!');
    await page.getByRole('button', { name: 'Sign in' }).click();
    await expect(page.getByRole('alert')).toHaveText('Wrong username or password');
    await expect(page.getByLabel('Password')).toHaveAttribute('aria-invalid', 'true');
    if (page.viewportSize()!.width < 768) await page.screenshot({ path: `${SHOTS}/S01-login-mobile.png` });
    await page.getByLabel('Password').fill(USER.password);
    await page.getByRole('button', { name: 'Sign in' }).click();
    await expect(page.getByRole('heading', { name: 'Photos', level: 1 })).toBeVisible();
  });

  test('S01-E1 a deep link without a session goes to sign in', async ({ page }) => {
    await page.goto('/admin/libraries');
    await expect(page).toHaveURL(/\/login$/);
  });
});

test('S01-E2 expired or forged session sends the user to sign in', async ({ page, context }) => {
  await page.goto('/');
  await expect(page.getByRole('heading', { name: 'Photos', level: 1 })).toBeVisible();
  await context.addCookies([{ name: 'rg_session', value: 'forged-or-expired', url: page.url() }]);
  await page.goto('/favorites');
  await expect(page).toHaveURL(/\/login$/);
});

test('S01-E3 sign out ends the session', async ({ browser }) => {
  // own context: signing out must not kill the shared session the other specs use
  const ctx = await browser.newContext();
  const page = await ctx.newPage();
  await page.goto('/login');
  await page.getByLabel('Username').fill(USER.name);
  await page.getByLabel('Password').fill(USER.password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await page.getByRole('button', { name: /^Sign out/ }).click();
  await expect(page).toHaveURL(/\/login$/);
  expect((await page.request.get('/api/me')).status()).toBe(401);
  await ctx.close();
});
