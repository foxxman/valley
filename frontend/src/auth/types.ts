export type Player = {
  id: string;
  login: string;
};

export type AuthErrorCode =
  | 'invalid_body'
  | 'invalid_login'
  | 'invalid_password'
  | 'login_taken'
  | 'invalid_credentials'
  | 'unauthorized'
  | 'unknown';

export type AuthResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; code: AuthErrorCode; message: string };
