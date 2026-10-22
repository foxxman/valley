export const maxGameNameLength = 40;

export type GameOwner = {
  id: string;
  login: string;
};

export type Game = {
  id: string;
  name: string;
  owner: GameOwner;
  createdAt: string;
  updatedAt: string;
};

export type GameErrorCode =
  | 'invalid_body'
  | 'invalid_name'
  | 'game_not_found'
  | 'forbidden'
  | 'unauthorized'
  | 'unknown';

export type GameResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; code: GameErrorCode; message: string };
