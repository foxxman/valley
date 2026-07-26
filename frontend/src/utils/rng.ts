/** LCG-генератор [0, 1) по целочисленному сиду. */
export function createRng(seed: number): () => number {
  let state = seed >>> 0;

  return () => {
    state = (state * 1664525 + 1013904223) >>> 0;
    return state / 0x100000000;
  };
}

/**
 * Детерминированный сид чанка из сида мира и координат (cx, cy).
 * Стабилен между сессиями при том же worldSeed.
 */
export function mixSeed(worldSeed: number, cx: number, cy: number): number {
  let h = worldSeed >>> 0;
  h = (Math.imul(h ^ (cx | 0), 0x9e3779b1) + (cy | 0)) >>> 0;
  h = (Math.imul(h ^ (cy | 0), 0x85ebca6b) + (cx | 0)) >>> 0;
  return h >>> 0;
}
