import { AssetKey } from '../assets';
import type { ChunkWorld } from './chunkWorld';
import { isLandTile, TileType } from './tiles';

/** Биты соседей по сторонам света. */
const Bit = {
  N: 1,
  E: 2,
  S: 4,
  W: 8,
} as const;

/** Биты диагональных соседей. */
const Diag = {
  NE: 1,
  SE: 2,
  SW: 4,
  NW: 8,
} as const;

export type TerrainVisual = {
  key: AssetKey;
  angle: number;
};

type NeighborMask = {
  cardinal: number;
  diagonal: number;
};

/**
 * Соседи-суша вокруг клетки воды.
 * Берег всегда на клетке Water, граничащей с сушей.
 */
function landNeighborsAt(world: ChunkWorld, x: number, y: number): NeighborMask {
  let cardinal = 0;
  let diagonal = 0;

  if (isLandTile(world.getTileAt(x, y - 1))) cardinal |= Bit.N;
  if (isLandTile(world.getTileAt(x + 1, y))) cardinal |= Bit.E;
  if (isLandTile(world.getTileAt(x, y + 1))) cardinal |= Bit.S;
  if (isLandTile(world.getTileAt(x - 1, y))) cardinal |= Bit.W;

  if (isLandTile(world.getTileAt(x + 1, y - 1))) diagonal |= Diag.NE;
  if (isLandTile(world.getTileAt(x + 1, y + 1))) diagonal |= Diag.SE;
  if (isLandTile(world.getTileAt(x - 1, y + 1))) diagonal |= Diag.SW;
  if (isLandTile(world.getTileAt(x - 1, y - 1))) diagonal |= Diag.NW;

  return { cardinal, diagonal };
}

/**
 * Визуал берега для клетки Water.
 *
 * Правила (приоритет сверху вниз):
 * 1. Две соседние стороны суши → внешний угол (суша огибает воду).
 * 2. Одна сторона суши → прямой берег.
 * 3. Иная ортогональная суша (канал/тупик) → запасной берег
 *    (никогда не water.png рядом с сушей).
 * 4. Нет ортогональной суши, но есть диагональная → внутренний уголок.
 * 5. Иначе → чистая вода (только внутри озера).
 *
 * Базовые ориентации (angle 0, по часовой в Phaser):
 * - straight: трава СВЕРХУ
 * - corner (внешний): трава ВЕРХ-ЛЕВО
 * - corner_inner: уголок травы ВЕРХ-ЛЕВО
 */
function waterShoreVisual(neighbors: NeighborMask): TerrainVisual {
  const { N, E, S, W } = Bit;
  const { NE, SE, SW, NW } = Diag;
  const { cardinal, diagonal } = neighbors;

  const outerCornerByMask: Record<number, number> = {
    [N | W]: 0,
    [N | E]: 90,
    [S | E]: 180,
    [S | W]: 270,
  };

  if (cardinal in outerCornerByMask) {
    return { key: AssetKey.ShoreCorner, angle: outerCornerByMask[cardinal]! };
  }

  const straightByMask: Record<number, number> = {
    [N]: 0,
    [E]: 90,
    [S]: 180,
    [W]: 270,
  };

  if (cardinal in straightByMask) {
    return { key: AssetKey.ShoreStraight, angle: straightByMask[cardinal]! };
  }

  // Запасной берег: при любой ортогональной суше нельзя показывать water.png.
  if (cardinal !== 0) {
    const cornerPairs: Array<readonly [number, number]> = [
      [N | W, 0],
      [N | E, 90],
      [S | E, 180],
      [S | W, 270],
    ];
    for (const [bits, angle] of cornerPairs) {
      if ((cardinal & bits) === bits) {
        return { key: AssetKey.ShoreCorner, angle };
      }
    }

    const straightBits: Array<readonly [number, number]> = [
      [N, 0],
      [E, 90],
      [S, 180],
      [W, 270],
    ];
    for (const [bit, angle] of straightBits) {
      if ((cardinal & bit) !== 0) {
        return { key: AssetKey.ShoreStraight, angle };
      }
    }
  }

  // Внутренний уголок только без ортогональной суши.
  const innerCornerByDiag: Record<number, number> = {
    [NW]: 0,
    [NE]: 90,
    [SE]: 180,
    [SW]: 270,
  };

  const diagOrder = [NW, NE, SE, SW] as const;
  for (const bit of diagOrder) {
    if ((diagonal & bit) !== 0) {
      return {
        key: AssetKey.ShoreCornerInner,
        angle: innerCornerByDiag[bit]!,
      };
    }
  }

  return { key: AssetKey.Water, angle: 0 };
}

const SOLID_VISUAL: Partial<Record<TileType, TerrainVisual>> = {
  [TileType.Grass]: { key: AssetKey.Grass, angle: 0 },
  [TileType.Stone]: { key: AssetKey.Stone, angle: 0 },
  [TileType.StoneShards]: { key: AssetKey.StoneShards, angle: 0 },
};

export function resolveTerrainVisual(world: ChunkWorld, x: number, y: number): TerrainVisual {
  const tile = world.getTileAt(x, y);
  const solid = SOLID_VISUAL[tile];
  if (solid) {
    return solid;
  }
  return waterShoreVisual(landNeighborsAt(world, x, y));
}
