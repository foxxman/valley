import {
  POND_CELLS_MAX,
  POND_CELLS_MIN,
  POND_INSET,
  POND_PLACE_ATTEMPTS,
  POND_SPAWN_CLEAR_RADIUS,
} from '../../../const/chunk';
import { indexOf, pickOne } from '../../../utils/array';
import { TileType } from '../tiles';

const Bit = {
  N: 1,
  E: 2,
  S: 4,
  W: 8,
} as const;

/** Маски, для которых есть shore-тайлы (прямой / внешний угол / чистая вода). */
const SHORE_SUPPORTED_MASKS = new Set<number>([
  0,
  Bit.N,
  Bit.E,
  Bit.S,
  Bit.W,
  Bit.N | Bit.W,
  Bit.N | Bit.E,
  Bit.S | Bit.E,
  Bit.S | Bit.W,
]);

function blocksSpawn(
  x: number,
  y: number,
  isOrigin: boolean,
  spawnCx: number,
  spawnCy: number,
): boolean {
  return (
    isOrigin &&
    Math.abs(x - spawnCx) <= POND_SPAWN_CLEAR_RADIUS &&
    Math.abs(y - spawnCy) <= POND_SPAWN_CLEAR_RADIUS
  );
}

function cellKey(x: number, y: number): string {
  return `${x},${y}`;
}

/** Ортогональная суша = всё, что не вода озера (включая край чанка). */
function cardinalLandMask(waterSet: Set<string>, x: number, y: number): number {
  let mask = 0;
  if (!waterSet.has(cellKey(x, y - 1))) mask |= Bit.N;
  if (!waterSet.has(cellKey(x + 1, y))) mask |= Bit.E;
  if (!waterSet.has(cellKey(x, y + 1))) mask |= Bit.S;
  if (!waterSet.has(cellKey(x - 1, y))) mask |= Bit.W;
  return mask;
}

/**
 * Убирает клетки, которым нужен channel/deadend (вода без корректного берега).
 * Повторяет, пока все оставшиеся клетки не будут с поддерживаемой маской.
 */
function erodeUnsupportedShoreCells(waterSet: Set<string>): void {
  let changed = true;
  while (changed) {
    changed = false;
    for (const key of [...waterSet]) {
      const [xs, ys] = key.split(',');
      const x = Number(xs);
      const y = Number(ys);
      if (!SHORE_SUPPORTED_MASKS.has(cardinalLandMask(waterSet, x, y))) {
        waterSet.delete(key);
        changed = true;
      }
    }
  }
}

/**
 * Ставит одно озеро неправильной формы (4-связный blob) или ничего.
 * После роста срезает тонкие места, чтобы у каждой воды был валидный берег.
 */
export function tryPlacePond(
  tiles: TileType[],
  chunkSize: number,
  rng: () => number,
  isOrigin: boolean,
  spawnCx: number,
  spawnCy: number,
): void {
  const usable = chunkSize - POND_INSET * 2;
  if (usable < 3) {
    return;
  }

  const targetCells =
    POND_CELLS_MIN + Math.floor(rng() * (POND_CELLS_MAX - POND_CELLS_MIN + 1));

  for (let attempt = 0; attempt < POND_PLACE_ATTEMPTS; attempt += 1) {
    const startX = POND_INSET + Math.floor(rng() * usable);
    const startY = POND_INSET + Math.floor(rng() * usable);

    if (tiles[indexOf(chunkSize, startX, startY)] !== TileType.Grass) {
      continue;
    }
    if (blocksSpawn(startX, startY, isOrigin, spawnCx, spawnCy)) {
      continue;
    }

    const water: Array<{ x: number; y: number }> = [{ x: startX, y: startY }];
    const waterSet = new Set<string>([cellKey(startX, startY)]);
    const frontier: Array<{ x: number; y: number }> = [{ x: startX, y: startY }];

    const canBecomeWater = (x: number, y: number): boolean => {
      if (
        x < POND_INSET ||
        y < POND_INSET ||
        x >= chunkSize - POND_INSET ||
        y >= chunkSize - POND_INSET
      ) {
        return false;
      }
      if (tiles[indexOf(chunkSize, x, y)] !== TileType.Grass) {
        return false;
      }
      if (blocksSpawn(x, y, isOrigin, spawnCx, spawnCy)) {
        return false;
      }
      return !waterSet.has(cellKey(x, y));
    };

    while (water.length < targetCells && frontier.length > 0) {
      const fi = Math.floor(rng() * frontier.length);
      const cell = frontier[fi]!;
      const neighbors = [
        { x: cell.x + 1, y: cell.y },
        { x: cell.x - 1, y: cell.y },
        { x: cell.x, y: cell.y + 1 },
        { x: cell.x, y: cell.y - 1 },
      ].filter((n) => canBecomeWater(n.x, n.y));

      if (neighbors.length === 0) {
        frontier.splice(fi, 1);
        continue;
      }

      const next = pickOne(rng, neighbors);
      waterSet.add(cellKey(next.x, next.y));
      water.push(next);
      frontier.push(next);
    }

    erodeUnsupportedShoreCells(waterSet);

    if (waterSet.size < POND_CELLS_MIN) {
      continue;
    }

    for (const key of waterSet) {
      const [xs, ys] = key.split(',');
      tiles[indexOf(chunkSize, Number(xs), Number(ys))] = TileType.Water;
    }
    return;
  }
}
