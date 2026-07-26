import type { FloraKind } from '../flora';
import type { TileType } from '../tiles';

export type ChunkTerrain = {
  tiles: TileType[];
  /** Та же длина, что и tiles; true — маленькое дерево поверх травы. */
  trees: boolean[];
  /** Оверлеи: цветы/трава на суше, кувшинки на воде; null — пусто или дерево. */
  flora: Array<FloraKind | null>;
};
