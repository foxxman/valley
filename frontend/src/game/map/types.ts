export type MapLayers = Record<string, string | null>;

export type MapCell = {
  x: number;
  y: number;
  layers: MapLayers;
};

export type MapJournal = {
  cells: MapCell[];
};

export type MapErrorCode =
  | 'game_not_found'
  | 'unauthorized'
  | 'already_taken'
  | 'bag_full'
  | 'not_in_reach'
  | 'invalid_kind'
  | 'unknown';

export type MapResult<T> =
  | { ok: true; data: T }
  | { ok: false; status: number; code: MapErrorCode; message: string };
