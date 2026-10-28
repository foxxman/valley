import type { Inventory } from '../inventory/types';
import { mapErrorMessage } from './messages';
import type { MapCell, MapErrorCode, MapJournal, MapResult } from './types';

const knownCodes: Record<string, MapErrorCode> = {
  game_not_found: 'game_not_found',
  unauthorized: 'unauthorized',
  already_taken: 'already_taken',
  bag_full: 'bag_full',
  not_in_reach: 'not_in_reach',
  invalid_kind: 'invalid_kind',
};

type ErrorPayload = {
  error?: {
    code?: string;
  };
};

export type Gathered = Inventory & {
  cell: MapCell;
};

export function loadMap(gameId: string): Promise<MapResult<MapJournal>> {
  return send<MapJournal>(`/api/games/${encodeURIComponent(gameId)}/map`);
}

export function gatherFlower(
  gameId: string,
  body: { x: number; y: number; kind: string },
): Promise<MapResult<Gathered>> {
  return send<Gathered>(`/api/games/${encodeURIComponent(gameId)}/actions/gather`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

async function send<T>(url: string, init?: RequestInit): Promise<MapResult<T>> {
  let response: Response;
  try {
    response = await fetch(url, { credentials: 'include', ...init });
  } catch {
    return { ok: false, status: 0, code: 'unknown', message: mapErrorMessage.unknown };
  }
  const payload = (await response.json().catch(() => null)) as T | ErrorPayload | null;
  if (!response.ok) {
    const rawCode = isErrorPayload(payload) ? payload.error?.code : undefined;
    const code = (rawCode && knownCodes[rawCode]) || 'unknown';
    return {
      ok: false,
      status: response.status,
      code,
      message: mapErrorMessage[code],
    };
  }
  return { ok: true, data: payload as T };
}

function isErrorPayload(value: unknown): value is ErrorPayload {
  return typeof value === 'object' && value !== null && 'error' in value;
}
