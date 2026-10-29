# Curated game textures

Organized under `public/assets/`:

| Folder | Contents |
|--------|----------|
| `soil/` | base ground (`grass.png`) |
| `water/` | water and shore |
| `stones/` | stone and shards |
| `vegetation/` | trees, flowers, tufts, lilies |
| `items/` | inventory icons that are not placed on the map |
| `player.png` | player (assets root) |

| File | Role | Base orientation (angle 0) |
|------|------|------------------------------|
| `soil/grass.png` | Interior plains | — |
| `water/water.png` | Pond interior | — |
| `water/shore_straight.png` | Straight shore | grass on TOP |
| `water/shore_corner.png` | Outer corner (land wraps water) | grass TOP-LEFT |
| `water/shore_corner_inner.png` | Inner corner (land tip on water) | corner TOP-LEFT |
| `stones/stone.png` | Solid stone (blocked) | — |
| `stones/stone_shards.png` | Stone shards on grass (walkable) | — |
| `vegetation/tree_small.png` | Small tree overlay (on grass) | — |
| `vegetation/flower_0.png` … `flower_3.png` | Flower overlays (walkable) | — |
| `vegetation/lily.png` | Lilies on water (overlay) | — |
| `vegetation/grass_tuft_1.png` | Grass tuft on land (overlay) | — |
| `items/wood.png` | Wood item icon (bag only) | — |
| `items/stone.png` | Stone item icon (bag only), Kenney Tiny Farm | — |
| `items/chest.png` | Chest item icon (bag only), Kenney Roguelike RPG | — |
| `player.png` | Player | — |

Source: Kenney Roguelike RPG / Characters (CC0).

Autotile: `src/game/world/terrainVisual.ts`.
Generation: `generate-chunk/`.
