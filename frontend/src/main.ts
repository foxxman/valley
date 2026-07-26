import Phaser from 'phaser';
import { createGameConfig } from './game/config';

const root = document.querySelector<HTMLElement>('#app');

if (!root) {
  throw new Error('Root element #app not found');
}

new Phaser.Game(createGameConfig(root));
