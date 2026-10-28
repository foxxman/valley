import { ASSET_URLS, AssetKey } from '../assets';

/** Known item icons. A missing key leaves the cell labeled with the item name. */
const itemIcons: Record<string, string> = {
  [AssetKey.Flower0]: ASSET_URLS[AssetKey.Flower0],
  [AssetKey.Flower1]: ASSET_URLS[AssetKey.Flower1],
  [AssetKey.Flower2]: ASSET_URLS[AssetKey.Flower2],
  [AssetKey.Flower3]: ASSET_URLS[AssetKey.Flower3],
  [AssetKey.Wood]: ASSET_URLS[AssetKey.Wood],
  [AssetKey.ItemStone]: ASSET_URLS[AssetKey.ItemStone],
};

export function itemIconUrl(iconKey: string): string | undefined {
  return itemIcons[iconKey];
}
