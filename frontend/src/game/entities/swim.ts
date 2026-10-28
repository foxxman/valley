import Phaser from 'phaser';
import { AssetKey } from '../assets';
import type { ChunkWorld } from '../world/chunkWorld';
import { resolveTerrainVisual } from '../world/terrainVisual';
import { TileType } from '../world/tiles';

/** На воде скорость в 2 раза ниже. */
export const SWIM_SPEED_FACTOR = 0.5;
/** Доля видимой высоты спрайта при плавании (низ обрезается на 20%). */
export const SWIM_VISIBLE_RATIO = 0.8;

/** Визуалы воды, где фигурка обрезается снизу (не прямой берег). */
const SWIM_CLIP_VISUALS: ReadonlySet<AssetKey> = new Set([
  AssetKey.Water,
  AssetKey.ShoreCorner,
  AssetKey.ShoreCornerInner,
]);

export type SwimState = {
  inWater: boolean;
  clipBottom: boolean;
};

export function swimStateAt(
  world: ChunkWorld,
  x: number,
  y: number,
  tileSize: number,
): SwimState {
  const tileX = Math.floor(x / tileSize);
  const tileY = Math.floor(y / tileSize);
  const tile = world.getTileAt(tileX, tileY);

  if (tile !== TileType.Water) {
    return { inWater: false, clipBottom: false };
  }

  const visual = resolveTerrainVisual(world, tileX, tileY);
  return {
    inWater: true,
    clipBottom: SWIM_CLIP_VISUALS.has(visual.key),
  };
}

/**
 * Прячет низ спрайта под водой.
 * Якорь остаётся в центре полной фигурки, поэтому голова не смещается.
 */
export function applySwimCrop(
  sprite: Phaser.GameObjects.Image,
  baseSize: number,
  clipBottom: boolean,
): void {
  if (!clipBottom) {
    sprite.setCrop();
    sprite.setDisplaySize(baseSize, baseSize);
    sprite.setOrigin(0.5, 0.5);
    return;
  }

  const frame = sprite.frame;
  const visibleH = baseSize * SWIM_VISIBLE_RATIO;
  sprite.setCrop(0, 0, frame.width, frame.height * SWIM_VISIBLE_RATIO);
  sprite.setDisplaySize(baseSize, visibleH);
  sprite.setOrigin(0.5, 0.5 / SWIM_VISIBLE_RATIO);
}
