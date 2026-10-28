import Phaser from 'phaser';
import { ASSET_URLS, AssetKey } from '../assets';
import { playSessionKey, type PlaySession } from '../config';
import { inventoryOpenKey } from '../inventory/open';
import { gatherFlower } from '../map/api';
import type { MapCell } from '../map/types';
import { gameEnv } from '../env';
import { Player } from '../entities/Player';
import { RemoteAvatar } from '../entities/RemoteAvatar';
import { openGameChannel, type GameChannel, type RemotePlayer } from '../net/channel';
import { Minimap, minimapChunkRadius, type MinimapMarker } from '../ui/Minimap';
import { ChunkWorld } from '../world/chunkWorld';
import { FLORA_ASSET } from '../world/flora';
import { gatherTarget, type GatherTarget } from '../world/gatherTarget';
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
const POSITION_INTERVAL_MS = 50;

export class WorldScene extends Phaser.Scene {
  private player: Player | null = null;
  private world: ChunkWorld | null = null;
  private minimap: Minimap | null = null;
  private channel: GameChannel | null = null;
  private readonly remotes = new Map<string, RemoteAvatar>();
  private readonly chunkVisuals = new Map<string, ChunkVisuals>();
  private latestRemotes: RemotePlayer[] = [];
  private selfColor: number | null = null;
  private remotesDirty = false;
  private tileSize = 32;
  private lastSentX = Number.NaN;
  private lastSentY = Number.NaN;
  private lastSentAt = Number.NEGATIVE_INFINITY;
  private positionDue = false;
  private inventoryKey: Phaser.Input.Keyboard.Key | null = null;
  private closeInventoryKey: Phaser.Input.Keyboard.Key | null = null;
  private gatherKey: Phaser.Input.Keyboard.Key | null = null;
  private highlight: Phaser.GameObjects.Graphics | null = null;
  private gatherBusy = false;
  private alive = true;

  constructor() {
    super('WorldScene');
  }

  preload(): void {
    for (const [key, url] of Object.entries(ASSET_URLS)) {
      this.load.image(key, url);
    }
  }

