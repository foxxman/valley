import { gameErrorMessage } from './messages';
import type { Game, GameErrorCode, GameResult } from './types';

const path = {
  collection: '/api/games',
} as const;

type ErrorPayload = {
  error?: {
    code?: string;
  };
};

const knownCodes: Record<string, GameErrorCode> = {
  invalid_body: 'invalid_body',
  invalid_name: 'invalid_name',
  game_not_found: 'game_not_found',
  forbidden: 'forbidden',
  unauthorized: 'unauthorized',
};

export function createGame(name: string): Promise<GameResult<Game>> {
  return send<Game>(path.collection, 'POST', { name });
}

export function listMyGames(): Promise<GameResult<Game[]>> {
  return send<{ games: Game[] }>(path.collection, 'GET').then((result) => {
    if (!result.ok) {
      return result;
    }
    return { ok: true, data: result.data.games };
  });
}

export function findGame(id: string): Promise<GameResult<Game>> {
  return send<Game>(`${path.collection}/${encodeURIComponent(id)}`, 'GET');
}

export function renameGame(id: string, name: string): Promise<GameResult<Game>> {
  return send<Game>(`${path.collection}/${encodeURIComponent(id)}`, 'PATCH', { name });
}

async function send<T>(url: string, method: 'GET' | 'POST' | 'PATCH', body?: unknown): Promise<GameResult<T>> {
  const response = await fetch(url, {
    method,
    credentials: 'include',
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  const payload = (await response.json().catch(() => null)) as T | ErrorPayload | null;
  if (!response.ok) {
    const rawCode = isErrorPayload(payload) ? payload.error?.code : undefined;
    const code = (rawCode && knownCodes[rawCode]) || 'unknown';
    return {
      ok: false,
      status: response.status,
      code,
      message: gameErrorMessage[code],
    };
  }

  return { ok: true, data: payload as T };
}

function isErrorPayload(value: unknown): value is ErrorPayload {
  return typeof value === 'object' && value !== null && 'error' in value;
}
