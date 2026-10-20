import Phaser from 'phaser';
import { ASSET_URLS, AssetKey } from '../assets';
import { gameEnv } from '../env';
import { Player } from '../entities/Player';
import { Minimap } from '../ui/Minimap';
import { ChunkWorld } from '../world/chunkWorld';
import { FLORA_ASSET } from '../world/flora';
import { resolveTerrainVisual } from '../world/terrainVisual';

type ChunkImages = Phaser.GameObjects.Image[];

/** How many chunks around the player to keep generated/drawn. */
const LOAD_RADIUS_CHUNKS = 1;
/** Keep drawn images within this Chebyshev radius; data stays in ChunkWorld. */
const DRAW_KEEP_RADIUS_CHUNKS = 2;

export class WorldScene extends Phaser.Scene {
  private player: Player | null = null;
  private world: ChunkWorld | null = null;
  private minimap: Minimap | null = null;
  private readonly chunkImages = new Map<string, ChunkImages>();
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
    if (this.chunkImages.has(key)) {
      return;
    }

    const { chunk } = this.world.ensureChunk(cx, cy);
    const images: ChunkImages = [];
    const size = this.world.chunkSize;
    const originTileX = cx * size;
    const originTileY = cy * size;

    for (let ly = 0; ly < size; ly += 1) {
      for (let lx = 0; lx < size; lx += 1) {
        const tx = originTileX + lx;
        const ty = originTileY + ly;
        const visual = resolveTerrainVisual(this.world, tx, ty);
        const image = this.add
          .image(
            tx * this.tileSize + this.tileSize / 2,
            ty * this.tileSize + this.tileSize / 2,
            visual.key,
          )
          .setDisplaySize(this.tileSize, this.tileSize)
          .setAngle(visual.angle)
          .setDepth(0);
        images.push(image);

        if (chunk.trees[ly * size + lx]) {
          const tree = this.add
            .image(
              tx * this.tileSize + this.tileSize / 2,
              ty * this.tileSize + this.tileSize / 2,
              AssetKey.TreeSmall,
            )
            .setDisplaySize(this.tileSize, this.tileSize)
            .setDepth(2);
          images.push(tree);
        } else {
          const floraKind = chunk.flora[ly * size + lx];
          if (floraKind) {
            const flora = this.add
              .image(
                tx * this.tileSize + this.tileSize / 2,
                ty * this.tileSize + this.tileSize / 2,
                FLORA_ASSET[floraKind],
              )
              .setDisplaySize(this.tileSize, this.tileSize)
              .setDepth(1);
            images.push(flora);
          }
        }
      }
    }

    this.chunkImages.set(key, images);
  }

  private cullFarChunks(tileX: number, tileY: number, keepRadiusChunks: number): void {
    if (!this.world) {
      return;
    }

    const { cx, cy } = this.world.worldToChunk(tileX, tileY);

    for (const key of [...this.chunkImages.keys()]) {
      const [chunkX, chunkY] = key.split(',').map(Number) as [number, number];
      if (
        Math.abs(chunkX - cx) > keepRadiusChunks ||
        Math.abs(chunkY - cy) > keepRadiusChunks
      ) {
        const images = this.chunkImages.get(key);
        images?.forEach((image) => image.destroy());
        this.chunkImages.delete(key);
      }
    }
  }
}
