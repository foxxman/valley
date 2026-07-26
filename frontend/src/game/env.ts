function readNumber(value: string | undefined, fallback: number): number {
  if (value === undefined || value === '') {
    return fallback;
  }

  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : fallback;
}

export const gameEnv = {
  tileSize: readNumber(import.meta.env.VITE_TILE_SIZE, 32),
  worldSeed: readNumber(import.meta.env.VITE_WORLD_SEED, 42),
  chunkSize: readNumber(import.meta.env.VITE_CHUNK_SIZE, 16),
  playerSpeed: readNumber(import.meta.env.VITE_PLAYER_SPEED, 160),
} as const;
