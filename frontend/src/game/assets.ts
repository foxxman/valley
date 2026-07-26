/** Curated textures under /public/assets/game (from Kenney packs + custom shores). */
export const AssetKey = {
  Grass: 'tile-grass',
  Water: 'tile-water',
  ShoreStraight: 'tile-shore-straight',
  ShoreCorner: 'tile-shore-corner',
  ShoreCornerInner: 'tile-shore-corner-inner',
  Stone: 'tile-stone',
  StoneShards: 'tile-stone-shards',
  TreeSmall: 'tree-small',
  Flower0: 'flower-0',
  Flower1: 'flower-1',
  Flower2: 'flower-2',
  Flower3: 'flower-3',
  Lily: 'lily',
  GrassTuft1: 'grass-tuft-1',
  Player: 'player',
} as const;

export type AssetKey = (typeof AssetKey)[keyof typeof AssetKey];

export const ASSET_URLS: Record<AssetKey, string> = {
  [AssetKey.Grass]: '/assets/game/grass.png',
  [AssetKey.Water]: '/assets/game/water.png',
  [AssetKey.ShoreStraight]: '/assets/game/shore_straight.png',
  [AssetKey.ShoreCorner]: '/assets/game/shore_corner.png',
  [AssetKey.ShoreCornerInner]: '/assets/game/shore_corner_inner.png',
  [AssetKey.Stone]: '/assets/game/stone.png',
  [AssetKey.StoneShards]: '/assets/game/stone_shards.png',
  [AssetKey.TreeSmall]: '/assets/game/tree_small.png',
  [AssetKey.Flower0]: '/assets/game/flower_0.png',
  [AssetKey.Flower1]: '/assets/game/flower_1.png',
  [AssetKey.Flower2]: '/assets/game/flower_2.png',
  [AssetKey.Flower3]: '/assets/game/flower_3.png',
  [AssetKey.Lily]: '/assets/game/grass_tuft_0.png',
  [AssetKey.GrassTuft1]: '/assets/game/grass_tuft_1.png',
  [AssetKey.Player]: '/assets/game/player.png',
};
