import { inventoryErrorMessage } from './messages';
import type { Inventory, InventoryErrorCode, InventoryResult } from './types';

const knownCodes: Record<string, InventoryErrorCode> = {
  game_not_found: 'game_not_found',
  unauthorized: 'unauthorized',
};

type ErrorPayload = {
  error?: {
    code?: string;
  };
};

export function loadInventory(gameId: string): Promise<InventoryResult<Inventory>> {
  return send(`/api/games/${encodeURIComponent(gameId)}/inventory`);
}

async function send(url: string): Promise<InventoryResult<Inventory>> {
  const response = await fetch(url, { credentials: 'include' });
  const payload = (await response.json().catch(() => null)) as Inventory | ErrorPayload | null;
  if (!response.ok) {
    const rawCode = isErrorPayload(payload) ? payload.error?.code : undefined;
    const code = (rawCode && knownCodes[rawCode]) || 'unknown';
    return {
      ok: false,
      status: response.status,
      code,
      message: inventoryErrorMessage[code],
    };
  }
  return { ok: true, data: payload as Inventory };
}

function isErrorPayload(value: unknown): value is ErrorPayload {
  return typeof value === 'object' && value !== null && 'error' in value;
}
