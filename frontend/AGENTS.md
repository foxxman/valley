# AGENTS.md — контекст для LLM / AI-агентов

Документ для языковых моделей: архитектура, ключевые алгоритмы и правила работы с этим фронтендом.
Человекочитаемый быстрый старт — в `README.md`.

## Назначение проекта

Браузерный **top-down 2D** прототип settlement/exploration игры.

Сейчас реализовано:

- бесконечный чанковый мир (генерация по запросу, 16×16);
- пруды (4-связный blob) + автотайл берега (straight / corner / inner / water);
- каменные гряды (ядро 4–10 камней + обводка осколков 1–2);
- маленькие деревья на траве (~7.5%, непроходимы);
- цветы и пучки травы (~7.5% каждый, проходимы);
- ходьба персонажа (WASD / стрелки; деревья и камень непроходимы);
- камера всегда держит персонажа в центре;
- idle bounce (squash/stretch по вертикали);
- мини-карта 150×150 (правый верх, цвета тайлов, игрок в центре).

Пока **нет**: сохранения на диск, комнат, чатов, друзей, боя, NPC, строительства.
Вход и лобби — React (`src/main.tsx`). Карта — Phaser внутри маршрута `/play`. Бэкенд: регистрация и сессия в куках.
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
  └─ src/main.tsx
       └─ React: /login | /lobby | /play
            └─ /play → Phaser.Game(createGameConfig)
            └─ WorldScene
                 ├─ preload()     — текстуры
                 ├─ create()      — ChunkWorld + spawn chunk + Player + follow camera
                 └─ update()      — player; ensure/draw chunks; cull; minimap

Логика мира (без Phaser):
  world/tiles.ts          — TileType, walkability
  world/chunkWorld.ts     — бесконечные чанки + snapshot API
  world/generate-chunk/   — пруды + деревья/flora внутри одного чанка
  world/flora.ts          — виды цветов/травы + маппинг на AssetKey
  world/terrainVisual.ts  — water/straight/corner

UI:
  ui/Minimap.ts           — HUD-мини-карта (Graphics, scrollFactor 0)

Сущности:
  entities/Player.ts      — ввод, коллизии (isWalkableAt), bounce
