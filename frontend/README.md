# Settlement Prototype (Frontend)

Браузерный прототип top-down игры на **Vite + TypeScript + Phaser 3**.
Сейчас: бесконечный чанковый мир (прямоугольные пруды, берег), персонаж и камера по центру.
Меню и сервер пока не подключены.

Для AI/LLM: подробный разбор архитектуры и алгоритмов — в [`AGENTS.md`](./AGENTS.md).

## Требования

- Node.js 20+
- npm 10+

## Установка

```bash
cd frontend
cp .env.example .env
npm install
```

## Скрипты

| Команда | Режим | Описание |
|---------|--------|----------|
| `npm run dev` | development | Dev-сервер Vite с HMR. По умолчанию: http://localhost:5173 |
| `npm run build` | production | Проверка TypeScript и сборка в `dist/` |
| `npm run preview` | production preview | Локальный просмотр production-сборки. По умолчанию: http://localhost:4173 |

### Примеры

```bash
# разработка
npm run dev

# production-сборка
npm run build

# проверка собранного билда
npm run preview
```

## Управление

- **W / A / S / D** или **стрелки** — перемещение
- Спрайт персонажа — Kenney Roguelike Characters (лёгкий bounce по вертикали)
- Трава / вода / берег (прямой, углы, канал, тупик) — Kenney + custom

## Ассеты

Лежат в `public/assets/` (раздаются статикой Vite):

```text
public/assets/
  game/                         # выбранные текстуры для игры
  kenney_roguelike-rpg-pack/    # трава / вода
  kenney_roguelike-characters/  # персонаж
  kenney_tiny-town/
  kenney_tiny-farm/
```

Лицензия паков Kenney: **CC0**. Источники выбранных кадров — в [`public/assets/README.md`](./public/assets/README.md).

## Переменные окружения

Скопируйте `.env.example` → `.env` и при необходимости измените значения.
Все клиентские переменные должны начинаться с `VITE_` (требование Vite).

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `VITE_DEV_PORT` | `5173` | Порт dev-сервера (`npm run dev`) |
| `VITE_PREVIEW_PORT` | `4173` | Порт preview (`npm run preview`) |
| `VITE_WORLD_WIDTH` | `48` | Ширина мира в тайлах |
| `VITE_WORLD_HEIGHT` | `32` | Высота мира в тайлах |
| `VITE_TILE_SIZE` | `32` | Размер тайла на экране (исходные PNG — 16×16, масштабируются) |
| `VITE_WORLD_SEED` | `42` | Сид генерации мира (одинаковый сид → одинаковая карта) |
| `VITE_PLAYER_SPEED` | `160` | Скорость персонажа (пиксели в секунду) |

После изменения `.env` перезапустите `npm run dev`.

## Структура

```text
frontend/
  public/assets/            # Kenney-паки и curated-текстуры
  src/
    main.ts
    game/
      assets.ts             # Ключи и URL текстур
      config.ts
      env.ts
      scenes/WorldScene.ts
      entities/Player.ts
      world/
        tiles.ts
        chunkWorld.ts        # чанки + snapshot (заготовка сейва)
        generateChunk.ts     # пруды в чанке
        terrainVisual.ts     # water / straight / corner
  .env.example
  index.html
  vite.config.ts
```

## Дальше

- сохранение мира
- Go-бэкенд
- UI на React
