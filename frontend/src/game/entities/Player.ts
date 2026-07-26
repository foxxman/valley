import Phaser from 'phaser';
import { gameEnv } from '../env';
import { AssetKey } from '../assets';
import type { ChunkWorld } from '../world/chunkWorld';

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
    this.updateBounce(deltaMs);

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

    // Mirror along the vertical axis by facing; keep last facing when idle or only vertical.
    if (vx !== 0) {
      this.sprite.setFlipX(vx < 0);
    }

    if (vx === 0 && vy === 0) {
      return;
    }

    const length = Math.hypot(vx, vy);
    vx = (vx / length) * this.speed * delta;
    vy = (vy / length) * this.speed * delta;

    const nextX = this.sprite.x + vx;
    const nextY = this.sprite.y + vy;

    if (this.canOccupy(nextX, this.sprite.y)) {
      this.sprite.x = nextX;
    }

    if (this.canOccupy(this.sprite.x, nextY)) {
      this.sprite.y = nextY;
    }
  }

  private updateBounce(deltaMs: number): void {
    this.bouncePhase += deltaMs * BOUNCE_SPEED;
    const wave = 0.5 + 0.5 * Math.sin(this.bouncePhase);
    const scaleY = SQUASH_MIN + (SQUASH_MAX - SQUASH_MIN) * wave;
    const scaleX = 2 - scaleY;

    this.sprite.setDisplaySize(this.baseSize * scaleX, this.baseSize * scaleY);
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
      // isWalkableAt generates the chunk on demand when walking into new areas.
      return this.world.isWalkableAt(tileX, tileY);
    });
  }
}
