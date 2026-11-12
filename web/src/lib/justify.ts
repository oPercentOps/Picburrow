// Justified rows: fill each row to the container width at roughly the target height, keeping aspect ratios.
export type Box = { index: number; width: number; height: number };

export function justify(aspects: number[], width: number, target: number, gap: number): Box[][] {
  const rows: Box[][] = [];
  let row: number[] = [];
  let sum = 0;
  const flush = (h: number) => {
    rows.push(row.map((i) => ({ index: i, width: aspects[i] * h, height: h })));
    row = [];
    sum = 0;
  };
  aspects.forEach((a, i) => {
    row.push(i);
    sum += a;
    const h = (width - gap * (row.length - 1)) / sum;
    if (h <= target) flush(h);
  });
  if (row.length) flush(target); // last row keeps the target height, not stretched
  return rows;
}

export const aspectOf = (w: number | null, h: number | null) =>
  w && h ? Math.min(Math.max(w / h, 0.4), 3) : 1; // clamp panoramas and slivers so no tile becomes unusable
