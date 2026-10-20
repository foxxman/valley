import Phaser from 'phaser';
import { ASSET_URLS, AssetKey } from '../assets';
import { gameEnv } from '../env';
import { Player } from '../entities/Player';
import { Minimap, minimapChunkRadius } from '../ui/Minimap';
import { ChunkWorld } from '../world/chunkWorld';
import { FLORA_ASSET } from '../world/flora';
import { resolveTerrainVisual } from '../world/terrainVisual';

type ChunkVisuals = Phaser.GameObjects.GameObject[];

/** How many chunks around the player to keep generated and drawn. */
const LOAD_RADIUS_CHUNKS = 1;
/** Keep baked chunk textures within this Chebyshev radius. */
const DRAW_KEEP_RADIUS_CHUNKS = 2;
/**
 * Extra ring past the minimap and the draw window.
 * Shore autotile reads one tile outside a chunk; without this ring those
 * neighbors would be generated and dropped on every step.
 */
const DATA_KEEP_PADDING_CHUNKS = 1;

export class WorldScene extends Phaser.Scene {
  private player: Player | null = null;
  private world: ChunkWorld | null = null;
  private minimap: Minimap | null = null;
  private readonly chunkVisuals = new Map<string, ChunkVisuals>();
  private tileSize = 32;

  constructor() {
    super('WorldScene');
  }

  preload(): void {
    for (const [key, url] of Object.entries(ASSET_URLS)) {
      this.load.image(key, url);
    }
  }

  create(): void {
    const { tileSize, worldSeed, chunkSize } = gameEnv;
    this.tileSize = tileSize;
    this.world = new ChunkWorld(worldSeed, chunkSize);

    const originCenter = Math.floor(this.world.chunkSize / 2);
    const spawnX = originCenter * tileSize + tileSize / 2;
    const spawnY = originCenter * tileSize + tileSize / 2;

    this.ensureAndDrawAround(originCenter, originCenter, LOAD_RADIUS_CHUNKS);

    this.player = new Player({
      scene: this,
      x: spawnX,
      y: spawnY,
      world: this.world,
      tileSize,
    });

    // No setBounds — infinite world; player locked to screen center.
    this.cameras.main.startFollow(this.player.sprite, true, 1, 1);
    this.cameras.main.setRoundPixels(true);
    this.cameras.main.setBackgroundColor(0x0b1020);

    this.add
      .text(12, 12, 'WASD / стрелки — движение', {
        fontFamily: 'monospace',
        fontSize: '14px',
        color: '#ffffff',
        backgroundColor: '#00000088',
        padding: { x: 8, y: 6 },
      })
      .setScrollFactor(0)
      .setDepth(100);

    this.minimap = new Minimap({ scene: this });
  }

  update(_time: number, delta: number): void {
    this.player?.update(delta);
    if (!this.player || !this.world) {
      return;
    }

    const tileX = Math.floor(this.player.sprite.x / this.tileSize);
    const tileY = Math.floor(this.player.sprite.y / this.tileSize);
    this.ensureAndDrawAround(tileX, tileY, LOAD_RADIUS_CHUNKS);
    this.cullFarChunks(tileX, tileY, DRAW_KEEP_RADIUS_CHUNKS);
    this.minimap?.update(this.world, tileX, tileY);
    const dataKeep = Math.max(
      DRAW_KEEP_RADIUS_CHUNKS,
      minimapChunkRadius(this.world.chunkSize),
    );
    this.world.releaseFarChunks(tileX, tileY, dataKeep + DATA_KEEP_PADDING_CHUNKS);
  }

  private ensureAndDrawAround(tileX: number, tileY: number, radiusChunks: number): void {
    if (!this.world) {
      return;
    }

    const touched = this.world.ensureChunksAround(tileX, tileY, radiusChunks);
    for (const { cx, cy } of touched) {
      this.drawChunkIfNeeded(cx, cy);
    }
  }

  private drawChunkIfNeeded(cx: number, cy: number): void {
    if (!this.world) {
      return;
    }

    const key = `${cx},${cy}`;
    if (this.chunkVisuals.has(key)) {
      return;
    }

    const { chunk } = this.world.ensureChunk(cx, cy);
    const size = this.world.chunkSize;
    const originTileX = cx * size;
    const originTileY = cy * size;
    const pixelSize = size * this.tileSize;
    const chunkTexture = this.add
      .renderTexture(originTileX * this.tileSize, originTileY * this.tileSize, pixelSize, pixelSize)
      .setOrigin(0, 0)
      .setDepth(0);
    chunkTexture.texture.setFilter(Phaser.Textures.FilterMode.NEAREST);

    const stamp = this.make.image({ key: AssetKey.Grass, add: false }).setOrigin(0.5, 0.5);
    const drawStamp = (textureKey: string, lx: number, ly: number, angle: number) => {
      stamp.setTexture(textureKey);
      stamp.setDisplaySize(this.tileSize, this.tileSize);
      stamp.setAngle(angle);
      chunkTexture.draw(
        stamp,
        lx * this.tileSize + this.tileSize / 2,
        ly * this.tileSize + this.tileSize / 2,
      );
    };

    for (let ly = 0; ly < size; ly += 1) {
      for (let lx = 0; lx < size; lx += 1) {
        const tx = originTileX + lx;
        const ty = originTileY + ly;
        const visual = resolveTerrainVisual(this.world, tx, ty);
        drawStamp(visual.key, lx, ly, visual.angle);

        if (chunk.trees[ly * size + lx]) {
          drawStamp(AssetKey.TreeSmall, lx, ly, 0);
        } else {
          const floraKind = chunk.flora[ly * size + lx];
          if (floraKind) {
            drawStamp(FLORA_ASSET[floraKind], lx, ly, 0);
          }
        }
      }
    }

    stamp.destroy();
    this.chunkVisuals.set(key, [chunkTexture]);
  }

  private cullFarChunks(tileX: number, tileY: number, keepRadiusChunks: number): void {
    if (!this.world) {
      return;
    }

    const { cx, cy } = this.world.worldToChunk(tileX, tileY);

    for (const key of [...this.chunkVisuals.keys()]) {
      const [chunkX, chunkY] = key.split(',').map(Number) as [number, number];
      if (
        Math.abs(chunkX - cx) > keepRadiusChunks ||
        Math.abs(chunkY - cy) > keepRadiusChunks
      ) {
        const visuals = this.chunkVisuals.get(key);
        visuals?.forEach((visual) => visual.destroy());
        this.chunkVisuals.delete(key);
      }
    }
  }
}
