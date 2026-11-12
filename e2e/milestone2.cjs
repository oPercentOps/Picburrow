// Milestone 2 end-to-end: run right after e2e/slice.cjs, against the same server (already set up, user ann).
// Usage: node e2e/milestone2.cjs <library-dir> [base-url] [screenshot-dir]
const { chromium } = require('playwright');
const assert = require('assert');
const fs = require('fs'), path = require('path'), os = require('os');
const LIB = process.argv[2], BASE = process.argv[3] || 'http://127.0.0.1:8090', OUT = (process.argv[4] || 'replica/clone-screens') + '/';
if (!LIB) { console.error('usage: node e2e/milestone2.cjs <library-dir> [base-url] [screenshot-dir]'); process.exit(2); }
const errors = [];
const watch = (p, tag) => {
  p.on('console', m => m.type() === 'error' && !/status of 4\d\d/.test(m.text()) && errors.push(`${tag}: ${m.text()}`));
  p.on('pageerror', e => errors.push(`${tag}: ${e.message}`));
  p.on('response', r => r.status() >= 500 && errors.push(`${tag}: ${r.status()} ${r.url()}`));
  p.on('dialog', d => d.accept());
};
const login = async (p) => {
  await p.goto(BASE + '/login'); await p.fill('#u', 'ann'); await p.fill('#p', 'correct horse');
  await p.click('button.primary'); await p.waitForURL(BASE + '/');
};
const waitIdle = async (p) => {
  for (let i = 0; i < 240; i++) {
    const j = await (await p.request.get(BASE + '/api/jobs')).json();
    if (!j.scanning && j.thumbs_todo === 0) return j;
    await p.waitForTimeout(500);
  }
  throw new Error('indexing did not finish');
};

