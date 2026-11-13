import { test as base, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { resolve } from 'node:path';

export const LIB = resolve('.fixtures');
export const SHOTS = resolve('../replica/clone-screens');
export const USER = { name: 'ann', password: 'correct horse' };

// Answer the next prompt() on this page (the guard handles every dialog, so tests don't add their own handlers).
const promptAnswers = new WeakMap<Page, string>();
export const answerPrompt = (page: Page, text: string) => promptAnswers.set(page, text);

// Every test fails on a console error (except expected 4xx resource errors the UI handles) or any 5xx response.
// Native confirm() dialogs are accepted.
export const test = base.extend<{ guard: void }>({
  guard: [
    async ({ page }, use) => {
      const problems: string[] = [];
      page.on('console', (m) => {
        if (m.type() === 'error' && !/status of 4\d\d/.test(m.text())) problems.push(`console: ${m.text()}`);
      });
      page.on('pageerror', (e) => problems.push(`page error: ${e.message}`));
      page.on('response', (r) => {
        if (r.status() >= 500) problems.push(`${r.status()} ${r.url()}`);
      });
      page.on('dialog', (d) => {
        if (d.type() === 'confirm') return d.accept();
        if (d.type() === 'prompt' && promptAnswers.has(page)) {
          const a = promptAnswers.get(page)!;
          promptAnswers.delete(page);
          return d.accept(a);
        }
        problems.push(`unexpected ${d.type()}: ${d.message()}`);
        return d.dismiss();
      });
      await use();
      expect(problems, 'console errors, page errors, 5xx or unexpected dialogs').toEqual([]);
    },
    { auto: true }
  ]
});
export { expect };

// WCAG 2.1 A/AA scan of the current screen.
export async function axe(page: Page, exclude: string[] = []) {
  let b = new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa']);
  for (const e of exclude) b = b.exclude(e);
  const { violations } = await b.analyze();
  expect(violations.map((v) => `${v.id}: ${v.help} (${v.nodes.length}) ${v.nodes[0]?.target}`)).toEqual([]);
}

export async function api<T = any>(page: Page, path: string): Promise<T> {
  const r = await page.request.get(path);
  expect(r.status(), path).toBe(200);
  return r.json();
}

export async function waitIndexed(page: Page) {
  await expect
    .poll(async () => {
      const j = await api(page, '/api/jobs');
      return !j.scanning && j.thumbs_todo === 0;
    }, { timeout: 180_000, intervals: [500] })
    .toBe(true);
}

// first photo of a month that is an image (not the video or the corrupt file)
export async function photoIn(page: Page, month: string) {
  const tiles = await api<any[]>(page, `/api/timeline?month=${month}`);
  return tiles.filter((t) => t.type === 'image' && !t.err);
}

export const tiles = (page: Page) => page.getByRole('link', { name: /^(Photo|Video) from/ });
export const viewer = (page: Page) => page.getByRole('dialog', { name: 'Photo viewer' });
