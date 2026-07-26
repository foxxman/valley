# Game textures (curated)

Selected / custom frames for the prototype.

| File | Role | Base orientation (angle 0) |
|------|------|------------------------------|
| `grass.png` | Interior plains | — |
| `water.png` | Pond interior | — |
| `shore_straight.png` | Прямой берег | трава СВЕРХУ |
| `shore_corner.png` | Внешний угол (суша огибает воду) | трава ВЕРХ-ЛЕВО |
| `shore_corner_inner.png` | Внутренний уголок (кусочек суши на воде) | уголок ВЕРХ-ЛЕВО |
| `stone.png` | Solid stone (blocked) | — |
| `stone_shards.png` | Stone shards on grass (walkable) | — |
| `tree_small.png` | Small tree overlay (on grass) | — |
| `flower_0.png` … `flower_3.png` | Flower overlays (walkable) | — |
| `grass_tuft_0.png` | Кувшинки на воде (overlay) | — |
| `grass_tuft_1.png` | Пучок травы на суше (overlay) | — |
| `player.png` | Player | — |

Источник: Kenney Roguelike RPG / Characters (CC0).
`shore_corner_inner.png` — спрайтшит row 2, col 1 (набор уголков 1-0…2-1).
`tree_small.png` — row 9, col 13. Flowers: row 9, cols 28–31. Grass tufts: row 10, cols 28 и 22.
Stone / shards: (0,9) / (1,9).

Автотайл: `src/game/world/terrainVisual.ts`.
Генерация: озеро-blob (~40%) + хребет (~50%) + декор — `generate-chunk/`.
