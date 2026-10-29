import type { Inventory } from '../inventory/types';
import type { Recipe } from './types';

export type CraftBlock = 'missing' | 'full';

type MutableSlot = {
  index: number;
  slug: string;
  quantity: number;
};

/** Mirrors inventory.planExchange. The server result wins if the two disagree. */
export function craftBlock(inventory: Inventory, recipe: Recipe): CraftBlock | null {
  const slots: MutableSlot[] = inventory.slots.map((slot) => ({
    index: slot.index,
    slug: slot.item.slug,
    quantity: slot.quantity,
  }));
  slots.sort((left, right) => left.index - right.index);

  for (const ingredient of recipe.ingredients) {
    let remaining = ingredient.quantity;
    for (const slot of slots) {
      if (remaining === 0) {
        break;
      }
      if (slot.slug !== ingredient.slug || slot.quantity === 0) {
        continue;
      }
      const take = Math.min(slot.quantity, remaining);
      slot.quantity -= take;
      remaining -= take;
    }
    if (remaining > 0) {
      return 'missing';
    }
  }

  let remaining = recipe.result.quantity;
  for (const slot of slots) {
    if (remaining === 0) {
      break;
    }
    if (slot.quantity === 0 || slot.slug !== recipe.result.slug) {
      continue;
    }
    const room = recipe.result.maxStack - slot.quantity;
    if (room <= 0) {
      continue;
    }
    const add = Math.min(room, remaining);
    slot.quantity += add;
    remaining -= add;
  }

  const used = new Set(slots.filter((slot) => slot.quantity > 0).map((slot) => slot.index));
  for (let index = 0; index < inventory.capacity && remaining > 0; index += 1) {
    if (used.has(index)) {
      continue;
    }
    remaining -= Math.min(recipe.result.maxStack, remaining);
    used.add(index);
  }
  if (remaining > 0) {
    return 'full';
  }
  return null;
}
