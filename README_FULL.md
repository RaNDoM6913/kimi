# TGApp — Полный README

Полный технический README по монорепозиторию проекта Telegram Dating Mini App.

## 1. Назначение проекта

Проект реализует backend и ботовую часть для дейтингового Mini App в Telegram, а также фронтенд-клиент.

Ключевые цели:
- поддержка пользовательского API (профили, лента, свайпы, матчи, квоты)
- модерация анкет через Telegram-бота
- anti-abuse и продуктовые метрики
- платежные сценарии и управление entitlements

## 2. Состав репозитория

- `backend/` — основной Go backend
- `tgbots/bot_moderator/` — отдельный moderation bot
- `frontend/` — клиентское приложение (React/Vite)
- `adminpanel/` — заготовка/отдельный фронтовый контур (в текущем состоянии не основной источник правды для бэкенд-логики)

## 3. Go-модули

В репозитории обнаружены 2 Go-модуля:

- `backend/go.mod` (`go 1.23.0`)
- `tgbots/bot_moderator/go.mod` (`go 1.23.0`)

`go.work` отсутствует.

## 4. Архитектура по модулям

## 4.1 `backend/`

Точки входа:
- `backend/cmd/api/main.go`
- `backend/cmd/bot/main.go`

Слои:
- `internal/transport/http` — handlers, DTO, API ошибки
- `internal/services` — бизнес-логика
- `internal/repo/postgres` — PostgreSQL репозитории
- `internal/repo/redis` — Redis репозитории
- `internal/app/apiapp` — сборка HTTP приложения и роутинг
- `internal/app/botapp` — встроенный backend Telegram bot

Интеграции:
- Postgres
- Redis
- S3/MinIO
- Telegram Bot API

Особенности:
- degraded mode: backend может стартовать без части зависимостей (например, Postgres/S3) с предупреждениями в логах
- API контракт описан в `backend/docs/openapi.yaml`
- архитектурные решения зафиксированы в `backend/docs/decisions/`

## 4.2 `tgbots/bot_moderator/`

Точка входа:
- `tgbots/bot_moderator/cmd/bot/main.go`

Ключевой runtime:
- `internal/app/app.go` — инициализация сервисов
- `internal/app/router.go` — обработка update/callback + state machine

Сервисы:
- `services/access` — роли/ACL
- `services/moderation` — очередь/approve/reject
- `services/lookup` — поиск пользователя и действия
- `services/bans` — бан/разбан
- `services/audit` — audit trail
- `services/stats` — отчеты day/week/month/all
- `services/system` — toggles и users count

Репозитории:
- `repo/postgres` — прямой доступ к БД
- `repo/adminhttp` — работа через backend Admin API
- `repo/dualrepo` — fallback стратегия между HTTP и DB

Интеграции:
- Telegram Bot API
- Postgres
- Redis
- Backend Admin API (`/admin/bot/*`)
- S3 presign (для медиа URL)

## 4.3 `frontend/`

Точка входа:
- `frontend/src/main.tsx`

Ключевые файлы:
- `src/app/router.tsx` — маршрутизация страниц
- `src/pages/*` — экраны
- `src/app/services/mockPublicApi.ts` — mock API слой

Текущий статус:
- фронт в основном демонстрационный
- привязка к Telegram WebApp SDK (initData bridge) пока не интегрирована
- есть UI-overlay/blur элементы, но полноценный privacy-consent flow не оформлен как отдельный продуктовый контур

## 5. Потоки данных (упрощенно)

Пользовательский поток:
1. `frontend` -> `backend` API
2. `backend` -> services -> repos (Postgres/Redis/S3)
3. результат возвращается в фронтенд

Модерационный поток:
1. модератор работает через `tgbots/bot_moderator`
2. бот дергает backend Admin API и/или работает напрямую с БД (dual mode)
3. модерационные решения фиксируются в audit/stats

## 6. Основные API поверхности

Backend публичные и v1-алиасы:
- Auth: `/auth/telegram`, `/auth/refresh`, `/auth/logout`, `/auth/logout_all`
- Feed/Swipe/Likes/Matches: `/v1/feed`, `/v1/swipes`, `/v1/likes`, `/v1/matches`, и т.д.
- Payments: `/purchase/create`, `/purchase/webhook`, `/entitlements` (+ v1 алиасы)

