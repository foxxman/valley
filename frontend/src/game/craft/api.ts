import type { Inventory } from '../inventory/types';
import { craftErrorMessage } from './messages';
import type { CraftErrorCode, CraftResult, Recipe } from './types';

const knownCodes: Record<string, CraftErrorCode> = {
  game_not_found: 'game_not_found',
  unauthorized: 'unauthorized',
  missing_ingredients: 'missing_ingredients',
  bag_full: 'bag_full',
  unknown_recipe: 'unknown_recipe',
};

type ErrorPayload = {
  error?: {
    code?: string;
  };
};

type RecipeListPayload = {
  recipes?: Recipe[];
};

export function loadRecipes(gameId: string): Promise<CraftResult<Recipe[]>> {
  return send<RecipeListPayload>(`/api/games/${encodeURIComponent(gameId)}/recipes`).then((result) => {
    if (!result.ok) {
      return result;
    }
    return { ok: true, data: result.data.recipes ?? [] };
  });
}

export function craftRecipe(gameId: string, recipe: string): Promise<CraftResult<Inventory>> {
  return send<Inventory>(`/api/games/${encodeURIComponent(gameId)}/actions/craft`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ recipe }),
  });
}

async function send<T>(url: string, init?: RequestInit): Promise<CraftResult<T>> {
  let response: Response;
  try {
    response = await fetch(url, { credentials: 'include', ...init });
  } catch {
    return { ok: false, status: 0, code: 'unknown', message: craftErrorMessage.unknown };
  }
  const payload = (await response.json().catch(() => null)) as T | ErrorPayload | null;
  if (!response.ok) {
    const rawCode = isErrorPayload(payload) ? payload.error?.code : undefined;
    const code = (rawCode && knownCodes[rawCode]) || 'unknown';
    return {
      ok: false,
      status: response.status,
      code,
      message: craftErrorMessage[code],
    };
  }
  return { ok: true, data: payload as T };
}

function isErrorPayload(value: unknown): value is ErrorPayload {
  return typeof value === 'object' && value !== null && 'error' in value;
}
