import { FloraKind } from './flora';

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
