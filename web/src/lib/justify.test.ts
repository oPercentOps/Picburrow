// @ts-nocheck -- plain Node script (node --experimental-strip-types), not part of the app build
// Run: node --experimental-strip-types src/lib/justify.test.ts
import assert from 'node:assert/strict';
import { justify, aspectOf } from './justify.ts';

const rows = justify([1.5, 1.5, 0.75, 1, 1.5, 1.33], 600, 200, 2);
for (const r of rows.slice(0, -1)) {
  const w = r.reduce((s, b) => s + b.width, 0) + 2 * (r.length - 1);
  assert.ok(Math.abs(w - 600) < 0.01, `full row fills width, got ${w}`);
  assert.ok(r[0].height <= 200, 'full rows are no taller than the target');
}
assert.equal(rows.flat().length, 6, 'every tile placed once');
assert.deepEqual(rows.flat().map((b) => b.index), [0, 1, 2, 3, 4, 5], 'order kept');
assert.equal(rows.at(-1)![0].height, 200, 'last row not stretched');
assert.deepEqual(justify([], 600, 200, 2), []);
assert.equal(aspectOf(null, 100), 1);
assert.equal(aspectOf(10000, 100), 3);
console.log('justify: ok');
