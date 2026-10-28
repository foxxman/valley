import { FloraKind } from './flora';
import { NodeKind } from './nodes';
import { TileType } from './tiles';

const flowers: Record<string, FloraKind> = {
  [FloraKind.Flower0]: FloraKind.Flower0,
  [FloraKind.Flower1]: FloraKind.Flower1,
  [FloraKind.Flower2]: FloraKind.Flower2,
  [FloraKind.Flower3]: FloraKind.Flower3,
};

export type GatherTarget = {
  x: number;
  y: number;
  kind: FloraKind;
};

/** Под ногами, иначе ортогональный сосед по взгляду. Трава и кувшинки не цветы. */
export function gatherTarget(
  floraAt: (x: number, y: number) => FloraKind | null,
  tileX: number,
  tileY: number,
  facing: { x: number; y: number },
): GatherTarget | null {
  const under = flowers[floraAt(tileX, tileY) ?? ''];
  if (under) {
    return { x: tileX, y: tileY, kind: under };
  }
  const ahead = flowers[floraAt(tileX + facing.x, tileY + facing.y) ?? ''];
  if (ahead) {
    return { x: tileX + facing.x, y: tileY + facing.y, kind: ahead };
  }
  return null;
}

export const ActionKind = {
  Gather: 'gather',
  Strike: 'strike',
} as const;

export type ActionKind = (typeof ActionKind)[keyof typeof ActionKind];

export type ActionTarget = {
  action: ActionKind;
  x: number;
  y: number;
  kind: string;
};

/** Дерево или сплошной камень по взгляду важнее цветка: на эти клетки встать нельзя. */
export function actionTarget(
  floraAt: (x: number, y: number) => FloraKind | null,
  hasTreeAt: (x: number, y: number) => boolean,
  tileAt: (x: number, y: number) => TileType,
  tileX: number,
  tileY: number,
  facing: { x: number; y: number },
): ActionTarget | null {
  const aheadX = tileX + facing.x;
  const aheadY = tileY + facing.y;
  if (hasTreeAt(aheadX, aheadY)) {
    return { action: ActionKind.Strike, x: aheadX, y: aheadY, kind: NodeKind.Tree };
  }
  if (tileAt(aheadX, aheadY) === TileType.Stone) {
    return { action: ActionKind.Strike, x: aheadX, y: aheadY, kind: NodeKind.Stone };
  }
  const flower = gatherTarget(floraAt, tileX, tileY, facing);
  if (!flower) {
    return null;
  }
  return { action: ActionKind.Gather, x: flower.x, y: flower.y, kind: flower.kind };
}
