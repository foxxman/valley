# Backend

HTTP API игры на Go и PostgreSQL. Роутер — `net/http.ServeMux`. Миграции применяются при старте процесса.

## Архитектура

`cmd/server` собирает процесс: читает конфиг, открывает пул соединений, прогоняет миграции и регистрирует HTTP-хендлеры. Бизнес-правила живут в сервисах, SQL — в репозиториях. Хендлеры зависят от интерфейсов, не от конкретных репозиториев.

```mermaid
flowchart TD
  server["cmd/server"] --> httpapi["internal/httpapi"]
  server --> health["internal/health"]
  httpapi --> auth["internal/auth"]
  httpapi --> game["internal/game"]
  httpapi --> inventory["internal/inventory"]
  httpapi --> realtime["internal/realtime"]
  httpapi --> presence["internal/presence"]
  presence --> realtime
  presence --> game
  inventory --> game
  auth --> player["internal/player"]
  auth --> db[(PostgreSQL)]
  game --> db
  inventory --> db
  player --> db
  health --> db
```

| Пакет | Роль |
| --- | --- |
| `cmd/server` | Точка входа и сборка зависимостей |
| `internal/config` | Переменные окружения |
| `internal/logger` | `slog` в stdout |
| `internal/db` | Пул pgx и встроенные SQL-миграции |
| `internal/httpapi` | Роуты, cookie, JSON-ошибки, Swagger UI |
| `internal/health` | `GET /healthcheck` |
| `internal/auth` | Регистрация, вход, JWT access, refresh-токены |
| `internal/player` | Игрок и таблица `players` |
| `internal/game` | Игры владельца и таблица `games` |
| `internal/inventory` | Каталог предметов и рюкзак пары игрок + игра |
| `internal/realtime` | Канал сообщений: соединения, комнаты, рассылка конвертов |
| `internal/presence` | Кто сейчас в игре и где стоит. Координаты только в памяти процесса |
| `docs` | Сгенерированная OpenAPI-спека |

Запрос идёт так: хендлер читает JSON и cookie, сервис проверяет правила, репозиторий пишет в Postgres. Ошибки сервиса хендлер переводит в код и JSON вида `{ "error": { "code", "message" } }`.

Сессия состоит из двух HttpOnly cookie. `access_token` — JWT с коротким сроком, путь `/`. `refresh_token` хранится в базе только как хеш, путь `/api/auth`. При обновлении сессии старый refresh удаляется и выпускается новая пара.

`GET /api/ws` — общий канал живых сообщений. Сейчас через него видно, кто находится в игре, и передаются координаты. Каталог событий в эту документацию не входит. Игра, сид и вход в мир остаются обычными HTTP-запросами: достаточно знать id игры. Координаты в Postgres не пишутся и пропадают вместе с соединением.

При создании игры можно передать `worldSeed` — целое число от 0 до 4 294 967 295. Если поле пустое, сервер выбирает сид сам и сохраняет его в игре. Одинаковый сид собирает одну и ту же карту. Уже существующие игры при миграции получают сид `42`, как раньше на клиенте.

Инвентарь принадлежит паре игрок + игра. `GET /api/games/{id}/inventory` возвращает рюкзак текущего игрока: `capacity` (сейчас 20) и `slots`. В списке только занятые ячейки, пустые строки в базе не хранятся. У ячейки есть `index` от 0 до 19, `quantity` и предмет (`id`, `slug`, `name`, `iconKey`, `description`, `maxStack`). Чужой рюкзак не отдаётся. Если игры нет, ответ `game_not_found`. Эндпоинта, которым клиент мог бы выдать себе предмет, нет.

Роуты:

| Метод | Путь | Доступ |
| --- | --- | --- |
| `GET` | `/healthcheck` | открытый |
| `POST` | `/api/auth/register`, `/api/auth/login` | открытые |
| `POST` | `/api/auth/refresh`, `/api/auth/logout` | `refresh_token` |
| `GET` | `/api/auth/me` | `access_token` |
| `POST`, `GET` | `/api/games` | `access_token` |
| `GET`, `PATCH` | `/api/games/{id}` | `access_token` |
| `GET` | `/api/games/{id}/inventory` | `access_token`, только свой рюкзак |
| `GET` | `/api/ws` | `access_token`, переход на WebSocket |
| `GET` | `/swagger/index.html` | открытый |

## Схема БД

Файлы лежат в `internal/db/migrations`. Имя файла — `001_name.sql`. При старте ещё не применённые версии выполняются по порядку, номер пишется в `schema_migrations`.

`players` — учётная запись.

| Колонка | Тип | Ограничения |
| --- | --- | --- |
| `id` | `UUID` | PK, `gen_random_uuid()` |
| `login` | `TEXT` | `^[a-zA-Z0-9_]{3,24}$`, уникален без учёта регистра |
| `password_hash` | `TEXT` | bcrypt |
| `created_at` | `TIMESTAMPTZ` | `now()` |

`refresh_tokens` — хеши refresh-токенов. Удаляются вместе с игроком.

