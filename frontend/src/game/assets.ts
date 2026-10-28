/** Curated textures under /public/assets/{soil,water,stones,vegetation,items}. */
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
  Wood: 'wood',
  ItemStone: 'stone',
  Player: 'player',
} as const;

export type AssetKey = (typeof AssetKey)[keyof typeof AssetKey];

export const ASSET_URLS: Record<AssetKey, string> = {
  [AssetKey.Grass]: '/assets/soil/grass.png',
  [AssetKey.Water]: '/assets/water/water.png',
  [AssetKey.ShoreStraight]: '/assets/water/shore_straight.png',
  [AssetKey.ShoreCorner]: '/assets/water/shore_corner.png',
  [AssetKey.ShoreCornerInner]: '/assets/water/shore_corner_inner.png',
  [AssetKey.Stone]: '/assets/stones/stone.png',
  [AssetKey.StoneShards]: '/assets/stones/stone_shards.png',
  [AssetKey.TreeSmall]: '/assets/vegetation/tree_small.png',
  [AssetKey.Flower0]: '/assets/vegetation/flower_0.png',
  [AssetKey.Flower1]: '/assets/vegetation/flower_1.png',
  [AssetKey.Flower2]: '/assets/vegetation/flower_2.png',
  [AssetKey.Flower3]: '/assets/vegetation/flower_3.png',
  [AssetKey.Lily]: '/assets/vegetation/lily.png',
  [AssetKey.GrassTuft1]: '/assets/vegetation/grass_tuft_1.png',
  [AssetKey.Wood]: '/assets/items/wood.png',
  [AssetKey.ItemStone]: '/assets/items/stone.png',
  [AssetKey.Player]: '/assets/player.png',
};
