import Phaser from 'phaser';
import { WorldScene } from './scenes/WorldScene';

export type PlaySession = {
  gameId: string;
  worldSeed: number;
  playerId: string;
  login: string;
  onToggleInventory: () => void;
  onCloseInventory: () => void;
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
