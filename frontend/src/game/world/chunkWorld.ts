import { generateChunkTerrain } from './generate-chunk';
import type { FloraKind } from './flora';
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

function chunkKey(cx: number, cy: number): string {
  return `${cx},${cy}`;
}

function floorDiv(value: number, size: number): number {
  return Math.floor(value / size);
}

function mod(value: number, size: number): number {
  return ((value % size) + size) % size;
}

/**
 * Infinite chunked world. Chunks generate on first access and stay in memory
 * (session cache). Use `toSnapshot` / `fromSnapshot` when persistence is added.
 */
export class ChunkWorld {
  readonly seed: number;
  readonly chunkSize: number;

  private readonly chunks = new Map<string, ChunkData>();

  constructor(seed: number, chunkSize: number = DEFAULT_CHUNK_SIZE) {
    this.seed = seed;
    this.chunkSize = chunkSize;
  }

  static fromSnapshot(snapshot: WorldSnapshot): ChunkWorld {
    const world = new ChunkWorld(snapshot.seed, snapshot.chunkSize);
    for (const chunk of snapshot.chunks) {
      world.chunks.set(chunkKey(chunk.cx, chunk.cy), {
        cx: chunk.cx,
        cy: chunk.cy,
        tiles: chunk.tiles.slice(),
        trees: chunk.trees.slice(),
        flora: chunk.flora.slice(),
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
    };
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
}
