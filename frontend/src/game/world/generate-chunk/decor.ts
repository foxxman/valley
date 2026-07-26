import { DECOR_CHANCE, LILY_CHANCE } from '../../../const/chunk';
import { indexOf, pickOne } from '../../../utils/array';
import {
  FLOWER_KINDS,
  GRASS_KINDS,
  LILY_KINDS,
  type FloraKind,
} from '../flora';
import { isLandTile, TileType } from '../tiles';
import { isInSpawnPad } from './spawn';

/** Клетка воды без ортогональной суши — не берег, можно ставить кувшинки. */
function isInteriorWater(
  tiles: TileType[],
  chunkSize: number,
  x: number,
  y: number,
): boolean {
  const neighbors = [
    [x, y - 1],
    [x + 1, y],
    [x, y + 1],
    [x - 1, y],
  ] as const;

  for (const [nx, ny] of neighbors) {
    if (nx < 0 || ny < 0 || nx >= chunkSize || ny >= chunkSize) {
      return false;
    }
    if (isLandTile(tiles[indexOf(chunkSize, nx, ny)]!)) {
      return false;
    }
  }
  return true;
}

/** Деревья и флора на траве; кувшинки — только на внутренней воде (не на берегу). */
export function placeDecor(
  tiles: TileType[],
  trees: boolean[],
  flora: Array<FloraKind | null>,
  chunkSize: number,
  rng: () => number,
  isOrigin: boolean,
  spawnCx: number,
  spawnCy: number,
): void {
  for (let y = 0; y < chunkSize; y += 1) {
    for (let x = 0; x < chunkSize; x += 1) {
      const i = indexOf(chunkSize, x, y);
      const tile = tiles[i];

      if (tile === TileType.Water) {
        if (isInteriorWater(tiles, chunkSize, x, y) && rng() < LILY_CHANCE) {
          flora[i] = pickOne(rng, LILY_KINDS);
        }
        continue;
      }

      if (tile !== TileType.Grass) {
        continue;
      }
      if (isOrigin && isInSpawnPad(x, y, spawnCx, spawnCy)) {
        continue;
      }

      if (rng() < DECOR_CHANCE) {
        trees[i] = true;
        continue;
      }
      if (rng() < DECOR_CHANCE) {
        flora[i] = pickOne(rng, FLOWER_KINDS);
        continue;
      }
      if (rng() < DECOR_CHANCE) {
        flora[i] = pickOne(rng, GRASS_KINDS);
      }
    }
  }
}
