export type InventoryItem = {
  id: string;
  slug: string;
  name: string;
  iconKey: string;
  description: string;
  maxStack: number;
};

export type InventorySlot = {
  index: number;
  quantity: number;
  item: InventoryItem;
};

export type Inventory = {
  capacity: number;
  slots: InventorySlot[];
};

export type InventoryErrorCode = 'game_not_found' | 'unauthorized' | 'unknown';

export type InventoryResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; code: InventoryErrorCode; message: string };