Admin:
- `/admin/*` — health/private/metrics/anti-abuse
- `/admin/bot/*` — endpoint-ы для moderation bot

Примечание по интеграции:
- в текущем коде `tgbots` ожидает более широкий набор `/admin/bot/*`, чем реально зарегистрирован в `backend/internal/app/apiapp/routes.go`
- из-за этого важен корректный `ADMIN_MODE` (`db/http/dual`) и fallback стратегия

## 7. Данные и хранилища

Postgres:
- пользователи, профили, moderation items, bans
- платежи и транзакции
- daily_metrics и события
- bot-таблицы (`bot_audit`, `bot_moderation_actions`, `bot_lookup_actions`, и т.д.)

Redis:
- session/refresh storage
- rate limiting и anti-abuse состояния
- anti-abuse dashboard counters/top offenders

S3/MinIO:
- медиа-объекты (фото/кружки)

## 8. Конфигурация и ENV

## 8.1 Backend (`backend/.env.example`)

Обязательные/критичные:
- `POSTGRES_DSN`
- `REDIS_ADDR`
- `JWT_SECRET`
- `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`
- `ADMIN_BOT_TOKEN` (если используется admin bot API)

Также:
- `APP_CONFIG`, `APP_ENV`
- `HTTP_*` таймауты
- `JWT_ACCESS_TTL`, `REFRESH_TTL`
- `BOT_TOKEN` (для встроенного botapp)

## 8.2 Moderator bot (`tgbots/bot_moderator/.env.example`)

Ключевые:
- `BOT_TOKEN`
- `OWNER_TG_ID`
- `DATABASE_URL`
- `ADMIN_API_URL`
- `ADMIN_BOT_TOKEN`
- `ADMIN_MODE` (`db`, `http`, `dual`)
- `ADMIN_HTTP_TIMEOUT_SECONDS`

## 9. Запуск локально

## 9.1 Backend

```bash
cd backend
docker compose -f docker/docker-compose.yml up -d
make migrate-up
make run-api
```

Опционально:

```bash
make run-bot
```

## 9.2 tgbots/bot_moderator

```bash
cd tgbots/bot_moderator
docker compose up -d
make run
```

## 9.3 Frontend

```bash
cd frontend
npm i
npm run dev
```

## 10. Миграции

Backend:
- SQL: `backend/migrations/`
- запуск: `make migrate-up` / `make migrate-down`
- runner: `backend/scripts/migrate.sh` (goose)

Moderator bot:
- SQL: `tgbots/bot_moderator/internal/repo/postgres/migrations/`
- отдельный стандартный runner в модуле не выделен (запускать выбранным инструментом миграций)

## 11. Проверки качества

Backend:

```bash
cd backend
go list ./...
go test ./...
go vet ./...
gofmt -l .
```

Bot moderator:

```bash
cd tgbots/bot_moderator
go list ./...
go test ./...
go vet ./...
gofmt -l .
```

## 12. Текущий статус и ограничения

Ключевые ограничения текущего состояния:
- фронтенд пока в mock-режиме, без полноценной интеграции с backend и Telegram WebApp SDK
- админский UI-контур не завершен
- интеграционный контракт `/admin/bot/*` между `backend` и `tgbots` требует финальной синхронизации

Технические пункты, которые обычно закрываются до production:
- строгая криптографическая проверка Telegram initData
- подпись/anti-replay защита payment webhook
- единая и явная политика degraded mode/fail-fast для production

## 13. Roadmap (до этапа “фронт + админка готово”)

1. Завершить frontend интеграцию с реальным API backend.
2. Завершить админский интерфейс и UX для операционных сценариев.
3. Зафиксировать единый контракт `backend <-> tgbots` по `/admin/bot/*`.
4. После финализации UI:
   - закрыть оставшиеся security/reliability задачи,
   - провести полный e2e прогон,
   - собрать release checklist.

## 14. Быстрые ссылки

- Backend README: `backend/README.md`
- Bot moderator README: `tgbots/bot_moderator/README.md`
- Frontend README: `frontend/README.md`
- OpenAPI: `backend/docs/openapi.yaml`
- ADR: `backend/docs/decisions/`

