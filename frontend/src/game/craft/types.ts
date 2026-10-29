export type RecipePart = {
  slug: string;
  name: string;
  iconKey: string;
  quantity: number;
  maxStack: number;
};

export type Recipe = {
  slug: string;
  result: RecipePart;
  ingredients: RecipePart[];
};

export type CraftErrorCode =
  | 'game_not_found'
  | 'unauthorized'
  | 'missing_ingredients'
  | 'bag_full'
  | 'unknown_recipe'
  | 'unknown';

export type CraftResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; code: CraftErrorCode; message: string };
