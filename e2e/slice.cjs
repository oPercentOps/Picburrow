const { chromium } = require('playwright');
const assert = require('assert');
// End-to-end check of the vertical slice against a running server with a FRESH data dir.
// Usage: node e2e/slice.cjs <library-dir> [base-url] [screenshot-dir]
// The library must come from e2e/make-fixtures.sh (the test asserts its dates and its broken file).
const LIB = process.argv[2], BASE = process.argv[3] || 'http://127.0.0.1:8090', OUT = (process.argv[4] || 'replica/clone-screens') + '/';
if (!LIB) { console.error('usage: node e2e/slice.cjs <library-dir> [base-url] [screenshot-dir]'); process.exit(2); }
const errors = [];
const watch = (p, tag) => {
  // expected 4xx (bad setup path, wrong password, no-preview file) are shown in the UI; any 5xx is a bug
  p.on('console', m => m.type() === 'error' && !/status of 4\d\d/.test(m.text()) && errors.push(`${tag}: ${m.text()}`));
  p.on('response', r => r.status() >= 500 && errors.push(`${tag}: ${r.status()} ${r.url()}`));
  p.on('pageerror', e => errors.push(`${tag}: ${e.message}`));
};
const rowsFit = async (p, tag) => {
  const worst = await p.evaluate(() => Math.max(...[...document.querySelectorAll('.row')].map(r => r.scrollWidth - r.parentElement.clientWidth)));
  assert(worst <= 1, `${tag}: a justified row is ${worst}px wider than its container`);
};
const noOverflow = async (p, tag) => assert(!(await p.evaluate(() => document.documentElement.scrollWidth > innerWidth)), `${tag}: horizontal overflow`);
(async () => {
  const b = await chromium.launch();
  // ---- desktop, light: setup -> timeline -> viewer -> info
  const ctx = await b.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: 'light' });
  const p = await ctx.newPage(); watch(p, 'desktop');
  await p.goto(BASE + '/');
  await p.waitForURL('**/setup');
  await p.screenshot({ path: OUT + 'S02-setup.png' });
  await p.fill('#u', 'ann'); await p.fill('#p', 'correct horse'); await p.fill('#f', '/does/not/exist');
  await p.click('button.primary');
  await p.getByRole('alert').waitFor();
  assert.match(await p.getByRole('alert').innerText(), /doesn't exist/);
  await p.screenshot({ path: OUT + 'S02-setup-error.png' });
  await p.fill('#f', LIB); await p.click('button.primary');
  await p.waitForURL(BASE + '/');
  await p.getByRole('status').first().waitFor({ timeout: 10000 }).catch(() => {});
  await p.screenshot({ path: OUT + 'S03-timeline-indexing.png' });
  const t0 = Date.now();
  for (;;) { // wait for indexing to finish
    const j = await (await p.request.get(BASE + '/api/jobs')).json();
    if (!j.scanning && j.thumbs_todo === 0) { console.log('indexed in', Date.now() - t0, 'ms', JSON.stringify(j)); break; }
    await p.waitForTimeout(500);
  }
  await p.reload(); await p.waitForSelector('a.tile img');
  await p.waitForTimeout(800);
  await noOverflow(p, 'S03 desktop'); await rowsFit(p, 'S03 desktop');
  await p.screenshot({ path: OUT + 'S03-timeline.png' });
  const tiles = await p.locator('a.tile').count(); console.log('tiles mounted near top:', tiles);
  assert(tiles > 10);
  assert(await p.getByText('No preview').count() >= 1, 'broken file shows a no-preview tile');
  // open viewer, scroll position kept on close
  await p.mouse.wheel(0, 600); await p.waitForTimeout(300);
  const scrollBefore = await p.evaluate(() => scrollY);
  // click a tile that is fully on screen (Playwright would otherwise scroll to it first)
  const onScreen = await p.evaluate(() => [...document.querySelectorAll('a.tile')].findIndex(a => { const r = a.getBoundingClientRect(); return r.top > 0 && r.bottom < innerHeight; }));
  assert(onScreen >= 0, 'a tile is on screen');
  await p.locator('a.tile').nth(onScreen).click();
  await p.waitForURL(/\/photos\/\d+$/);
  await p.getByRole('dialog').locator('img, video').first().waitFor();
  await p.keyboard.press('Escape');
  await p.waitForURL(BASE + '/');
  assert.equal(await p.getByRole('dialog').count(), 0, 'viewer closed');
  assert(Math.abs((await p.evaluate(() => scrollY)) - scrollBefore) < 5, 'scroll position kept');
  // a known camera photo: viewer, next, info
  const photo = (await (await p.request.get(BASE + '/api/timeline?month=2026-09')).json())[0];
  await p.goto(BASE + '/photos/' + photo.id);
  await p.getByRole('dialog').locator('img').waitFor(); await p.waitForTimeout(400);
  await p.screenshot({ path: OUT + 'S04-viewer.png' });
  const first = p.url();
  await p.keyboard.press('ArrowRight'); await p.waitForTimeout(300);
  assert.notEqual(p.url(), first, 'ArrowRight moves to the next photo');
  await p.keyboard.press('i');
  await p.getByRole('complementary', { name: 'Details' }).waitFor();
  assert.match(await p.locator('aside').innerText(), /Pixel 8/);
  await p.screenshot({ path: OUT + 'S05-info.png' });
  await p.keyboard.press('Escape'); await p.keyboard.press('Escape');
  await p.waitForURL(BASE + '/');
  // deep link + video
  let vid;
  for (const bk of await (await p.request.get(BASE + '/api/timeline/buckets')).json()) {
    vid = vid || (await (await p.request.get(BASE + '/api/timeline?month=' + bk.month)).json()).find(t => t.type === 'video');
  }
  await p.goto(BASE + '/photos/' + vid.id);
  await p.locator('video').waitFor();
  assert((await p.locator('video').getAttribute('src')).endsWith('/original'));
  await p.keyboard.press('Escape'); await p.waitForURL(BASE + '/');
  // ---- keyboard only: Tab to a tile, Enter opens, Escape closes, focus returns
  await p.goto(BASE + '/'); await p.waitForSelector('a.tile img');
  for (let i = 0; i < 40 && !(await p.evaluate(() => document.activeElement?.classList.contains('tile'))); i++) await p.keyboard.press('Tab');
  const label = await p.evaluate(() => document.activeElement.getAttribute('aria-label'));
  assert(label, 'a tile is keyboard focusable');
  await p.keyboard.press('Enter'); await p.getByRole('dialog').waitFor();
  await p.keyboard.press('Escape'); await p.waitForURL(BASE + '/');
  assert.equal(await p.evaluate(() => document.activeElement.getAttribute('aria-label')), label, 'focus returns to the tile');
  // ---- dark desktop viewer stub pages
  await p.goto(BASE + '/folders'); await p.getByRole('heading', { name: 'Folders' }).waitFor();
  // ---- mobile, dark: login -> timeline -> viewer -> info sheet
  const m = await b.newContext({ viewport: { width: 390, height: 844 }, colorScheme: 'dark', hasTouch: true, isMobile: true, deviceScaleFactor: 2 });
  const mp = await m.newPage(); watch(mp, 'mobile');
  await mp.goto(BASE + '/'); await mp.waitForURL('**/login');
  await mp.screenshot({ path: OUT + 'S01-login-mobile.png' });
  await mp.fill('#u', 'ann'); await mp.fill('#p', 'wrong password!');
  await mp.click('button.primary'); await mp.getByRole('alert').waitFor();
  await mp.fill('#p', 'correct horse'); await mp.click('button.primary');
  await mp.waitForURL(BASE + '/'); await mp.waitForSelector('a.tile img'); await mp.waitForTimeout(800);
  await noOverflow(mp, 'S03 mobile'); await rowsFit(mp, 'S03 mobile');
  const today = new Date().toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
  assert(await mp.getByText(today).count() >= 1, 'dateless videos land on their file date (today)');
  await mp.screenshot({ path: OUT + 'S03-timeline-mobile.png' });
  await mp.locator('a.tile').nth(2).tap();
  await mp.getByRole('dialog').locator('img, video').first().waitFor(); await mp.waitForTimeout(400);
  await mp.screenshot({ path: OUT + 'S04-viewer-mobile.png' });
  await mp.getByRole('button', { name: 'Details' }).tap();
  await mp.locator('aside').waitFor(); await mp.waitForTimeout(200);
  await mp.screenshot({ path: OUT + 'S05-info-mobile.png' });
  await b.close();
  if (errors.length) { console.log('CONSOLE ERRORS:\n' + errors.join('\n')); process.exit(1); }
  console.log('e2e: all checks passed');
})().catch(e => { console.error(e); process.exit(1); });