| Колонка | Тип | Ограничения |
| --- | --- | --- |
| `id` | `UUID` | PK, `gen_random_uuid()` |
| `player_id` | `UUID` | FK → `players.id` `ON DELETE CASCADE` |
| `token_hash` | `TEXT` | уникальный |
| `expires_at` | `TIMESTAMPTZ` | |
| `created_at` | `TIMESTAMPTZ` | `now()` |

`games` — игры игрока. Удаляются вместе с владельцем.

| Колонка | Тип | Ограничения |
| --- | --- | --- |
| `id` | `UUID` | PK, `gen_random_uuid()` |
| `name` | `TEXT` | длина 1–40 |
| `owner_id` | `UUID` | FK → `players.id` `ON DELETE CASCADE` |
| `world_seed` | `BIGINT` | от 0 до 4 294 967 295 |
| `created_at` | `TIMESTAMPTZ` | `now()` |
| `updated_at` | `TIMESTAMPTZ` | `now()` |

`items` — каталог определений предметов. В миграции таблица пустая.

| Колонка | Тип | Ограничения |
| --- | --- | --- |
| `id` | `UUID` | PK, `gen_random_uuid()` |
| `slug` | `TEXT` | уникальный стабильный код |
| `name` | `TEXT` | подпись в инвентаре |
| `icon_key` | `TEXT` | ключ картинки на клиенте |
| `description` | `TEXT` | текст тултипа, по умолчанию пустая строка |
| `max_stack` | `INT` | от 1 до 999, по умолчанию 1 |
| `created_at` | `TIMESTAMPTZ` | `now()` |

`inventory_slots` — занятые ячейки рюкзака. Удаляются вместе с игрой и игроком. Пока предмет лежит в ячейке, строку каталога удалить нельзя.

| Колонка | Тип | Ограничения |
| --- | --- | --- |
| `game_id` | `UUID` | FK → `games.id` `ON DELETE CASCADE`, часть PK |
| `player_id` | `UUID` | FK → `players.id` `ON DELETE CASCADE`, часть PK |
| `slot_index` | `SMALLINT` | 0–19, часть PK |
| `item_id` | `UUID` | FK → `items.id` |
| `quantity` | `INT` | не меньше 1 |

Первичный ключ `(game_id, player_id, slot_index)` и есть индекс выборки рюкзака. Отдельного индекса на эти колонки нет.

`schema_migrations` создаётся кодом, не SQL-файлом: `version TEXT` PK и `applied_at TIMESTAMPTZ`.

## Запуск

Обычный способ — из корня репозитория. Нужны Docker и Docker Compose. Если `.env` нет, `dev-up.sh` копирует его из `.env.example`.

```bash
bash scripts/dev-up.sh
```

В dev-режиме контейнер backend монтирует `backend/` и перезапускает процесс через Air после сохранения файлов. Postgres слушает `5432`, API — `8080`.

| Команда | Что делает |
| --- | --- |
| `bash scripts/dev-up.sh` | Собрать контейнеры и показать логи backend |
| `bash scripts/dev-restart.sh` | Перезапустить уже поднятые контейнеры |
| `bash scripts/dev-down.sh` | Остановить контейнеры, данные Postgres оставить |
| `bash scripts/dev-drop.sh` | Остановить контейнеры и удалить volumes |

Без Docker, когда Postgres уже доступен:

```bash
cd backend
export DATABASE_URL=postgres://game:game@localhost:5432/game?sslmode=disable
export JWT_SECRET=dev-only-secret-change-me-32bytes
go run ./cmd/server
```

Тесты:

```bash
cd backend
go test ./...
```

Переменные окружения:

| Переменная | По умолчанию | Смысл |
| --- | --- | --- |
| `DATABASE_URL` | обязательна | Строка подключения Postgres |
| `JWT_SECRET` | обязательна, минимум 32 символа | Секрет подписи access JWT |
| `HTTP_ADDR` | `:8080` | Адрес HTTP-сервера |
| `ACCESS_TTL` | `15m` | Срок `access_token` |
| `REFRESH_TTL` | `720h` | Срок `refresh_token` |
| `COOKIE_SECURE` | `false` | Флаг `Secure` у cookie |
| `LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error` |
| `LOG_FORMAT` | text | `json` переключает JSON-логи |

В Docker `DATABASE_URL` собирается из `POSTGRES_USER`, `POSTGRES_PASSWORD` и `POSTGRES_DB`.

## Swagger

Спека собирается из комментариев над хендлерами, а не из списка `HandleFunc`. После правки комментария, `json`-тегов или модели:

```bash
cd backend
go generate ./cmd/server
```

Команда обновляет `docs/docs.go`, `docs/swagger.json` и `docs/swagger.yaml`. Эти файлы компилируются в сервер, поэтому после генерации процесс нужно перезапустить. В dev-контейнере Air сделает это сам, если `docs/` смонтирован вместе с остальным кодом.

Страница: `http://localhost:8080/swagger/index.html`. **Try it out** ходит в этот же API. Сначала вызовите `login` или `register`: cookie HttpOnly, браузер приложит их к следующим запросам со страницы.