(async () => {
  const b = await chromium.launch();
  const ctx = await b.newContext({ viewport: { width: 1440, height: 900 }, colorScheme: 'light' });
  const p = await ctx.newPage(); watch(p, 'desktop');
  await login(p); await p.waitForSelector('a.tile img');

  // ---- favourite from the viewer, keyboard F, then the favourites page
  const sept = await (await p.request.get(BASE + '/api/timeline?month=2026-09')).json();
  await p.goto(BASE + '/photos/' + sept[0].id);
  const heart = p.getByRole('button', { name: 'Favourite' });
  await heart.waitFor();
  await p.keyboard.press('f');
  await p.waitForFunction(() => document.querySelector('[aria-label="Favourite"]')?.getAttribute('aria-pressed') === 'true');
  await p.keyboard.press('ArrowRight'); await p.waitForTimeout(300);
  await heart.click();
  await p.waitForFunction(() => document.querySelector('[aria-label="Favourite"]')?.getAttribute('aria-pressed') === 'true');
  await p.goto(BASE + '/favorites'); await p.waitForSelector('a.tile img');
  assert.equal(await p.locator('a.tile').count(), 2, 'two favourites');
  assert.match(await p.locator('a.tile').first().getAttribute('aria-label'), /favourite/);
  await p.screenshot({ path: OUT + 'S09-favourites.png' });
  // open from favourites: the viewer steps through favourites only, unfavouriting removes it from the page
  await p.locator('a.tile').first().click(); await p.getByRole('dialog').locator('img').waitFor();
  assert.equal(await p.getByRole('button', { name: 'Next' }).count(), 1, 'next within favourites');
  await p.keyboard.press('ArrowRight'); await p.waitForTimeout(300);
  assert.equal(await p.getByRole('button', { name: 'Next' }).count(), 0, 'last favourite has no next');
  await p.keyboard.press('f'); await p.waitForTimeout(300);
  await p.keyboard.press('Escape'); await p.waitForURL(BASE + '/favorites');
  assert.equal(await p.locator('a.tile').count(), 1, 'unfavourited photo leaves the page');

  // ---- archive: viewer moves on, photo leaves the timeline, shows in archive
  const before = (await (await p.request.get(BASE + '/api/timeline/buckets')).json()).find(x => x.month === '2026-09').count;
  await p.goto(BASE + '/photos/' + sept[2].id); await p.getByRole('button', { name: 'Archive' }).click();
  await p.waitForURL(u => !u.pathname.endsWith('/' + sept[2].id));
  const after = (await (await p.request.get(BASE + '/api/timeline/buckets')).json()).find(x => x.month === '2026-09').count;
  assert.equal(after, before - 1, 'archived photo left the timeline');
  await p.goto(BASE + '/archive'); await p.waitForSelector('a.tile img');
  assert.equal(await p.locator('a.tile').count(), 1);
  await p.screenshot({ path: OUT + 'S10-archive.png' });

  // ---- folders: root -> DCIM -> Camera, breadcrumb, viewer follows the folder
  await p.goto(BASE + '/folders');
  await p.locator('.dirs a').first().waitFor();
  assert.equal(await p.locator('.dirs a').count(), 1, 'one library root');
  await p.screenshot({ path: OUT + 'S06-folders-root.png' });
  await p.locator('.dirs a').first().click(); await p.getByRole('link', { name: /^DCIM/ }).click();
  await p.getByRole('link', { name: /^Camera/ }).click();
  await p.waitForSelector('a.tile img'); await p.waitForTimeout(500);
  assert.equal(await p.locator('h2.day').count(), 0, 'folder view has no day headers');
  assert.equal(await p.locator('nav[aria-label="Breadcrumb"] a[aria-current="page"]').innerText(), 'Camera');
  await p.screenshot({ path: OUT + 'S06-folder.png' });
  await p.locator('a.tile').first().click(); await p.getByRole('dialog').locator('img, video').first().waitFor();
  const u1 = p.url(); await p.keyboard.press('ArrowRight'); await p.waitForTimeout(200);
  assert.notEqual(p.url(), u1, 'next inside the folder');
  await p.keyboard.press('Escape');
  await p.goto(BASE + '/folders/etc'); await p.getByRole('alert').waitFor();

  // ---- scrubber: keyboard End jumps to the oldest month
  await p.goto(BASE + '/'); await p.waitForSelector('a.tile img');
  const slider = p.getByRole('slider', { name: 'Jump to month' });
  await slider.focus(); await p.keyboard.press('End'); await p.waitForTimeout(600);
  assert.equal(await slider.getAttribute('aria-valuetext'), 'January 2026');
  const top = await p.evaluate(() => document.getElementById('m-2026-01').getBoundingClientRect().top);
  assert(Math.abs(top) < 60, `oldest month scrolled to the top (top=${top})`);
  await p.screenshot({ path: OUT + 'S03-scrubbed.png' });
  // drag on the rail
  const box = await slider.boundingBox();
  await p.mouse.move(box.x + box.width / 2, box.y + 5); await p.mouse.down();
  await p.mouse.move(box.x + box.width / 2, box.y + box.height * 0.5, { steps: 5 }); await p.mouse.up();
  assert.notEqual(await slider.getAttribute('aria-valuetext'), 'January 2026', 'dragging moves the timeline');

  // ---- libraries: validation, then add a folder and see it indexed
  await p.goto(BASE + '/admin/libraries'); await p.locator('textarea').first().waitFor();
  await p.screenshot({ path: OUT + 'S13-libraries.png' });
  const paths = p.locator('textarea').first();
  await paths.fill(LIB + '\n/no/such/folder'); await p.click('button.primary');
  await p.getByRole('alert').waitFor();
  assert.match(await p.getByRole('alert').innerText(), /no\/such\/folder/);
  await p.screenshot({ path: OUT + 'S13-libraries-error.png' });
  const extra = fs.mkdtempSync(path.join(os.tmpdir(), 'rg-extra-'));
  fs.copyFileSync(path.join(LIB, 'DCIM/Camera/IMG_20260103_1.jpg'), path.join(extra, 'from_tablet.jpg'));
  const count0 = (await (await p.request.get(BASE + '/api/libraries')).json())[0].assets;
  await paths.fill(LIB + '\n' + extra); await p.click('button.primary');
  await p.getByText('Saved. Scanning for changes now.').waitFor();
  await waitIdle(p);
  const count1 = (await (await p.request.get(BASE + '/api/libraries')).json())[0].assets;
  assert.equal(count1, count0 + 1, 'the added folder was scanned');
  // removing it again asks first (dialog auto-accepted) and its photo goes to trash on the next scan
  await p.reload(); await p.locator('textarea').first().fill(LIB); await p.click('button.primary');
  await p.getByText('Saved. Scanning for changes now.').waitFor(); await waitIdle(p);
  assert.equal((await (await p.request.get(BASE + '/api/libraries')).json())[0].missing, 1, 'the removed folder\'s photo is now missing (trash)');

  // ---- jobs page lists the corrupt file
  await p.goto(BASE + '/admin/jobs'); await p.getByText(/Files without a preview \(1\)/).waitFor();
  assert.equal(await p.locator('.errs .path', { hasText: 'IMG_broken.jpg' }).count(), 1, 'the corrupt file is listed');
  await p.screenshot({ path: OUT + 'S14-indexing.png' });

  // ---- mobile: scrubber thumb doesn't steal taps from the right-hand photos
  const m = await b.newContext({ viewport: { width: 390, height: 844 }, colorScheme: 'dark', hasTouch: true, isMobile: true, deviceScaleFactor: 2 });
  const mp = await m.newPage(); watch(mp, 'mobile');
  await login(mp); await mp.waitForSelector('a.tile img'); await mp.waitForTimeout(500);
  const thumbBox = await mp.locator('.scrubber .thumb').boundingBox();
  const target = await mp.evaluate((ty) => {
    const t = [...document.querySelectorAll('a.tile')].map(a => a.getBoundingClientRect())
      .find(r => r.right > innerWidth - 10 && r.top > 80 && r.bottom < innerHeight - 80 && (r.bottom < ty - 30 || r.top > ty + 30));
    return t && { x: innerWidth - 12, y: (t.top + t.bottom) / 2 };
  }, thumbBox.y + thumbBox.height / 2);
  assert(target, 'a right-edge tile away from the thumb');
  await mp.touchscreen.tap(target.x, target.y);
  await mp.getByRole('dialog').waitFor();
  await mp.keyboard.press('Escape');
  await mp.goto(BASE + '/folders'); await mp.locator('.dirs a').first().waitFor();
  assert(!(await mp.evaluate(() => document.documentElement.scrollWidth > innerWidth)), 'folders: no horizontal overflow');
  await mp.screenshot({ path: OUT + 'S06-folders-mobile.png' });
  await mp.goto(BASE + '/admin/libraries'); await mp.locator('textarea').first().waitFor();
  assert(!(await mp.evaluate(() => document.documentElement.scrollWidth > innerWidth)), 'libraries: no horizontal overflow');
  await mp.screenshot({ path: OUT + 'S13-libraries-mobile.png', fullPage: true });

  await b.close();
  fs.rmSync(extra, { recursive: true, force: true });
  if (errors.length) { console.log('CONSOLE ERRORS:\n' + errors.join('\n')); process.exit(1); }
  console.log('milestone2: all checks passed');
})().catch(e => { console.error(e); process.exit(1); });
