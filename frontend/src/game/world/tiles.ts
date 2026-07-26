export const TileType = {
  Grass: 'grass',
  Water: 'water',
  Stone: 'stone',
  StoneShards: 'stone-shards',
} as const;

export type TileType = (typeof TileType)[keyof typeof TileType];

export const TILE_WALKABLE: Record<TileType, boolean> = {
  [TileType.Grass]: true,
  [TileType.Water]: false,
  [TileType.Stone]: false,
  [TileType.StoneShards]: true,
};

/** Не-вода считается сушей для автотайла берега. */
export function isLandTile(tile: TileType): boolean {
  return tile !== TileType.Water;
}
