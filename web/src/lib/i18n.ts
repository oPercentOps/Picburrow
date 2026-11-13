import { fr } from './fr';

// The English text is the key. French when the browser or phone prefers French; anything not in fr.ts stays English.
// A key may carry context after "|" when one English word needs two French ones ("Archive|verb"); English drops it.
export const lang: 'fr' | 'en' = typeof navigator !== 'undefined' && /^fr\b/i.test(navigator.language) ? 'fr' : 'en';
if (typeof document !== 'undefined') document.documentElement.lang = lang; // screen readers pick the right voice

export function t(s: string, vars?: Record<string, string | number>): string {
  const out = (lang === 'fr' && fr[s]) || s.split('|')[0];
  return vars ? out.replace(/\{(\w+)\}/g, (m, k) => (k in vars ? String(vars[k]) : m)) : out;
}

// Server messages that carry a path or a number are matched by shape instead of exact text.
const patterns: [RegExp, string][] = [
  [/^The server isn't allowed through (.+) to reach that folder\. Give the picburrow user access to it \(README: Photo folder permissions\)$/,
    "Le serveur n'a pas le droit de traverser $1 pour atteindre ce dossier. Donnez-en l'accès à l'utilisateur picburrow (README : Photo folder permissions)"],
  [/^The library data \((.+)\) is on the SD card \((.+)\)\. .*$/,
    "Les données de la bibliothèque ($1) sont sur la carte SD ($2). Les aperçus seront lents et la carte s'use. Déplacez-les sur un SSD ou un disque USB : sudo dpkg-reconfigure picburrow (ou lancez Picburrow avec --data-dir)."],
  [/^Request failed \((\d+)\)$/, 'La requête a échoué ($1)'],
  [/^(\w+) must look like (.+)$/, '$1 doit ressembler à $2']
];

// tServer translates a message sent by the server (errors, warnings).
export function tServer(msg: string): string {
  if (lang !== 'fr') return msg;
  if (fr[msg]) return fr[msg];
  for (const [re, to] of patterns) if (re.test(msg)) return msg.replace(re, to);
  const i = msg.indexOf(': '); // "<folder>: <problem>" from the library checks
  if (i > 0 && msg.startsWith('/')) return msg.slice(0, i + 2) + tServer(msg.slice(i + 2));
  return msg;
}
