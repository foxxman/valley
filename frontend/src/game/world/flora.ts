import { AssetKey } from '../assets';

/** Проходимые оверлеи (не деревья). Кувшинки — на воде, остальное — на траве. */
export const FloraKind = {
  Flower0: 'flower-0',
  Flower1: 'flower-1',
  Flower2: 'flower-2',
  Flower3: 'flower-3',
  Lily: 'lily',
  Grass1: 'grass-1',
} as const;

export type FloraKind = (typeof FloraKind)[keyof typeof FloraKind];

export const FLOWER_KINDS: readonly FloraKind[] = [
  FloraKind.Flower0,
  FloraKind.Flower1,
  FloraKind.Flower2,
  FloraKind.Flower3,
];

export const GRASS_KINDS: readonly FloraKind[] = [FloraKind.Grass1];

export const LILY_KINDS: readonly FloraKind[] = [FloraKind.Lily];

export const FLORA_ASSET: Record<FloraKind, AssetKey> = {
  [FloraKind.Flower0]: AssetKey.Flower0,
  [FloraKind.Flower1]: AssetKey.Flower1,
  [FloraKind.Flower2]: AssetKey.Flower2,
  [FloraKind.Flower3]: AssetKey.Flower3,
  [FloraKind.Lily]: AssetKey.Lily,
  [FloraKind.Grass1]: AssetKey.GrassTuft1,
};
