/** Known item icons. A missing key leaves the cell labeled with the item name. */
const itemIcons: Record<string, string> = {};

export function itemIconUrl(iconKey: string): string | undefined {
  return itemIcons[iconKey];
}
