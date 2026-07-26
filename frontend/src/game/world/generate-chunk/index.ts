import { POND_CHANCE, RIDGE_CHANCE } from '../../../const/chunk';
import { createRng, mixSeed } from '../../../utils/rng';
import { TileType } from '../tiles';
import type { FloraKind } from '../flora';
import { placeDecor } from './decor';
import { tryPlacePond } from './pond';
import { clearSpawnPad } from './spawn';
import { tryPlaceStoneRidge } from './stoneRidge';
import type { ChunkTerrain } from './types';

export type { ChunkTerrain } from './types';

/**
 * Заполняет чанк травой, опционально озером и каменным хребтом, затем декором.
 * Озеро: неправильный 4-связный blob. Хребет: 4–10 камней + рамка осколков.
 * Зона спавна остаётся чистой.
 */
export function generateChunkTerrain(
  worldSeed: number,
  chunkX: number,
  chunkY: number,
  chunkSize: number,
): ChunkTerrain {
  const cellCount = chunkSize * chunkSize;
  const tiles: TileType[] = new Array(cellCount).fill(TileType.Grass);
  const trees: boolean[] = new Array(cellCount).fill(false);
  const flora: Array<FloraKind | null> = new Array(cellCount).fill(null);
  const rng = createRng(mixSeed(worldSeed, chunkX, chunkY));

  const isOrigin = chunkX === 0 && chunkY === 0;
  const spawnCx = Math.floor(chunkSize / 2);
  const spawnCy = Math.floor(chunkSize / 2);

  if (rng() < POND_CHANCE) {
    tryPlacePond(tiles, chunkSize, rng, isOrigin, spawnCx, spawnCy);
  }

  if (rng() < RIDGE_CHANCE) {
    tryPlaceStoneRidge(tiles, chunkSize, rng, isOrigin, spawnCx, spawnCy);
  }

  if (isOrigin) {
    clearSpawnPad(tiles, chunkSize, spawnCx, spawnCy);
  }

  placeDecor(tiles, trees, flora, chunkSize, rng, isOrigin, spawnCx, spawnCy);

  return { tiles, trees, flora };
}
