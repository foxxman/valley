import type { MapCell, MapLayers } from '../map/types';
import { generateChunkTerrain } from './generate-chunk';
import { FloraKind } from './flora';
import { NodeKind, nodeDurability } from './nodes';
import { TileType, TILE_WALKABLE } from './tiles';

export const DEFAULT_CHUNK_SIZE = 16;

export type ChunkData = {
  cx: number;
  cy: number;
  tiles: TileType[];
  /** Overlay layer; true = small tree on that cell (only on grass). */
  trees: boolean[];
  /** Walkable flower/grass overlays. */
  flora: Array<FloraKind | null>;
  /** Stored, not drawn. Shores stay derived from terrain. */
  features: Array<string | null>;
};

/** Serializable world state — ready for future save/load. */
export type WorldSnapshot = {
  seed: number;
  chunkSize: number;
  chunks: ReadonlyArray<{
    cx: number;
    cy: number;
    tiles: TileType[];
    trees: boolean[];
    flora: Array<FloraKind | null>;
  }>;
};

const tileByName: Record<string, TileType> = {
  [TileType.Grass]: TileType.Grass,
  [TileType.Water]: TileType.Water,
  [TileType.Stone]: TileType.Stone,
  [TileType.StoneShards]: TileType.StoneShards,
};

const floraByName: Record<string, FloraKind> = {
  [FloraKind.Flower0]: FloraKind.Flower0,
  [FloraKind.Flower1]: FloraKind.Flower1,
  [FloraKind.Flower2]: FloraKind.Flower2,
  [FloraKind.Flower3]: FloraKind.Flower3,
  [FloraKind.Lily]: FloraKind.Lily,
  [FloraKind.Grass1]: FloraKind.Grass1,
};

function chunkKey(cx: number, cy: number): string {
  return `${cx},${cy}`;
}

function floorDiv(value: number, size: number): number {
  return Math.floor(value / size);
}

function mod(value: number, size: number): number {
  return ((value % size) + size) % size;
}

function remainingDurability(value: string | null | undefined): number | null {
  if (typeof value !== 'string' || !/^[0-9]+$/.test(value)) {
    return null;
  }
  const parsed = Number(value);
  if (!Number.isInteger(parsed) || parsed < 1) {
    return null;
  }
  return parsed;
}

function blankFeatures(size: number): Array<string | null> {
  return Array.from({ length: size * size }, () => null);
}

/**
 * Infinite chunked world. Chunks generate on first access.
 * `releaseFarChunks` drops generated chunks outside the player's neighborhood;
 * the same seed rebuilds them. Journal overrides stay and are painted after
 * generation. Chunks loaded via `fromSnapshot` stay pinned, because a save
 * may differ from regeneration.
 */
export class ChunkWorld {
  readonly seed: number;
  readonly chunkSize: number;

  private readonly chunks = new Map<string, ChunkData>();
  /** Snapshot chunks. Not released, so a loaded save is not overwritten by generation. */
  private readonly pinnedChunks = new Set<string>();
  /** Sparse journal. Survives `releaseFarChunks`. */
  private readonly overrides = new Map<string, MapLayers>();

  constructor(seed: number, chunkSize: number = DEFAULT_CHUNK_SIZE) {
    this.seed = seed;
    this.chunkSize = chunkSize;
  }

  static fromSnapshot(snapshot: WorldSnapshot): ChunkWorld {
    const world = new ChunkWorld(snapshot.seed, snapshot.chunkSize);
    for (const chunk of snapshot.chunks) {
      const key = chunkKey(chunk.cx, chunk.cy);
      world.pinnedChunks.add(key);
      world.chunks.set(key, {
        cx: chunk.cx,
        cy: chunk.cy,
        tiles: chunk.tiles.slice(),
        trees: chunk.trees.slice(),
        flora: chunk.flora.slice(),
        features: blankFeatures(snapshot.chunkSize),
      });
    }
    return world;
  }

