import { craftOpenKey } from '../craft/open';
import { inventoryOpenKey } from '../inventory/open';

type FlagRegistry = {
  get(key: string): unknown;
};

export function isPlayPaused(registry: FlagRegistry): boolean {
  return registry.get(inventoryOpenKey) === true || registry.get(craftOpenKey) === true;
}
