// F07 Admin manages libraries (S13) and watches indexing (S14).
import { test, expect, axe, SHOTS, LIB, api, waitIndexed } from './helpers';
import { mkdtempSync, copyFileSync, rmSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';

test('F07-H2 add a folder, it gets scanned; remove it, its photo goes to trash @mobile', async ({ page }) => {
  const mobile = page.viewportSize()!.width < 768;
  await page.goto('/admin/libraries');
  const folders = page.getByLabel('Folders');
  await expect(folders).toHaveValue(LIB);
  await page.screenshot({ path: `${SHOTS}/S13-libraries${mobile ? '-mobile' : ''}.png`, fullPage: mobile });
  await axe(page);
  if (mobile) return; // the rest edits shared state; once is enough

  // F07-N4 a folder that doesn't exist: nothing is saved
  await folders.fill(`${LIB}\n/no/such/folder`);
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByRole('alert')).toContainText('/no/such/folder');
  await page.screenshot({ path: `${SHOTS}/S13-libraries-error.png` });
  // F07-N5 a broken pattern
  await folders.fill(LIB);
  await page.getByLabel('Skip files matching').fill('**/{broken');
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByRole('alert')).toContainText('exclusion pattern');
  expect((await api(page, '/api/libraries'))[0].exclusion_globs).toEqual(['**/@eaDir/**', '**/.*/**']);
  await page.reload();

  const extra = mkdtempSync(join(tmpdir(), 'rg-extra-'));
  copyFileSync(join(LIB, 'DCIM/Camera/IMG_20260103_1.jpg'), join(extra, 'from_tablet.jpg'));
  const before = (await api(page, '/api/libraries'))[0];
  await page.getByLabel('Folders').fill(`${LIB}\n${extra}`);
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Saved. Scanning for changes now.' })).toBeVisible();
  await waitIndexed(page);
  expect((await api(page, '/api/libraries'))[0].assets).toBe(before.assets + 1);

  // F07-E4 removing a folder asks first (accepted by the guard) and its photo moves to trash
  await page.reload();
  await page.getByLabel('Folders').fill(LIB);
  await page.getByRole('button', { name: 'Save' }).click();
  await expect(page.getByRole('status').filter({ hasText: 'Saved.' })).toBeVisible();
  await waitIndexed(page);
  const after = (await api(page, '/api/libraries'))[0];
  expect(after.assets).toBe(before.assets);
  expect(after.missing).toBe(before.missing + 1);
  rmSync(extra, { recursive: true, force: true });
});

test('F07-H3 indexing page lists the file without a preview', async ({ page }) => {
  await page.goto('/admin/jobs');
  await expect(page.getByRole('heading', { name: /Files without a preview \(1\)/ })).toBeVisible();
  await expect(page.getByRole('listitem').filter({ hasText: 'IMG_broken.jpg' })).toHaveCount(1);
  await page.screenshot({ path: `${SHOTS}/S14-indexing.png` });
  await axe(page);
});

test('F07-E5 "Scan now" starts a scan', async ({ page }) => {
  await page.goto('/admin/libraries');
  await page.getByRole('button', { name: 'Scan now' }).click();
  await expect(page.getByRole('button', { name: 'Scan started' })).toBeVisible();
  await waitIndexed(page);
});
