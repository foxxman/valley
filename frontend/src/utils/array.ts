/** Индекс в плоском массиве размера `size × size` (ряд-major). */
export function indexOf(size: number, x: number, y: number): number {
  return y * size + x;
}

/** Случайный элемент из непустого массива. */
export function pickOne<T>(rng: () => number, items: readonly T[]): T {
  return items[Math.floor(rng() * items.length)]!;
}
