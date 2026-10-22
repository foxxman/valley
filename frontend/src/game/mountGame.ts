import Phaser from 'phaser';
import { createGameConfig } from './config';

export function mountGame(parent: HTMLElement): Phaser.Game {
  return new Phaser.Game(createGameConfig(parent));
}
