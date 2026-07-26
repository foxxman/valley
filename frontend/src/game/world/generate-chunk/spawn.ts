import { SPAWN_PAD_RADIUS } from '../../../const/chunk';
import { indexOf } from '../../../utils/array';
import { TileType } from '../tiles';

/** Клетка внутри квадрата спавна вокруг центра. */
export function isInSpawnPad(
  x: number,
  y: number,
  spawnCx: number,
  spawnCy: number,
): boolean {
  return (
    Math.abs(x - spawnCx) <= SPAWN_PAD_RADIUS &&
    Math.abs(y - spawnCy) <= SPAWN_PAD_RADIUS
  );
}

/** Гарантирует траву в зоне спавна (3×3 вокруг центра). */
export function clearSpawnPad(
  tiles: TileType[],
  chunkSize: number,
  spawnCx: number,
  spawnCy: number,
): void {
  for (let dy = -SPAWN_PAD_RADIUS; dy <= SPAWN_PAD_RADIUS; dy += 1) {
    for (let dx = -SPAWN_PAD_RADIUS; dx <= SPAWN_PAD_RADIUS; dx += 1) {
      const x = spawnCx + dx;
      const y = spawnCy + dy;
      if (x < 0 || y < 0 || x >= chunkSize || y >= chunkSize) {
        continue;
      }
      tiles[indexOf(chunkSize, x, y)] = TileType.Grass;
    }
  }
}