  create(): void {
    const session = this.game.registry.get(playSessionKey) as PlaySession | undefined;
    if (!session) {
      throw new Error('Play session is missing');
    }
    const { tileSize, chunkSize } = gameEnv;
    this.tileSize = tileSize;
    this.world = new ChunkWorld(session.worldSeed, chunkSize);
    this.world.loadCells(session.cells);

    const originCenter = Math.floor(this.world.chunkSize / 2);
    const spawnX = originCenter * tileSize + tileSize / 2;
    const spawnY = originCenter * tileSize + tileSize / 2;
    this.lastSentX = spawnX;
    this.lastSentY = spawnY;

    this.ensureAndDrawAround(originCenter, originCenter, LOAD_RADIUS_CHUNKS);

    this.player = new Player({
      scene: this,
      x: spawnX,
      y: spawnY,
      world: this.world,
      tileSize,
    });

    this.channel = openGameChannel({
      gameId: session.gameId,
      selfId: session.playerId,
      x: spawnX,
      y: spawnY,
      onPresence: ({ selfColor, players }) => {
        this.selfColor = selfColor;
        this.latestRemotes = players;
        this.remotesDirty = true;
      },
      onMap: (cell) => {
        this.applyMapCell(cell);
      },
    });
    this.events.once(Phaser.Scenes.Events.SHUTDOWN, () => {
      this.alive = false;
      this.channel?.close();
      this.channel = null;
      for (const avatar of this.remotes.values()) {
        avatar.destroy();
      }
      this.remotes.clear();
    });

    // No setBounds — infinite world; player locked to screen center.
    this.cameras.main.startFollow(this.player.sprite, true, 1, 1);
    this.cameras.main.setRoundPixels(true);
    this.cameras.main.setBackgroundColor(0x0b1020);

    const keyboard = this.input.keyboard;
    if (!keyboard) {
      throw new Error('Keyboard input is required for the inventory');
    }
    this.inventoryKey = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.I);
    this.closeInventoryKey = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.ESC);
    this.gatherKey = keyboard.addKey(Phaser.Input.Keyboard.KeyCodes.E);
    this.highlight = this.add.graphics().setDepth(9);

    this.add
      .text(12, 12, 'WASD / стрелки — движение\nE — действие\nI — инвентарь', {
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

  update(time: number, delta: number): void {
    this.readInventoryKeys();
    this.player?.update(delta);
    this.publishPosition(time);
    this.syncRemotes();
    for (const avatar of this.remotes.values()) {
      avatar.update(delta);
    }
    if (!this.player || !this.world) {
      return;
    }

    const tileX = Math.floor(this.player.sprite.x / this.tileSize);
    const tileY = Math.floor(this.player.sprite.y / this.tileSize);
    this.ensureAndDrawAround(tileX, tileY, LOAD_RADIUS_CHUNKS);
    this.syncTarget();
    this.cullFarChunks(tileX, tileY, DRAW_KEEP_RADIUS_CHUNKS);
    this.minimap?.update(this.world, tileX, tileY, this.selfColor, this.remoteMarkers());
    const dataKeep = Math.max(
      DRAW_KEEP_RADIUS_CHUNKS,
      minimapChunkRadius(this.world.chunkSize),
    );
    this.world.releaseFarChunks(tileX, tileY, dataKeep + DATA_KEEP_PADDING_CHUNKS);
  }

  private syncTarget(): void {
    const target = this.currentTarget();
    this.paintHighlight(target);
    this.readGatherKey(target);
  }

  private currentTarget(): GatherTarget | null {
    if (!this.player || !this.world) {
      return null;
    }
    return gatherTarget(
      (x, y) => this.world?.floraAt(x, y) ?? null,
      Math.floor(this.player.sprite.x / this.tileSize),
      Math.floor(this.player.sprite.y / this.tileSize),
      this.player.facing,
    );
  }

  private paintHighlight(target: GatherTarget | null): void {
    this.highlight?.clear();
    if (!target || !this.highlight) {
      return;
    }
    this.highlight.lineStyle(2, 0xffe08a, 1);
    this.highlight.strokeRect(target.x * this.tileSize, target.y * this.tileSize, this.tileSize, this.tileSize);
  }

  private readGatherKey(target: GatherTarget | null): void {
    const session = this.game.registry.get(playSessionKey) as PlaySession | undefined;
    if (!session || !this.gatherKey || isTextEntry(document.activeElement)) {
      return;
    }
    if (!Phaser.Input.Keyboard.JustDown(this.gatherKey)) {
      return;
    }
    if (this.game.registry.get(inventoryOpenKey) === true || this.gatherBusy || !target) {
      return;
    }
    this.gatherBusy = true;
    void gatherFlower(session.gameId, { x: target.x, y: target.y, kind: target.kind })
      .then((result) => {
        if (!this.alive || !result.ok) {
          return;
        }
        const current = this.game.registry.get(playSessionKey) as PlaySession | undefined;
        current?.onInventory({
          capacity: result.data.capacity,
          slots: result.data.slots ?? [],
        });
        this.applyMapCell(result.data.cell);
      })
      .catch(() => undefined)
      .finally(() => {
        this.gatherBusy = false;
      });
  }

  private applyMapCell(cell: MapCell): void {
    if (!this.alive || !this.world) {
      return;
    }
    this.world.applyCell(cell);
    const { cx, cy } = this.world.worldToChunk(cell.x, cell.y);
    this.redrawChunk(cx, cy);
  }

  private redrawChunk(cx: number, cy: number): void {
    const key = `${cx},${cy}`;
    const visuals = this.chunkVisuals.get(key);
    if (!visuals) {
      return;
    }
    visuals.forEach((visual) => visual.destroy());
    this.chunkVisuals.delete(key);
    this.drawChunkIfNeeded(cx, cy);
  }

  private readInventoryKeys(): void {
    const session = this.game.registry.get(playSessionKey) as PlaySession | undefined;
    if (!session || isTextEntry(document.activeElement)) {
      return;
    }
    if (this.inventoryKey && Phaser.Input.Keyboard.JustDown(this.inventoryKey)) {
      session.onToggleInventory();
    }
    if (
      this.closeInventoryKey &&
      Phaser.Input.Keyboard.JustDown(this.closeInventoryKey) &&
      this.game.registry.get(inventoryOpenKey) === true
    ) {
      session.onCloseInventory();
    }
  }

  private publishPosition(now: number): void {
    if (!this.player || !this.channel) {
      return;
    }
    const x = this.player.sprite.x;
    const y = this.player.sprite.y;
    if (Math.hypot(x - this.lastSentX, y - this.lastSentY) >= 0.5) {
      this.positionDue = true;
    }
    if (!this.positionDue || now - this.lastSentAt < POSITION_INTERVAL_MS) {
      return;
    }
    this.channel.move(x, y);
    this.lastSentX = x;
    this.lastSentY = y;
    this.lastSentAt = now;
    this.positionDue = false;
  }

  private syncRemotes(): void {
    const world = this.world;
    if (!this.remotesDirty || !world) {
      return;
    }
    this.remotesDirty = false;
    const alive = new Set(this.latestRemotes.map((player) => player.playerId));
    for (const [id, avatar] of this.remotes) {
      if (alive.has(id)) {
        continue;
      }
      avatar.destroy();
      this.remotes.delete(id);
    }
    for (const player of this.latestRemotes) {
      const existing = this.remotes.get(player.playerId);
      if (existing) {
        existing.moveTo(player.x, player.y);
        continue;
      }
      this.remotes.set(
        player.playerId,
        new RemoteAvatar({
          scene: this,
          x: player.x,
          y: player.y,
          login: player.login,
          tileSize: this.tileSize,
          world,
        }),
      );
    }
  }

  private remoteMarkers(): MinimapMarker[] {
    return this.latestRemotes.map((player) => ({
      tileX: Math.floor(player.x / this.tileSize),
      tileY: Math.floor(player.y / this.tileSize),
      color: player.color,
    }));
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

const textEntryTags: Record<string, true> = {
  INPUT: true,
  TEXTAREA: true,
  SELECT: true,
};

function isTextEntry(target: EventTarget | null): boolean {
  return target instanceof HTMLElement && (textEntryTags[target.tagName] === true || target.isContentEditable);
}
