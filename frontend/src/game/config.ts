import Phaser from 'phaser';
import { WorldScene } from './scenes/WorldScene';

export function createGameConfig(parent: HTMLElement): Phaser.Types.Core.GameConfig {
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
  };
}
