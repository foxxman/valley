import Phaser from 'phaser';
import type { ChunkWorld } from '../world/chunkWorld';
import { TileType } from '../world/tiles';

export type MinimapMarker = {
  tileX: number;
  tileY: number;
  color: number;
};

const SIZE_PX = 150;
const PADDING_PX = 12;
/** Odd so the local player sits on the center cell. ~3 chunks across. */
const TILES_ACROSS = 49;
const HALF_TILES = Math.floor(TILES_ACROSS / 2);

/** How many chunks around the center the minimap must keep generated. */
export function minimapChunkRadius(chunkSize: number): number {
  return Math.ceil(HALF_TILES / chunkSize);
}

const COLOR = {
  Grass: 0x8dc435,
  Stone: 0x78909c,
  Water: 0x63c5cf,
  Panel: 0x0b1020,
  FrameOuter: 0x1a2332,
  FrameAccent: 0x63c5cf,
  FrameInner: 0xffffff,
  LocalPlayer: 0xffeb3b,
} as const;

function terrainColor(world: ChunkWorld, tileX: number, tileY: number): number {
  const tile = world.getTileAt(tileX, tileY);
  const byTile: Record<TileType, number> = {
    [TileType.Grass]: COLOR.Grass,
    [TileType.StoneShards]: COLOR.Grass,
    [TileType.Stone]: COLOR.Stone,
    [TileType.Water]: COLOR.Water,
  };
  return byTile[tile];
}

type MinimapOptions = {
  scene: Phaser.Scene;
  depth?: number;
};

/**
 * Fixed HUD minimap (top-right). Local player is always drawn at the center;
 * other markers are offset relative to them.
 */
export class Minimap {
  private readonly scene: Phaser.Scene;
  private readonly graphics: Phaser.GameObjects.Graphics;
  private readonly cellPx: number;
  private lastCenterTileX = Number.NaN;
  private lastCenterTileY = Number.NaN;
  private lastMarkers: ReadonlyArray<MinimapMarker> = [];

  constructor({ scene, depth = 200 }: MinimapOptions) {
    this.scene = scene;
    this.cellPx = SIZE_PX / TILES_ACROSS;
    this.graphics = scene.add.graphics().setScrollFactor(0).setDepth(depth);
  }

  /**
   * @param markers — optional remote players etc.; local player is always centered
   *   and drawn as LocalPlayer color (do not include local in markers).
   */
  update(
    world: ChunkWorld,
    localTileX: number,
    localTileY: number,
    markers: ReadonlyArray<MinimapMarker> = [],
  ): void {
    this.layout();

    const terrainMoved =
      localTileX !== this.lastCenterTileX || localTileY !== this.lastCenterTileY;
    if (!terrainMoved && this.sameMarkers(markers)) {
      return;
    }

    if (terrainMoved) {
      this.lastCenterTileX = localTileX;
      this.lastCenterTileY = localTileY;
      world.ensureChunksAround(
        localTileX,
        localTileY,
        minimapChunkRadius(world.chunkSize),
      );
    }
    this.lastMarkers = markers.map((marker) => ({ ...marker }));

    this.graphics.clear();
    this.drawPanel();
    this.drawTerrain(world, localTileX, localTileY);
    this.drawMarkers(localTileX, localTileY, markers);
    this.drawLocalPlayer();
    this.drawFrame();
  }

  destroy(): void {
    this.graphics.destroy();
  }

  private sameMarkers(markers: ReadonlyArray<MinimapMarker>): boolean {
    if (markers.length !== this.lastMarkers.length) {
      return false;
    }
    return markers.every((marker, index) => {
      const previous = this.lastMarkers[index];
      return (
        previous !== undefined &&
        marker.tileX === previous.tileX &&
        marker.tileY === previous.tileY &&
        marker.color === previous.color
      );
    });
  }

  private layout(): void {
    const cam = this.scene.cameras.main;
    this.graphics.setPosition(cam.width - SIZE_PX - PADDING_PX, PADDING_PX);
  }

  private drawPanel(): void {
    this.graphics.fillStyle(COLOR.Panel, 0.9);
    this.graphics.fillRect(0, 0, SIZE_PX, SIZE_PX);
  }

  private drawFrame(): void {
    // Soft drop shadow outside the map.
    this.graphics.lineStyle(5, 0x000000, 0.28);
    this.graphics.strokeRect(-2.5, -2.5, SIZE_PX + 5, SIZE_PX + 5);

    // Thick dark body of the frame.
    this.graphics.lineStyle(4, COLOR.FrameOuter, 1);
    this.graphics.strokeRect(0, 0, SIZE_PX, SIZE_PX);

    // Accent rim (matches water palette).
    this.graphics.lineStyle(2, COLOR.FrameAccent, 0.75);
    this.graphics.strokeRect(2, 2, SIZE_PX - 4, SIZE_PX - 4);

    // Inner highlight for depth.
    this.graphics.lineStyle(1, COLOR.FrameInner, 0.22);
    this.graphics.strokeRect(4.5, 4.5, SIZE_PX - 9, SIZE_PX - 9);
  }

  private drawTerrain(world: ChunkWorld, centerX: number, centerY: number): void {
    for (let dy = -HALF_TILES; dy <= HALF_TILES; dy += 1) {
      for (let dx = -HALF_TILES; dx <= HALF_TILES; dx += 1) {
        const color = terrainColor(world, centerX + dx, centerY + dy);
        const px = (dx + HALF_TILES) * this.cellPx;
        const py = (dy + HALF_TILES) * this.cellPx;
        this.graphics.fillStyle(color, 1);
        this.graphics.fillRect(px, py, this.cellPx + 0.5, this.cellPx + 0.5);
      }
    }
  }

  private drawMarkers(
    centerX: number,
    centerY: number,
    markers: ReadonlyArray<MinimapMarker>,
  ): void {
    for (const marker of markers) {
      const dx = marker.tileX - centerX;
      const dy = marker.tileY - centerY;
      if (Math.abs(dx) > HALF_TILES || Math.abs(dy) > HALF_TILES) {
        continue;
      }
      this.drawDot(dx, dy, marker.color);
    }
  }

  private drawLocalPlayer(): void {
    this.drawDot(0, 0, COLOR.LocalPlayer);
  }

  private drawDot(dx: number, dy: number, color: number): void {
    const cx = (dx + HALF_TILES + 0.5) * this.cellPx;
    const cy = (dy + HALF_TILES + 0.5) * this.cellPx;
    const radius = Math.max(2, this.cellPx * 0.9);
    this.graphics.fillStyle(color, 1);
    this.graphics.fillCircle(cx, cy, radius);
    this.graphics.lineStyle(1, 0x000000, 0.45);
    this.graphics.strokeCircle(cx, cy, radius);
  }
}
