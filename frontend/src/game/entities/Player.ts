import Phaser from 'phaser';
import { gameEnv } from '../env';
import { AssetKey } from '../assets';
import { inventoryOpenKey } from '../inventory/open';
import type { ChunkWorld } from '../world/chunkWorld';
import { applySwimCrop, SWIM_SPEED_FACTOR, swimStateAt, type SwimState } from './swim';

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
  private facingAxis: { x: -1 | 0 | 1; y: -1 | 0 | 1 } = { x: 0, y: 1 };

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

  /** Ровно одна ось ненулевая. До первого нажатия взгляд направлен вниз. */
  get facing(): { x: -1 | 0 | 1; y: -1 | 0 | 1 } {
    return this.facingAxis;
  }

  update(deltaMs: number): void {
    this.trackFacing();
    const swim = swimStateAt(this.world, this.sprite.x, this.sprite.y, this.tileSize);
    this.updateBounce(deltaMs, swim);

    if (this.sprite.scene.registry.get(inventoryOpenKey) === true) {
      return;
    }

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
    if (swim.inWater) {
      applySwimCrop(this.sprite, this.baseSize, swim.clipBottom);
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

  private trackFacing(): void {
    const held: Array<{ key: Phaser.Input.Keyboard.Key | undefined; x: -1 | 0 | 1; y: -1 | 0 | 1 }> = [
      { key: this.wasd.left, x: -1, y: 0 },
      { key: this.cursors.left, x: -1, y: 0 },
      { key: this.wasd.right, x: 1, y: 0 },
      { key: this.cursors.right, x: 1, y: 0 },
      { key: this.wasd.up, x: 0, y: -1 },
      { key: this.cursors.up, x: 0, y: -1 },
      { key: this.wasd.down, x: 0, y: 1 },
      { key: this.cursors.down, x: 0, y: 1 },
    ];
    let latest = -1;
    let next = this.facingAxis;
    for (const direction of held) {
      const key = direction.key;
      if (!key?.isDown || key.timeDown < latest) {
        continue;
      }
      latest = key.timeDown;
      next = { x: direction.x, y: direction.y };
    }
    if (latest >= 0) {
      this.facingAxis = next;
    }
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
