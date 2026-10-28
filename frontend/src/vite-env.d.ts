/// <reference types="vite/client" />

interface ImportMetaEnv {
  readonly VITE_DEV_PORT?: string;
  readonly VITE_PREVIEW_PORT?: string;
  readonly VITE_CHUNK_SIZE?: string;
  readonly VITE_TILE_SIZE?: string;
  readonly VITE_PLAYER_SPEED?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}
