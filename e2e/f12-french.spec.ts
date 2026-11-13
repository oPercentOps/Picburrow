// Issue #25: French when the browser prefers French; server messages translated too.
import { test, expect, axe, SHOTS, photoIn, viewer } from './helpers';

test.describe('in French', () => {
  test.use({ locale: 'fr-FR' });

  test('FR-1 navigation, timeline, viewer and details are in French @mobile', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('html')).toHaveAttribute('lang', 'fr');
    await expect(page.getByRole('heading', { level: 1, name: 'Photos' })).toBeVisible();
    await expect(page).toHaveTitle('Photos · Picburrow');
    await expect(page.getByRole('link', { name: 'Diaporama' })).toBeVisible();
    const nav = page.getByRole('navigation', { name: 'Principale' }).filter({ visible: true });
    await expect(nav.getByRole('link', { name: 'Dossiers' })).toBeVisible();
    await expect(page.getByRole('link', { name: /^Photo du / }).first()).toBeVisible();
    await page.screenshot({ path: `${SHOTS}/S19-french${page.viewportSize()!.width < 768 ? '-mobile' : ''}.png` });
    await axe(page);
    const [p] = await photoIn(page, '2026-09');
    await page.goto(`/photos/${p.id}`);
    const v = page.getByRole('dialog', { name: 'Visionneuse' });
    await expect(v).toBeVisible();
    await v.getByRole('button', { name: 'Détails' }).click();
    const info = page.getByRole('complementary', { name: 'Détails' });
    await expect(info).toContainText('Appareil');
    await expect(info).toContainText('(26 mm équiv. 24×36)');
    await axe(page);
  });

  test('FR-2 a message from the server is shown in French', async ({ browser }) => {
    const ctx = await browser.newContext({ locale: 'fr-FR', viewport: { width: 1440, height: 900 } });
    const page = await ctx.newPage();
    await page.goto('/login');
    await expect(page.getByRole('heading', { level: 1, name: 'Se connecter' })).toBeVisible();
    await page.getByLabel("Nom d'utilisateur").fill('nobody');
    await page.getByLabel('Mot de passe').fill('wrong password');
    await page.getByRole('button', { name: 'Se connecter' }).click();
    await expect(page.getByRole('alert')).toHaveText("Nom d'utilisateur ou mot de passe incorrect");
    await ctx.close();
  });
});
