# Game

Монорепозиторий: браузерный клиент (`frontend/`) и Go-бэкенд (`backend/`) с PostgreSQL.

## Dev: бэкенд + БД

Нужны Docker и Docker Compose. Конфиг — `.env` (создаётся из `.env.example` при первом запуске).

| Команда | Что делает |
|---------|------------|
| `bash scripts/dev-up.sh` | Собрать и поднять контейнеры, следить за логами backend |
| `bash scripts/dev-restart.sh` | Перезапустить уже поднятые контейнеры |
| `bash scripts/dev-down.sh` | Остановить и удалить контейнеры (данные Postgres сохраняются) |
| `bash scripts/dev-drop.sh` | То же + удалить volumes (полный сброс БД и кэшей) |

В дев-режиме используется `Dockerfile.dev` + [Air](https://github.com/air-verse/air): правки в `backend/` подхватываются без ручной пересборки образа.

Порты по умолчанию: backend `8080`, Postgres `5432`.

## Frontend

```bash
cd frontend
npm install
npm run dev
```

Подробности — в `frontend/README.md`.