  toSnapshot(): WorldSnapshot {
    return {
      seed: this.seed,
      chunkSize: this.chunkSize,
      chunks: [...this.chunks.values()].map((chunk) => ({
        cx: chunk.cx,
        cy: chunk.cy,
        tiles: chunk.tiles.slice(),
        trees: chunk.trees.slice(),
        flora: chunk.flora.slice(),
      })),
    };
  }

  /** Remember the journal, then paint any chunks that are already generated. */
  loadCells(cells: readonly MapCell[]): void {
    for (const cell of cells) {
      this.applyCell(cell);
    }
  }

  /** Merge one cell. Applying the same patch twice leaves the same layers. */
  applyCell(cell: MapCell): void {
    const key = `${cell.x},${cell.y}`;
    const stored = { ...(this.overrides.get(key) ?? {}) };
    for (const [name, value] of Object.entries(cell.layers)) {
      if (typeof value === 'string' || value === null) {
        stored[name] = value;
      }
    }
    this.overrides.set(key, stored);

    const located = this.worldToChunk(cell.x, cell.y);
    const chunk = this.chunks.get(chunkKey(located.cx, located.cy));
    if (!chunk) {
      return;
    }
    this.paint(chunk, located.lx, located.ly, cell.layers);
  }

  /**
   * Cells whose durability layer holds a positive remainder.
   * A cleared tree hides the bar even if a stale number is still stored.
   */
  damagedCells(): Array<{ x: number; y: number; remaining: number; max: number }> {
    const found: Array<{ x: number; y: number; remaining: number; max: number }> = [];
    for (const [key, layers] of this.overrides) {
      if (layers.tree === null) {
        continue;
      }
      const remaining = remainingDurability(layers.durability);
      if (remaining === null) {
        continue;
      }
      const [xText, yText] = key.split(',');
      const x = Number(xText);
      const y = Number(yText);
      found.push({ x, y, remaining, max: this.breakableMax(x, y) });
    }
    return found;
  }

  /** Полная прочность клетки: камень дольше дерева. Осколки сюда не входят. */
  private breakableMax(tileX: number, tileY: number): number {
    if (this.hasTreeAt(tileX, tileY)) {
      return nodeDurability[NodeKind.Tree];
    }
    if (this.getTileAt(tileX, tileY) === TileType.Stone) {
      return nodeDurability[NodeKind.Stone];
    }
    return nodeDurability[NodeKind.Tree];
  }

  /**
   * Flora on a generated chunk. Does not create chunks: the tiles next to the
   * player are already loaded, and a distant read must not force generation.
   */
  floraAt(tileX: number, tileY: number): FloraKind | null {
    const { cx, cy, lx, ly } = this.worldToChunk(tileX, tileY);
    const chunk = this.chunks.get(chunkKey(cx, cy));
    if (!chunk) {
      return null;
    }
    return chunk.flora[ly * this.chunkSize + lx] ?? null;
  }

  hasChunk(cx: number, cy: number): boolean {
    return this.chunks.has(chunkKey(cx, cy));
  }

  listChunkCoords(): Array<{ cx: number; cy: number }> {
    return [...this.chunks.values()].map((c) => ({ cx: c.cx, cy: c.cy }));
  }

  /** Create chunk if missing; returns whether it was newly generated. */
  ensureChunk(cx: number, cy: number): { chunk: ChunkData; created: boolean } {
    const key = chunkKey(cx, cy);
    const existing = this.chunks.get(key);
    if (existing) {
      return { chunk: existing, created: false };
    }

    const terrain = generateChunkTerrain(this.seed, cx, cy, this.chunkSize);
    const chunk: ChunkData = {
      cx,
      cy,
      tiles: terrain.tiles,
      trees: terrain.trees,
      flora: terrain.flora,
      features: blankFeatures(this.chunkSize),
    };
    this.applyStored(chunk);
    this.chunks.set(key, chunk);
    return { chunk, created: true };
  }

  worldToChunk(tileX: number, tileY: number): { cx: number; cy: number; lx: number; ly: number } {
    const cx = floorDiv(tileX, this.chunkSize);
    const cy = floorDiv(tileY, this.chunkSize);
    return {
      cx,
      cy,
      lx: mod(tileX, this.chunkSize),
      ly: mod(tileY, this.chunkSize),
    };
  }

