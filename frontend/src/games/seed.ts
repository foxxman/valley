export const minWorldSeed = 0;
export const maxWorldSeed = 4_294_967_295;

export const worldSeedHint =
  'Необязательно. Целое число от 0 до 4 294 967 295. Одинаковое число всегда собирает одну и ту же карту. Пустое поле — сервер выберет сид сам.';

export function parseWorldSeed(raw: string): { ok: true; seed: number | null } | { ok: false } {
  const trimmed = raw.trim();
  if (trimmed === '') {
    return { ok: true, seed: null };
  }
  if (!/^\d+$/.test(trimmed)) {
    return { ok: false };
  }
  const seed = Number(trimmed);
  if (!Number.isSafeInteger(seed) || seed < minWorldSeed || seed > maxWorldSeed) {
    return { ok: false };
  }
  return { ok: true, seed };
}
