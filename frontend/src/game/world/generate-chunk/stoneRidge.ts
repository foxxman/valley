import {
  RIDGE_PLACE_ATTEMPTS,
  RIDGE_STONE_MAX,
  RIDGE_STONE_MIN,
} from '../../../const/chunk';
import { indexOf, pickOne } from '../../../utils/array';
import { TileType } from '../tiles';
import { isInSpawnPad } from './spawn';

/**
 * Растит связанный камень (4–10 клеток), затем кольцо осколков
 * (рамка Чебышева толщиной 1–2). Заменяет только траву.
 */
export function tryPlaceStoneRidge(
  tiles: TileType[],
  chunkSize: number,
  rng: () => number,
  isOrigin: boolean,
  spawnCx: number,
  spawnCy: number,
): void {
  const border = 1 + (rng() < 0.5 ? 1 : 0);
  const targetStones =
    RIDGE_STONE_MIN + Math.floor(rng() * (RIDGE_STONE_MAX - RIDGE_STONE_MIN + 1));

  const stoneInset = border;
  const usable = chunkSize - stoneInset * 2;
  if (usable < 2) {
    return;
  }

  for (let attempt = 0; attempt < RIDGE_PLACE_ATTEMPTS; attempt += 1) {
    const startX = stoneInset + Math.floor(rng() * usable);
    const startY = stoneInset + Math.floor(rng() * usable);

    if (tiles[indexOf(chunkSize, startX, startY)] !== TileType.Grass) {
      continue;
    }
    if (isOrigin && isInSpawnPad(startX, startY, spawnCx, spawnCy)) {
      continue;
    }

    const stones: Array<{ x: number; y: number }> = [{ x: startX, y: startY }];
    const stoneSet = new Set<string>([`${startX},${startY}`]);
    const frontier: Array<{ x: number; y: number }> = [{ x: startX, y: startY }];

    const canBecomeStone = (x: number, y: number): boolean => {
      if (
        x < stoneInset ||
        y < stoneInset ||
        x >= chunkSize - stoneInset ||
        y >= chunkSize - stoneInset
      ) {
        return false;
      }
      if (tiles[indexOf(chunkSize, x, y)] !== TileType.Grass) {
        return false;
      }
      if (isOrigin && isInSpawnPad(x, y, spawnCx, spawnCy)) {
        return false;
      }
      return !stoneSet.has(`${x},${y}`);
    };

    while (stones.length < targetStones && frontier.length > 0) {
      const fi = Math.floor(rng() * frontier.length);
      const cell = frontier[fi]!;
      const neighbors = [
        { x: cell.x + 1, y: cell.y },
        { x: cell.x - 1, y: cell.y },
        { x: cell.x, y: cell.y + 1 },
        { x: cell.x, y: cell.y - 1 },
      ].filter((n) => canBecomeStone(n.x, n.y));

      if (neighbors.length === 0) {
        frontier.splice(fi, 1);
        continue;
      }

      const next = pickOne(rng, neighbors);
      stoneSet.add(`${next.x},${next.y}`);
      stones.push(next);
      frontier.push(next);
    }

    if (stones.length < RIDGE_STONE_MIN) {
      continue;
    }

    for (const { x, y } of stones) {
      tiles[indexOf(chunkSize, x, y)] = TileType.Stone;
    }

    for (const { x, y } of stones) {
      for (let dy = -border; dy <= border; dy += 1) {
        for (let dx = -border; dx <= border; dx += 1) {
          if (dx === 0 && dy === 0) {
            continue;
          }
          const nx = x + dx;
          const ny = y + dy;
          if (nx < 0 || ny < 0 || nx >= chunkSize || ny >= chunkSize) {
            continue;
          }
          if (isOrigin && isInSpawnPad(nx, ny, spawnCx, spawnCy)) {
            continue;
          }
          const i = indexOf(chunkSize, nx, ny);
          if (tiles[i] === TileType.Grass) {
            tiles[i] = TileType.StoneShards;
          }
        }
      }
    }

    return;
  }
}
