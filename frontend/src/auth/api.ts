import { authErrorMessage } from './messages';
import type { AuthErrorCode, AuthResult, Player } from './types';

const path = {
  register: '/api/auth/register',
  login: '/api/auth/login',
  refresh: '/api/auth/refresh',
  logout: '/api/auth/logout',
  me: '/api/auth/me',
} as const;

type ErrorPayload = {
  error?: {
    code?: string;
    message?: string;
  };
};

const knownCodes: Record<string, AuthErrorCode> = {
  invalid_body: 'invalid_body',
  invalid_login: 'invalid_login',
  invalid_password: 'invalid_password',
  login_taken: 'login_taken',
  invalid_credentials: 'invalid_credentials',
  unauthorized: 'unauthorized',
};

export function register(login: string, password: string): Promise<AuthResult<Player>> {
  return sendJSON(path.register, { login, password });
}

export function login(loginName: string, password: string): Promise<AuthResult<Player>> {
  return sendJSON(path.login, { login: loginName, password });
}

export function refresh(): Promise<AuthResult<Player>> {
  return sendJSON(path.refresh);
}

export function logout(): Promise<AuthResult<void>> {
  return sendJSON(path.logout);
}

export function me(): Promise<AuthResult<Player>> {
  return sendJSON(path.me, undefined, 'GET');
}

export async function restoreSession(): Promise<Player | null> {
  const current = await me();
  if (current.ok) {
    return current.data;
  }
  if (current.status !== 401) {
    throw new Error(current.message);
  }

  const refreshed = await refresh();
  if (refreshed.ok) {
    return refreshed.data;
  }
  if (refreshed.status === 401) {
    return null;
  }
  throw new Error(refreshed.message);
}

async function sendJSON<T>(
  url: string,
  body?: unknown,
  method: 'GET' | 'POST' = 'POST',
): Promise<AuthResult<T>> {
  const response = await fetch(url, {
    method,
    credentials: 'include',
    headers: body === undefined ? undefined : { 'Content-Type': 'application/json' },
    body: body === undefined ? undefined : JSON.stringify(body),
  });

  if (response.status === 204) {
    return { ok: true, data: undefined as T };
  }

  const payload = (await response.json().catch(() => null)) as T | ErrorPayload | null;
  if (!response.ok) {
    const rawCode = isErrorPayload(payload) ? payload.error?.code : undefined;
    const code = (rawCode && knownCodes[rawCode]) || 'unknown';
    return {
      ok: false,
      status: response.status,
      code,
      message: authErrorMessage[code],
    };
  }

  return { ok: true, data: payload as T };
}

function isErrorPayload(value: unknown): value is ErrorPayload {
  return typeof value === 'object' && value !== null && 'error' in value;
}
