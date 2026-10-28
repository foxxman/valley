/** Разрушаемые объекты. Число — полная прочность голой рукой: дерево 10 ударов, камень 20. */
export const NodeKind = {
  Tree: 'tree',
  Stone: 'stone',
} as const;

export type NodeKind = (typeof NodeKind)[keyof typeof NodeKind];

export const nodeDurability: Record<NodeKind, number> = {
  [NodeKind.Tree]: 10,
  [NodeKind.Stone]: 20,
};
