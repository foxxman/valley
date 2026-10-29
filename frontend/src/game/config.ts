import Phaser from 'phaser';
import type { Inventory } from './inventory/types';
import type { MapCell } from './map/types';
import { WorldScene } from './scenes/WorldScene';

export type PlaySession = {
  gameId: string;
  worldSeed: number;
  playerId: string;
  login: string;
  cells: MapCell[];
  onToggleInventory: () => void;
  onCloseInventory: () => void;
  onToggleCraft: () => void;
  onCloseCraft: () => void;
  onInventory: (inventory: Inventory) => void;
};

export const playSessionKey = 'playSession';

export function createGameConfig(
  parent: HTMLElement,
  session: PlaySession,
): Phaser.Types.Core.GameConfig {
  return {
    type: Phaser.AUTO,
    parent,
    backgroundColor: '#0b1020',
    pixelArt: true,
    scale: {
      mode: Phaser.Scale.RESIZE,
      width: parent.clientWidth || window.innerWidth,
      height: parent.clientHeight || window.innerHeight,
      autoCenter: Phaser.Scale.CENTER_BOTH,
    },
    scene: [WorldScene],
    banner: false,
    callbacks: {
      preBoot(game) {
        game.registry.set(playSessionKey, session);
      },
    },
  };
}
