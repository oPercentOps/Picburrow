// @ts-nocheck -- plain Node script (node --experimental-strip-types), not part of the app build
// Run: node --experimental-strip-types src/lib/fr.test.ts
// Every t('…') / tr('…') key in the app has French, with the same {placeholders}.
import assert from 'node:assert/strict';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { fr } from './fr.ts';

const files = (dir: string): string[] =>
  readdirSync(dir).flatMap((f) => { const p = join(dir, f); return statSync(p).isDirectory() ? files(p) : /\.(svelte|ts)$/.test(p) ? [p] : []; });

const lit = `'((?:\\\\.|[^'\\\\])*)'|"((?:\\\\.|[^"\\\\])*)"`;
const call = new RegExp(`\\b(?:t|tr)\\(\\s*(?:[^'"()]*\\?\\s*(?:${lit})\\s*:\\s*)?(?:${lit})`, 'g');
const keys = new Set<string>();
for (const f of files(new URL('..', import.meta.url).pathname)) {
  if (f.endsWith('.test.ts') || f.endsWith('fr.ts')) continue;
  for (const m of readFileSync(f, 'utf8').matchAll(call)) for (const k of m.slice(1)) if (k !== undefined) keys.add(k.replace(/\\(['"])/g, '$1'));
}
assert.ok(keys.size > 150, `found only ${keys.size} keys: the scanner is broken`);
const holes = (s: string) => [...s.matchAll(/\{(\w+)\}/g)].map((m) => m[1]).sort().join(',');
const missing = [...keys].filter((k) => !(k in fr));
assert.deepEqual(missing, [], 'strings without French');
for (const [en, f] of Object.entries(fr)) assert.equal(holes(f), holes(en), `placeholders differ for "${en}"`);
console.log(`fr.test: ${keys.size} strings, all translated`);
