import Phaser from 'phaser';
import { gameEnv } from '../env';
import { AssetKey } from '../assets';
import type { ChunkWorld } from '../world/chunkWorld';
import { resolveTerrainVisual } from '../world/terrainVisual';
import { TileType } from '../world/tiles';

type PlayerOptions = {
  scene: Phaser.Scene;
  x: number;
  y: number;
  world: ChunkWorld;
  tileSize: number;
};

const BOUNCE_SPEED = 0.01;
const SQUASH_MIN = 0.86;
const SQUASH_MAX = 1;
/** На воде скорость в 2 раза ниже. */
const SWIM_SPEED_FACTOR = 0.5;
/** Доля видимой высоты спрайта при плавании (низ обрезается на 20%). */
const SWIM_VISIBLE_RATIO = 0.8;

/** Визуалы воды, где фигурка обрезается снизу (не прямой берег). */
const SWIM_CLIP_VISUALS: ReadonlySet<AssetKey> = new Set([
  AssetKey.Water,
  AssetKey.ShoreCorner,
  AssetKey.ShoreCornerInner,
]);

type SwimState = {
  inWater: boolean;
  clipBottom: boolean;
};

export class Player {
  readonly sprite: Phaser.GameObjects.Image;

  private readonly cursors: Phaser.Types.Input.Keyboard.CursorKeys;
  private readonly wasd: {
    up: Phaser.Input.Keyboard.Key;
    down: Phaser.Input.Keyboard.Key;
    left: Phaser.Input.Keyboard.Key;
    right: Phaser.Input.Keyboard.Key;
  };

  private readonly world: ChunkWorld;
  private readonly tileSize: number;
  private readonly speed: number;
  private readonly halfSize: number;
  private readonly baseSize: number;
  private bouncePhase = 0;

  constructor({ scene, x, y, world, tileSize }: PlayerOptions) {
    this.world = world;
    this.tileSize = tileSize;
    this.speed = gameEnv.playerSpeed;
    this.halfSize = tileSize * 0.3;
    this.baseSize = tileSize * 0.9;

    this.sprite = scene.add
      .image(x, y, AssetKey.Player)
      .setDisplaySize(this.baseSize, this.baseSize)
      .setDepth(10);

    if (!scene.input.keyboard) {
      throw new Error('Keyboard input is required for player movement');
    }

    this.cursors = scene.input.keyboard.createCursorKeys();
    this.wasd = {
      up: scene.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.W),
      down: scene.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.S),
      left: scene.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.A),
      right: scene.input.keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.D),
    };
  }

  update(deltaMs: number): void {
    const swim = this.getSwimState();
    this.updateBounce(deltaMs, swim);

    const delta = deltaMs / 1000;
    let vx = 0;
    let vy = 0;

    if (this.cursors.left?.isDown || this.wasd.left.isDown) {
      vx -= 1;
    }
    if (this.cursors.right?.isDown || this.wasd.right.isDown) {
      vx += 1;
    }
    if (this.cursors.up?.isDown || this.wasd.up.isDown) {
      vy -= 1;
    }
    if (this.cursors.down?.isDown || this.wasd.down.isDown) {
      vy += 1;
    }

    // Зеркало по вертикали при движении вбок; последнее направление при простое.
    if (vx !== 0) {
      this.sprite.setFlipX(vx < 0);
    }

    if (vx === 0 && vy === 0) {
      return;
    }

    const moveSpeed = this.speed * (swim.inWater ? SWIM_SPEED_FACTOR : 1);
    const length = Math.hypot(vx, vy);
    vx = (vx / length) * moveSpeed * delta;
    vy = (vy / length) * moveSpeed * delta;

    const nextX = this.sprite.x + vx;
    const nextY = this.sprite.y + vy;

    if (this.canOccupy(nextX, this.sprite.y)) {
      this.sprite.x = nextX;
    }

    if (this.canOccupy(this.sprite.x, nextY)) {
      this.sprite.y = nextY;
    }
  }

  /**
   * На суше — squash/stretch.
   * В воде — без сжатия; при необходимости только обрезка низа.
   */
  private updateBounce(deltaMs: number, swim: SwimState): void {
    const frame = this.sprite.frame;

    if (swim.inWater) {
      const fullW = this.baseSize;
      const fullH = this.baseSize;

      if (swim.clipBottom) {
        const visibleH = fullH * SWIM_VISIBLE_RATIO;
        this.sprite.setCrop(0, 0, frame.width, frame.height * SWIM_VISIBLE_RATIO);
        this.sprite.setDisplaySize(fullW, visibleH);
        this.sprite.setOrigin(0.5, 0.5 / SWIM_VISIBLE_RATIO);
      } else {
        this.sprite.setCrop();
        this.sprite.setDisplaySize(fullW, fullH);
        this.sprite.setOrigin(0.5, 0.5);
      }
      return;
    }

    this.bouncePhase += deltaMs * BOUNCE_SPEED;
    const wave = 0.5 + 0.5 * Math.sin(this.bouncePhase);
    const scaleY = SQUASH_MIN + (SQUASH_MAX - SQUASH_MIN) * wave;
    const scaleX = 2 - scaleY;

    this.sprite.setCrop();
    this.sprite.setDisplaySize(this.baseSize * scaleX, this.baseSize * scaleY);
    this.sprite.setOrigin(0.5, 0.5);
  }

  private getSwimState(): SwimState {
    const tileX = Math.floor(this.sprite.x / this.tileSize);
    const tileY = Math.floor(this.sprite.y / this.tileSize);
    const tile = this.world.getTileAt(tileX, tileY);

    if (tile !== TileType.Water) {
      return { inWater: false, clipBottom: false };
    }

    const visual = resolveTerrainVisual(this.world, tileX, tileY);
    return {
      inWater: true,
      clipBottom: SWIM_CLIP_VISUALS.has(visual.key),
    };
  }

  private canOccupy(x: number, y: number): boolean {
    const points = [
      { x: x - this.halfSize, y: y - this.halfSize },
      { x: x + this.halfSize, y: y - this.halfSize },
      { x: x - this.halfSize, y: y + this.halfSize },
      { x: x + this.halfSize, y: y + this.halfSize },
    ];

    return points.every((point) => {
      const tileX = Math.floor(point.x / this.tileSize);
      const tileY = Math.floor(point.y / this.tileSize);
      return this.world.isWalkableAt(tileX, tileY);
    });
  }
}
