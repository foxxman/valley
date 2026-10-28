import Phaser from 'phaser';
import { createGameConfig, type PlaySession } from './config';

export function mountGame(parent: HTMLElement, session: PlaySession): Phaser.Game {
  return new Phaser.Game(createGameConfig(parent, session));
}
