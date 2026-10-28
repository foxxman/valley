# AGENTS.md

Карта репозитория для агентов. Открывай строку из таблицы и работай в ней. Не обходи дерево проекта и не запускай субагента на разведку, если вопрос уже есть ниже.

Подробный фронтенд (генерация, берег, игрок, инвентарь, сбор) — `frontend/AGENTS.md`. Роуты, схема БД и транзакции — `backend/README.md`. Запуск — корневой `README.md`.

## Что это

Монорепозиторий: `frontend/` (Vite, React для входа и HUD, Phaser для мира) и `backend/` (Go, `net/http`, PostgreSQL). Миграции применяются при старте сервера.

Карта собирается на клиенте из `worldSeed` игры. Сервер сид не пересчитывает. В Postgres лежит только журнал изменённых клеток `map_cells`. Координаты живых игроков — в памяти `backend/internal/presence`, в базу не пишутся.

## Куда смотреть

| Вопрос | Куда идти |
| --- | --- |
| Пруд, камни, деревья, цветы, спавн | `frontend/src/game/world/generate-chunk/` (`pond.ts`, `stoneRidge.ts`, `decor.ts`, `spawn.ts`). Алгоритм уже расписан в `frontend/AGENTS.md` |
| Угол берега, стык воды и суши | `frontend/src/game/world/terrainVisual.ts`. Берег в журнал не пишется |
| Тип тайла и проходимость | `frontend/src/game/world/tiles.ts` |
| Чанк, сид, журнал поверх генерации | `frontend/src/game/world/chunkWorld.ts` |
| Клавиши, полоска прочности, запекание чанка | `frontend/src/game/scenes/WorldScene.ts`. Это склейка сцены, не правила движения и не правила удара |
| Ходьба, взгляд, коллизии | `frontend/src/game/entities/Player.ts` |
| Чужой спрайт, в том числе в воде | `frontend/src/game/entities/RemoteAvatar.ts`, `frontend/src/game/entities/swim.ts` |
| Мини-карта | `frontend/src/game/ui/Minimap.ts` |
| Ключи текстур мира | `frontend/src/game/assets.ts` |
| Иконка предмета в рюкзаке | `frontend/src/game/inventory/icons.ts` |
| Модалка инвентаря | `frontend/src/game/inventory/` |
| HTTP журнала, сбора и удара на клиенте | `frontend/src/game/map/api.ts` |
| Какая клетка под клавишей E | `frontend/src/game/world/gatherTarget.ts` |
| WebSocket, чужие координаты, конверт `map` | `frontend/src/game/net/channel.ts`, `backend/internal/realtime/`, `backend/internal/presence/` |
| Сборка процесса | `backend/cmd/server/main.go` |
| Роуты | `backend/internal/httpapi/` — `auth.go`, `games.go`, `inventory.go`, `map.go`, `realtime.go`, `swagger.go` |
| Сбор цветка и удар | `backend/internal/gather/nodes.go`, `backend/internal/gather/service.go` |
| Журнал клеток | `backend/internal/mapstate/` |
| Рюкзак и выдача предмета | `backend/internal/inventory/` |
| Миграции `001`…`007` | `backend/internal/db/migrations/` |
| Конфиг сервера | `backend/internal/config/config.go`, корневой `.env.example` |
| Swagger | `GET /swagger/index.html`, спека в `backend/docs/` |

Новая картинка: файл в `frontend/public/assets/{soil,water,stones,vegetation,items}/`, ключ в `assets.ts`, описание в `frontend/public/assets/README.md`. Иконка, которой нет на карте, ещё и в `inventory/icons.ts`. Исходные паки Kenney лежат вне репозитория, в `/Users/maxim/work/game_images/`.

## Числа, которые не нужно заново искать

Источник правды по удару — `backend/internal/gather/nodes.go`. Клиентская полоска читает полную прочность из `frontend/src/game/world/nodes.ts`. Если числа разойдутся, верен сервер.

- Урон голой руки — 1. Инструментов нет.
- Дерево: 10 ударов, дроп 4 `wood`, слой `tree` очищается.
- Сплошной камень: 20 ударов, дроп 5 `stone`, `terrain` становится `grass`. Осколки не добываются.
- Цветы `flower-0`…`flower-3` собираются одним запросом, без прочности. Трава и кувшинки не собираются.
- Дистанция действия: клетка игрока или ортогональный сосед. Тайл 32 px.
- Рюкзак: 20 ячеек, 5 в ряд, свой у пары игрок + игра.
- Чанк 16×16. Вход в мир: `GET /api/games/{id}/map`, затем `GET /api/games/{id}/inventory`.

## Чего в игре нет

Не описывай как готовое: инструменты, использование предметов, постройки, снос прудов, чат, друзей, бой, NPC. Слой `feature` в журнале есть и на карту не рисуется.
