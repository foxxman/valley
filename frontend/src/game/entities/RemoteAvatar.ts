import Phaser from 'phaser';
import { AssetKey } from '../assets';
import type { ChunkWorld } from '../world/chunkWorld';
import { applySwimCrop, swimStateAt } from './swim';

const APPROACH_MS = 80;
const SNAP_DISTANCE_TILES = 4;

type RemoteAvatarOptions = {
  scene: Phaser.Scene;
  x: number;
  y: number;
  login: string;
  tileSize: number;
  world: ChunkWorld;
};

/** Чужой игрок: тот же спрайт, что у своего, и логин мелким шрифтом. */
export class RemoteAvatar {
  readonly sprite: Phaser.GameObjects.Image;

  private readonly label: Phaser.GameObjects.Text;
  private readonly tileSize: number;
  private readonly world: ChunkWorld;
  private readonly baseSize: number;
  private targetX: number;
  private targetY: number;

  constructor({ scene, x, y, login, tileSize, world }: RemoteAvatarOptions) {
    this.tileSize = tileSize;
    this.world = world;
    this.targetX = x;
    this.targetY = y;
    this.baseSize = tileSize * 0.9;
    this.sprite = scene.add
      .image(x, y, AssetKey.Player)
      .setDisplaySize(this.baseSize, this.baseSize)
      .setDepth(10);
    this.applySwimPose();
    this.label = scene.add
      .text(x, y - this.baseSize * 0.65, login, {
        fontFamily: 'monospace',
        fontSize: '11px',
        color: '#ffffff',
        backgroundColor: '#00000088',
        padding: { x: 3, y: 1 },
      })
      .setOrigin(0.5, 1)
      .setDepth(11);
  }

  moveTo(x: number, y: number): void {
    this.targetX = x;
    this.targetY = y;
  }

  update(deltaMs: number): void {
    const distance = Math.hypot(this.targetX - this.sprite.x, this.targetY - this.sprite.y);
    if (distance > this.tileSize * SNAP_DISTANCE_TILES) {
      this.sprite.setPosition(this.targetX, this.targetY);
    } else if (distance > 0.05) {
      const dx = this.targetX - this.sprite.x;
      const dy = this.targetY - this.sprite.y;
      const step = Math.min(1, deltaMs / APPROACH_MS);
      if (dx < -0.5) {
        this.sprite.setFlipX(true);
      } else if (dx > 0.5) {
        this.sprite.setFlipX(false);
      }
      this.sprite.x += dx * step;
      this.sprite.y += dy * step;
    }
    this.applySwimPose();
    this.label.setPosition(this.sprite.x, this.sprite.y - this.baseSize * 0.65);
  }

  /** Та же обрезка низа, что у своей фигурки: в открытой воде видна только верхняя часть. */
  private applySwimPose(): void {
    const swim = swimStateAt(this.world, this.sprite.x, this.sprite.y, this.tileSize);
    applySwimCrop(this.sprite, this.baseSize, swim.clipBottom);
  }

  destroy(): void {
    this.sprite.destroy();
    this.label.destroy();
  }
}
