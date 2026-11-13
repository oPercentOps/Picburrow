<script lang="ts">
  import { formatMonth, type Bucket } from '$lib/api';

  let { buckets }: { buckets: Bucket[] } = $props();

  let rail: HTMLElement;
  let height = $state(0);
  let current = $state(0); // index of the month at the top of the screen
  let dragging = $state(false);

  // each month gets rail space in proportion to its photo count, like the timeline itself
  const starts = $derived.by(() => {
    const total = buckets.reduce((s, b) => s + b.count, 0) || 1;
    let acc = 0;
    return buckets.map((b) => { const f = acc / total; acc += b.count; return f; });
  });
  const years = $derived.by(() => {
    const out: { year: string; top: number }[] = [];
    buckets.forEach((b, i) => {
      const top = starts[i] * height;
      if (b.month.slice(0, 4) !== buckets[i - 1]?.month.slice(0, 4) && top - (out.at(-1)?.top ?? -99) > 18) out.push({ year: b.month.slice(0, 4), top });
    });
    return out;
  });

  function jump(i: number) {
    current = Math.max(0, Math.min(buckets.length - 1, i));
    document.getElementById(`m-${buckets[current].month}`)?.scrollIntoView({ block: 'start' });
  }
  function fromPointer(e: PointerEvent) {
    const r = rail.getBoundingClientRect();
    const f = Math.max(0, Math.min(1, (e.clientY - r.top) / r.height));
    let i = 0;
    while (i + 1 < starts.length && starts[i + 1] <= f) i++;
    if (i !== current || !dragging) jump(i);
  }

  // follow the page: which month section is at the top of the viewport
  $effect(() => {
    let raf = 0;
    const onScroll = () => {
      if (dragging || raf) return;
      raf = requestAnimationFrame(() => {
        raf = 0;
        // at the very bottom the oldest month can't reach the top of the screen, but it is the current one (BUG-006)
        const atEnd = innerHeight + scrollY >= document.documentElement.scrollHeight - 2;
        const i = atEnd ? buckets.length - 1 : buckets.findIndex((b) => (document.getElementById(`m-${b.month}`)?.getBoundingClientRect().bottom ?? 0) > 80);
        if (i >= 0) current = i;
      });
    };
    addEventListener('scroll', onScroll, { passive: true });
    return () => { removeEventListener('scroll', onScroll); cancelAnimationFrame(raf); };
  });

  function onkey(e: KeyboardEvent) {
    const step = { ArrowUp: -1, PageUp: -1, ArrowDown: 1, PageDown: 1, Home: -buckets.length, End: buckets.length }[e.key];
    if (step === undefined) return;
    e.preventDefault();
    jump(current + step);
  }
</script>

<div class="scrubber" bind:this={rail} bind:clientHeight={height} role="slider" tabindex="0"
  aria-label="Jump to month" aria-orientation="vertical" aria-valuemin={0} aria-valuemax={buckets.length - 1}
  aria-valuenow={current} aria-valuetext={formatMonth(buckets[current].month)}
  onpointerdown={(e) => { dragging = true; rail.setPointerCapture(e.pointerId); fromPointer(e); }}
  onpointermove={(e) => dragging && fromPointer(e)}
  onpointerup={() => (dragging = false)} onpointercancel={() => (dragging = false)} onkeydown={onkey}>
  {#each years as y}<span class="year" style:top="{y.top}px" aria-hidden="true">{y.year}</span>{/each}
  <span class="thumb" style:top="{starts[current] * height}px" aria-hidden="true">
    {#if dragging}<span class="label">{formatMonth(buckets[current].month)}</span>{/if}
  </span>
</div>

<style>
  /* mobile: only the thumb takes touches, so taps on the photos underneath still open them */
  .scrubber { position: fixed; z-index: 5; right: 0; top: 64px; bottom: calc(var(--bottom-nav) + env(safe-area-inset-bottom) + var(--space-4));
    width: var(--touch-target); touch-action: none; user-select: none; pointer-events: none; }
  .scrubber:focus-visible { outline-offset: -2px; }
  .year { position: absolute; right: 4px; translate: 0 -50%; font: var(--weight-xs) var(--text-xs) var(--font-sans); color: var(--text-muted); display: none; }
  .thumb { position: absolute; right: 0; width: var(--touch-target); height: var(--touch-target); translate: 0 -50%; pointer-events: auto; cursor: ns-resize; }
  .thumb::after { content: ''; position: absolute; right: 6px; top: 50%; width: 6px; height: 32px; translate: 0 -50%;
    border-radius: var(--radius-pill); background: var(--accent); box-shadow: var(--shadow-pop); }
  .label { position: absolute; right: var(--touch-target); top: 50%; translate: 0 -50%; white-space: nowrap; padding: var(--space-2) var(--space-4);
    border-radius: var(--radius-md); background: var(--surface); color: var(--text); box-shadow: var(--shadow-pop); font: 600 var(--text-sm) var(--font-sans); }
  @media (min-width: 768px) {
    /* desktop: the whole rail is clickable; it sits in the page's right padding, not over photos */
    .scrubber { top: var(--space-6); bottom: var(--space-6); width: var(--space-8); pointer-events: auto; cursor: ns-resize; }
    .year { display: block; }
    .thumb { height: 4px; width: 36px; right: 4px; border-radius: var(--radius-pill); background: var(--accent); }
    .thumb::after { display: none; }
  }
</style>
