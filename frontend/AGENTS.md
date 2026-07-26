# AGENTS.md — контекст для LLM / AI-агентов

Документ для языковых моделей: архитектура, ключевые алгоритмы и правила работы с этим фронтендом.
Человекочитаемый быстрый старт — в `README.md`.

## Назначение проекта

Браузерный **top-down 2D** прототип settlement/exploration игры.

Сейчас реализовано:

- бесконечный чанковый мир (генерация по запросу, 16×16);
- прямоугольные пруды + автотайл берега (straight / corner / water);
- каменные гряды (ядро 4–10 камней + обводка осколков 1–2);
- маленькие деревья на траве (~7.5%, непроходимы);
- цветы и пучки травы (~7.5% каждый, проходимы);
- ходьба персонажа (WASD / стрелки; деревья и камень непроходимы);
- камера всегда держит персонажа в центре;
- idle bounce (squash/stretch по вертикали).

Пока **нет**: меню, сохранения на диск, сервера, React UI, боя, NPC, строительства.
(Есть `ChunkWorld.toSnapshot` / `fromSnapshot` — заготовка под будущее сохранение, включая деревья и flora.)

Планируемый стек дальше: клиент (этот проект) + позже **Go**-бэкенд; React — только HUD/меню, не игровой цикл.

## Стек

| Слой | Технология |
|------|------------|
| Bundler / dev | Vite 6 |
| Язык | TypeScript (strict) |
| Игровой движок | Phaser 3 |
| Ассеты | Kenney CC0 (в `public/assets/`) |

Скрипты: `npm run dev` | `npm run build` | `npm run preview`.

Конфиг игры из env (`VITE_*`) — см. `.env.example` и `src/game/env.ts`.

## Архитектура

```text
index.html
  └─ src/main.ts
       └─ Phaser.Game(createGameConfig)
            └─ WorldScene
                 ├─ preload()     — текстуры
                 ├─ create()      — ChunkWorld + spawn chunk + Player + follow camera
                 └─ update()      — player; ensure/draw chunks; cull far images

Логика мира (без Phaser):
  world/tiles.ts          — TileType, walkability
  world/chunkWorld.ts     — бесконечные чанки + snapshot API
  world/generateChunk.ts  — пруды + деревья/flora внутри одного чанка
  world/flora.ts          — виды цветов/травы + маппинг на AssetKey
  world/terrainVisual.ts  — water/straight/corner

Сущности:
  entities/Player.ts      — ввод, коллизии (isWalkableAt), bounce
```

### Важные разделения

1. **Логическая карта** ≠ **визуальный тайл**.  
   В чанке `grass | water | stone | stone-shards` + слои `trees` / `flora`. Берег — в `resolveTerrainVisual`.

2. **Данные чанка** живут в `ChunkWorld` на всю сессию.  
   **Картинки** Phaser можно cull'ить вне viewport — при возврате рисуются снова из тех же tiles/trees/flora.

3. **Phaser** — цикл, камера, ввод, отрисовка. Генерация — чистые функции.

4. Предпочтение в TS: **объекты-словари вместо `switch`**.

## Поток (рантайм)

1. Старт: `ChunkWorld(seed)`, генерируется чанк `(0,0)` и соседи, спавн в центре `(0,0)`.
2. Камера: `startFollow(..., 1, 1)`, **без** `setBounds`.
3. Игрок идёт → `isWalkableAt` / `getTileAt` создаёт чанк при первом касании.
4. `update`: `ensureChunksAround` + отрисовка terrain + trees/flora; далеко — destroy images (данные остаются).
5. Будущий сейв: `world.toSnapshot()` (tiles + trees + flora) → JSON/сервер; загрузка — `ChunkWorld.fromSnapshot`.

## Алгоритм генерации чанка

Файл: `src/game/world/generateChunk.ts` (вызывается из `ChunkWorld.ensureChunk`).

1. Чанк 16×16 (или `VITE_CHUNK_SIZE`), вся трава.
2. Пруд — с шансом ~40% один прямоугольник ≥3×3, **inset ≥1** от края.
3. Каменная гряда — с шансом ~50%: связное ядро 4–10 камней + обводка осколков толщиной 1–2 (Chebyshev); без воды/декора на камне и осколках.
4. Чанк `(0,0)`: центр 3×3 без воды/камня/декора (спавн).
5. На чистой траве (кроме спавна): ~7.5% дерево, иначе ~7.5% цветок, иначе ~7.5% пучок травы.
6. Сид чанка = mix(`worldSeed`, `cx`, `cy`) — детерминированно.

## Алгоритм берега (автотайлинг)

Файл: `src/game/world/terrainVisual.ts`.

Только три визуала воды:

| landMask (соседи-трава) | Текстура | Смысл |
|-------------------------|----------|--------|
| 0 | `water.png` | середина пруда |
| N / E / S / W | `shore_straight` | прямой берег |
| N\|W, N\|E, S\|E, S\|W | `shore_corner` | угол: **суша огибает воду** |

Трава всегда `grass.png`. Inner / channel / deadend **не используются**.

Вода граничит с травой только через клетку берега (периметр прямоугольника).

### Ориентация (angle 0, по часовой в Phaser)

- `shore_straight`: трава **сверху**
- `shore_corner`: трава **верх-лево**

| mask | angle |
|------|-------|
| N / E / S / W | 0 / 90 / 180 / 270 |
| N\|W / N\|E / S\|E / S\|W | 0 / 90 / 180 / 270 |

**Не ломай** соглашения ориентации без обновления `public/assets/game/README.md`.

## Персонаж

Файл: `src/game/entities/Player.ts`.

- Ввод: стрелки + WASD.
- Движение с нормализацией диагонали; ось X/Y раздельно проверяются на коллизии.
- Коллизии: 4 угла хитбокса → `world.isWalkableAt` (чанк on-demand) → трава без дерева.
- Bounce: каждый кадр `scaleY` осциллирует (`SQUASH_MIN`…`SQUASH_MAX`), `scaleX = 2 - scaleY` (лёгкий squash/stretch). Работает всегда, не только при ходьбе.

## Ассеты

Curated runtime-файлы:

| Файл | Роль |
|------|------|
| `grass.png` | interior land |
| `water.png` | interior water |
| `shore_straight.png` | прямой берег |
| `shore_corner.png` | угол (суша огибает воду) |
| `stone.png` | непроходимый камень |
| `stone_shards.png` | осколки на траве (walkable) |
| `tree_small.png` | маленькое дерево (overlay) |
| `flower_*.png` | цветы (walkable overlay) |
| `grass_tuft_*.png` | пучки травы (walkable overlay) |
| `player.png` | игрок |

Ключи Phaser: `src/game/assets.ts` → `AssetKey` + `ASSET_URLS`.

## Конфигурация Phaser

`src/game/config.ts`:

- `pixelArt: true` (без размытия при апскейле 16→`tileSize`);
- `Scale.RESIZE` на весь viewport;
- сцена: только `WorldScene`.

## Env-переменные

| Переменная | Default | Смысл |
|------------|---------|--------|
| `VITE_WORLD_WIDTH` | 48 | ширина в тайлах |
| `VITE_WORLD_HEIGHT` | 32 | высота в тайлах |
| `VITE_TILE_SIZE` | 32 | размер клетки на экране (px) |
| `VITE_WORLD_SEED` | 42 | сид генерации |
| `VITE_PLAYER_SPEED` | 160 | px/s |
| `VITE_DEV_PORT` | 5173 | dev-сервер |
| `VITE_PREVIEW_PORT` | 4173 | preview |

## Правила для агентов при изменениях

1. **Не смешивать** логический тип тайла и текстуру берега — берег только в `terrainVisual`.
2. Новые типы местности: сначала `TileType` + генерация + walkability, потом визуал/автотайл.
3. Новые текстуры: файл в `public/assets/game/`, ключ в `assets.ts`, при необходимости кадр в `game/README.md`.
4. Не тащить React в игровой цикл Phaser; UI — отдельно, когда понадобится.
5. Не коммитить секреты; `.env` в gitignore, образец — `.env.example`.
6. Язык общения с пользователем репозитория — **русский**; код/идентификаторы — английский.
7. Перед крупным рефактором — уточнить у пользователя (особенно смена генерации/автотайла).
8. После правок TS — `npm run build` (включает `tsc --noEmit`).

## Что ещё не сделано (не выдумывать как существующее)

- сохранение/загрузка мира;
- Go API;
- чанки / бесконечный мир;
- здания, ресурсы, NPC;
- анимация ходьбы спрайтами (сейчас только squash bounce);
- меню / пауза.

## Карта файлов (быстрый поиск)

| Задача | Файл |
|--------|------|
| Точка входа | `src/main.ts` |
| Конфиг Phaser | `src/game/config.ts` |
| Сцена мира | `src/game/scenes/WorldScene.ts` |
| Генерация чанка | `src/game/world/generateChunk.ts` |
| Чанковый мир | `src/game/world/chunkWorld.ts` |
| Автотайлинг берега | `src/game/world/terrainVisual.ts` |
| Типы тайлов | `src/game/world/tiles.ts` |
| Игрок | `src/game/entities/Player.ts` |
| Ключи ассетов | `src/game/assets.ts` |
| Env | `src/game/env.ts` |
| Человеческий README | `README.md` |