```

### Важные разделения

1. **Логическая карта** ≠ **визуальный тайл**.  
   В чанке `grass | water | stone | stone-shards` + слои `trees` / `flora`. Берег — в `resolveTerrainVisual`.

2. **Данные чанка** живут рядом с игроком. `releaseFarChunks` забывает сгенерированные чанки; тот же сид собирает их снова. Чанки из `fromSnapshot` закреплены и не выбрасываются.  
   **Картинка чанка** — один `RenderTexture`. Далеко от камеры он уничтожается и запекается заново из tiles/trees/flora.

3. **Phaser** — цикл, камера, ввод, отрисовка. Генерация — чистые функции.

4. Предпочтение в TS: **объекты-словари вместо `switch`**.

## Поток (рантайм)

1. Старт: `ChunkWorld(seed)`, генерируется чанк `(0,0)` и соседи, спавн в центре `(0,0)`.
2. Камера: `startFollow(..., 1, 1)`, **без** `setBounds`.
3. Игрок идёт → `isWalkableAt` / `getTileAt` создаёт чанк при первом касании.
4. `update`: `ensureChunksAround` + запекание чанка в текстуру; далеко — destroy текстуры и `releaseFarChunks`.
5. Будущий сейв: `world.toSnapshot()` (tiles + trees + flora) → JSON/сервер; загрузка — `ChunkWorld.fromSnapshot`.

## Алгоритм генерации чанка

Файл: `src/game/world/generate-chunk/` (вызывается из `ChunkWorld.ensureChunk`).

1. Чанк 16×16 (или `VITE_CHUNK_SIZE`), вся трава.
2. Пруд — с шансом ~40% один 4-связный blob, **inset ≥1** от края; тонкие места срезаются, чтобы у воды был валидный берег.
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

Трава всегда `grass.png`. Канал и тупик отдельных текстур не имеют. Внутренний угол — `shore_corner_inner`, если ортогональной суши нет, но есть диагональная.

Вода граничит с травой только через клетку берега (периметр прямоугольника).

### Ориентация (angle 0, по часовой в Phaser)

- `shore_straight`: трава **сверху**
- `shore_corner`: трава **верх-лево**

| mask | angle |
|------|-------|
| N / E / S / W | 0 / 90 / 180 / 270 |
| N\|W / N\|E / S\|E / S\|W | 0 / 90 / 180 / 270 |

**Не ломай** соглашения ориентации без обновления `public/assets/README.md`.

## Персонаж

Файл: `src/game/entities/Player.ts`.

- Ввод: стрелки + WASD.
- Движение с нормализацией диагонали; ось X/Y раздельно проверяются на коллизии.
- Коллизии: 4 угла хитбокса → `world.isWalkableAt` (чанк on-demand) → трава без дерева.
- Bounce: каждый кадр `scaleY` осциллирует (`SQUASH_MIN`…`SQUASH_MAX`), `scaleX = 2 - scaleY` (лёгкий squash/stretch). Работает всегда, не только при ходьбе.

## Ассеты

Curated runtime-файлы (`public/assets/`):

| Файл | Роль |
|------|------|
| `soil/grass.png` | interior land |
| `water/water.png` | interior water |
| `water/shore_straight.png` | прямой берег |
| `water/shore_corner.png` | угол (суша огибает воду) |
| `water/shore_corner_inner.png` | внутренний угол / кусочек суши на воде |
| `stones/stone.png` | непроходимый камень |
| `stones/stone_shards.png` | осколки на траве (walkable) |
| `vegetation/tree_small.png` | маленькое дерево (overlay) |
| `vegetation/flower_*.png` | цветы (walkable overlay) |
| `vegetation/lily.png` | кувшинки на воде (walkable overlay) |
| `vegetation/grass_tuft_1.png` | пучок травы (walkable overlay) |
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
| `VITE_CHUNK_SIZE` | 16 | сторона чанка в тайлах |
| `VITE_TILE_SIZE` | 32 | размер клетки на экране (px) |
| `VITE_WORLD_SEED` | 42 | сид генерации |
| `VITE_PLAYER_SPEED` | 160 | px/s |
| `VITE_DEV_PORT` | 5173 | dev-сервер |
| `VITE_PREVIEW_PORT` | 4173 | preview |

## Правила для агентов при изменениях

1. **Не смешивать** логический тип тайла и текстуру берега — берег только в `terrainVisual`.
2. Новые типы местности: сначала `TileType` + генерация + walkability, потом визуал/автотайл.
3. Новые текстуры: файл в `public/assets/{soil|water|stones|vegetation}/`, ключ в `assets.ts`, описание в `public/assets/README.md`.
4. Не тащить React в игровой цикл Phaser; UI — отдельно, когда понадобится.
5. Не коммитить секреты; `.env` в gitignore, образец — `.env.example`.
6. Язык общения с пользователем репозитория — **русский**; код/идентификаторы — английский.
7. Перед крупным рефактором — уточнить у пользователя (особенно смена генерации/автотайла).
8. После правок TS — `npm run build` (включает `tsc --noEmit`).

## Что ещё не сделано (не выдумывать как существующее)

- сохранение/загрузка мира;
- игровые ручки бэкенда (сейчас только healthcheck);
- здания, ресурсы, NPC;
- анимация ходьбы спрайтами (сейчас только squash bounce);
- меню / пауза.

## Карта файлов (быстрый поиск)

| Задача | Файл |
|--------|------|
| Точка входа | `src/main.tsx` |
| Конфиг Phaser | `src/game/config.ts` |
| Сцена мира | `src/game/scenes/WorldScene.ts` |
| Генерация чанка | `src/game/world/generate-chunk/` |
| Чанковый мир | `src/game/world/chunkWorld.ts` |
| Автотайлинг берега | `src/game/world/terrainVisual.ts` |
| Типы тайлов | `src/game/world/tiles.ts` |
| Игрок | `src/game/entities/Player.ts` |
| Мини-карта | `src/game/ui/Minimap.ts` |
| Ключи ассетов | `src/game/assets.ts` |
| Env | `src/game/env.ts` |
| Человеческий README | `README.md` |