  /**
   * Tile at world coordinates. Generates the owning chunk on demand.
   * Always returns grass|water (infinite world — never null).
   */
  getTileAt(tileX: number, tileY: number): TileType {
    const { cx, cy, lx, ly } = this.worldToChunk(tileX, tileY);
    const { chunk } = this.ensureChunk(cx, cy);
    return chunk.tiles[ly * this.chunkSize + lx] ?? TileType.Grass;
  }

  /** Whether a small tree occupies this cell. Generates chunk on demand. */
  hasTreeAt(tileX: number, tileY: number): boolean {
    const { cx, cy, lx, ly } = this.worldToChunk(tileX, tileY);
    const { chunk } = this.ensureChunk(cx, cy);
    return chunk.trees[ly * this.chunkSize + lx] ?? false;
  }

  /** Walkable if terrain allows and there is no tree. */
  isWalkableAt(tileX: number, tileY: number): boolean {
    if (!TILE_WALKABLE[this.getTileAt(tileX, tileY)]) {
      return false;
    }
    return !this.hasTreeAt(tileX, tileY);
  }

  ensureChunksAround(tileX: number, tileY: number, radiusChunks: number): Array<{ cx: number; cy: number; created: boolean }> {
    const { cx, cy } = this.worldToChunk(tileX, tileY);
    const result: Array<{ cx: number; cy: number; created: boolean }> = [];

    for (let dy = -radiusChunks; dy <= radiusChunks; dy += 1) {
      for (let dx = -radiusChunks; dx <= radiusChunks; dx += 1) {
        const ncx = cx + dx;
        const ncy = cy + dy;
        const { created } = this.ensureChunk(ncx, ncy);
        result.push({ cx: ncx, cy: ncy, created });
      }
    }

    return result;
  }

  /**
   * Forget generated chunks outside the Chebyshev radius around the tile.
   * Pinned snapshot chunks and journal overrides are kept.
   */
  releaseFarChunks(tileX: number, tileY: number, keepRadiusChunks: number): void {
    const { cx, cy } = this.worldToChunk(tileX, tileY);
    const stale: string[] = [];

    for (const [key, chunk] of this.chunks) {
      if (this.pinnedChunks.has(key)) {
        continue;
      }
      if (
        Math.abs(chunk.cx - cx) > keepRadiusChunks ||
        Math.abs(chunk.cy - cy) > keepRadiusChunks
      ) {
        stale.push(key);
      }
    }

    for (const key of stale) {
      this.chunks.delete(key);
    }
  }

  private applyStored(chunk: ChunkData): void {
    const originX = chunk.cx * this.chunkSize;
    const originY = chunk.cy * this.chunkSize;
    for (let ly = 0; ly < this.chunkSize; ly += 1) {
      for (let lx = 0; lx < this.chunkSize; lx += 1) {
        const layers = this.overrides.get(`${originX + lx},${originY + ly}`);
        if (!layers) {
          continue;
        }
        this.paint(chunk, lx, ly, layers);
      }
    }
  }

  private paint(chunk: ChunkData, lx: number, ly: number, layers: MapLayers): void {
    const index = ly * this.chunkSize + lx;
    const painters: Record<string, (value: string | null) => void> = {
      terrain: (value) => {
        if (value === null) {
          return;
        }
        const tile = tileByName[value];
        if (tile) {
          chunk.tiles[index] = tile;
        }
      },
      tree: (value) => {
        if (value === null) {
          chunk.trees[index] = false;
          return;
        }
        if (value === 'present') {
          chunk.trees[index] = true;
        }
      },
      flora: (value) => {
        if (value === null) {
          chunk.flora[index] = null;
          return;
        }
        const kind = floraByName[value];
        if (kind) {
          chunk.flora[index] = kind;
        }
      },
      feature: (value) => {
        chunk.features[index] = value;
      },
    };
    for (const [name, value] of Object.entries(layers)) {
      if (typeof value !== 'string' && value !== null) {
        continue;
      }
      painters[name]?.(value);
    }
  }
}
