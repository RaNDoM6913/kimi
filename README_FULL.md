# TGApp — Полный README

Полный технический README по монорепозиторию Telegram Dating Mini App.

## 1. Назначение проекта

Проект покрывает:
- пользовательский API для Mini App;
- модерацию и поддержку через Telegram-ботов;
- админский контур (admin panel + безопасный login backend);
- платежные и anti-abuse сценарии.

## 2. Состав репозитория

- `backend/` — основной Go backend (публичный API + `/admin/*` + `/admin/bot/*`)
- `tgbots/bot_moderator/` — moderation bot
- `tgbots/bot_support/` — support bot
- `frontend/` — клиентский фронтенд продукта
- `adminpanel/frontend/` — фронтенд админки
- `adminpanel/backend/login/` — backend логина админки (`telegram -> 2fa -> password`)

## 3. Go-модули

В репозитории 4 Go-модуля:
- `backend/go.mod`
- `tgbots/bot_moderator/go.mod`
- `tgbots/bot_support/go.mod`
- `adminpanel/backend/login/go.mod`

`go.work` отсутствует.

## 4. Архитектура по модулям

## 4.1 `backend/`

Точки входа:
- `backend/cmd/api/main.go`
- `backend/cmd/bot/main.go`

Ключевые слои:
- `internal/transport/http` — handlers/DTO/errors
- `internal/services` — бизнес-логика
- `internal/repo/postgres` и `internal/repo/redis` — доступ к данным
- `internal/app/apiapp` — сборка приложения и роутинг

Ключевое по админке:
- `/admin/*` защищены `AdminWebAuthMiddleware`;
- access token валидируется по JWT и `sid`;
- `sid` проверяется в `admin_sessions` с touch по idle timeout;
- роли берутся из `admin_users.role`.

## 4.2 `tgbots/bot_moderator/`

Отдельный бот модерации с ACL, очередями, аудитом и fallback-режимами работы через backend API и/или БД.

## 4.3 `tgbots/bot_support/`

Отдельный бот саппорта:
- принимает входящие сообщения пользователей;
- отправляет их в backend (`/admin/bot/support/incoming`);
- забирает outbox и отправляет ответы пользователям.

## 4.4 `adminpanel/backend/login/`

Отдельный backend авторизации админки:
- шаг 1: Telegram auth payload;
- шаг 2: обязательный TOTP (Google Authenticator);
- шаг 3: пароль (`bcrypt`);
- выпуск JWT access token с `sid`;
- серверные сессии с idle timeout и max lifetime;
- генерация QR (otpauth + PNG data URL) для привязки 2FA.

## 4.5 Frontend-контуры

- `frontend/` — пользовательский интерфейс;
- `adminpanel/frontend/` — интерфейс админки (часть экранов уже интегрирована с backend, login flow пока требует финальной API-связки с `adminpanel/backend/login`).

## 5. Потоки данных (упрощенно)

Пользовательский поток:
1. `frontend` -> `backend` (`/v1/*`).
2. `backend` -> services -> repos (Postgres/Redis/S3).
3. Ответ возвращается в клиент.

Модерация/саппорт:
1. `tgbots/*` -> `backend /admin/bot/*`.
2. backend сохраняет/читает состояние в Postgres.

Вход в админку:
1. `adminpanel/frontend` -> `adminpanel/backend/login`.
2. login backend создает запись в `admin_sessions`, выдает JWT с `sid`.
3. `adminpanel/frontend` ходит в `backend /admin/*` с тем же JWT.
4. backend валидирует JWT + активную сессию в БД.

## 6. Основные API поверхности

Основной backend:
- user auth: `/auth/telegram`, `/auth/refresh`, `/auth/logout`, `/auth/logout_all` (+ `/v1/auth/*`)
- user domain: `/v1/feed`, `/v1/swipes`, `/v1/likes`, `/v1/matches`, `/v1/purchase/*`, ...
- admin web: `/admin/health`, `/admin/users/{id}/private`, `/admin/metrics/daily`, `/admin/antiabuse/*`
- admin bot: `/admin/bot/mod/*`, `/admin/bot/lookup/*`, `/admin/bot/users/*`, `/admin/bot/support/*`

Login backend (`adminpanel/backend/login`):
- `POST /v1/auth/telegram/start`
- `POST /v1/auth/2fa/verify`
- `POST /v1/auth/password/verify`
- `GET /v1/auth/me`
- `POST /v1/auth/logout`
- `POST /v1/admin/2fa/setup/start` (bootstrap key)
- `POST /v1/admin/2fa/setup/confirm` (bootstrap key)

## 7. Данные и хранилища

Postgres:
- core доменные таблицы (users/profiles/feed/swipes/matches/payments/metrics);
- bot-таблицы для модерации/саппорта;
- admin auth таблицы: `admin_users`, `admin_login_challenges`, `admin_totp_setup_tokens`, `admin_sessions`.

Redis:
- пользовательские refresh/session данные;
- rate limit и anti-abuse состояние;
- anti-abuse dashboard counters.

S3/MinIO:
- медиа-объекты.

## 8. Конфигурация и ENV

## 8.1 Backend (`backend/.env.example`)

Критично:
- `POSTGRES_DSN`
- `REDIS_ADDR`
- `JWT_SECRET`
- `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`
- `ADMIN_BOT_TOKEN`
- `ADMIN_WEB_JWT_SECRET`
- `ADMIN_WEB_SESSION_IDLE_TIMEOUT`

Важно:
- `ADMIN_WEB_JWT_SECRET` должен совпадать с `LOGIN_JWT_SECRET` из login backend.

## 8.2 Login backend (`adminpanel/backend/login/.env.example`)

Критично:
- `LOGIN_POSTGRES_DSN`
- `LOGIN_JWT_SECRET`
- `LOGIN_TOTP_SECRET_KEY`
- `LOGIN_TELEGRAM_BOT_TOKEN`
- `LOGIN_BOOTSTRAP_KEY`

Политика сессий:
- `LOGIN_SESSION_IDLE_TIMEOUT` (default `30m`)
- `LOGIN_SESSION_MAX_LIFETIME` (default `12h`)
- `LOGIN_JWT_TTL` (обычно `<= LOGIN_SESSION_MAX_LIFETIME`)

Безопасность:
- `LOGIN_MAX_FAILED_ATTEMPTS`
- `LOGIN_LOCK_DURATION`
- `LOGIN_TELEGRAM_AUTH_MAX_AGE`

## 8.3 Moderator bot (`tgbots/bot_moderator/.env.example`)

- `BOT_TOKEN`
- `OWNER_TG_ID`
- `DATABASE_URL`
- `ADMIN_API_URL`
- `ADMIN_BOT_TOKEN`
- `ADMIN_MODE` (`db`, `http`, `dual`)

## 8.4 Support bot (`tgbots/bot_support/.env.example`)

- `BOT_TOKEN`
- `ADMIN_API_URL`
- `ADMIN_BOT_TOKEN`

## 9. Запуск локально

## 9.1 Backend

```bash
cd backend
cp .env.example .env
docker compose -f docker/docker-compose.yml up -d
make migrate-up
make run-api
```

## 9.2 Moderator bot

```bash
cd tgbots/bot_moderator
cp .env.example .env
docker compose up -d
make run
```

## 9.3 Support bot

```bash
cd tgbots/bot_support
cp .env.example .env
make run
```

## 9.4 Admin login backend

```bash
cd adminpanel/backend/login
cp .env.example .env
go mod tidy
go run ./cmd/login-api
```

## 9.5 Frontend

```bash
cd frontend
npm i
npm run dev
```

```bash
cd adminpanel/frontend
npm i
npm run dev
```

## 10. Миграции

Backend:
- SQL: `backend/migrations/`
- запуск: `make migrate-up` / `make migrate-down`
- admin web auth добавлено в `000012_add_admin_web_auth.*.sql`

Login backend:
- SQL: `adminpanel/backend/login/migrations/`
- ключевая миграция: `000001_admin_login.*.sql`

## 11. Проверки качества

Backend:

```bash
cd backend
go test ./...
go vet ./...
gofmt -l .
```

Moderator bot:

```bash
cd tgbots/bot_moderator
go test ./...
go vet ./...
gofmt -l .
```

Support bot:

```bash
cd tgbots/bot_support
go test ./...
go vet ./...
gofmt -l .
```

Login backend:

```bash
cd adminpanel/backend/login
go test ./...
go vet ./...
gofmt -l .
```

## 12. Текущий статус и ограничения

- backend-side admin auth (JWT + server sessions + idle timeout) уже внедрен;
- login backend для админки внедрен, включая TOTP setup с QR;
- `adminpanel/frontend` login экран пока требует полной API-интеграции;
- контракт `/admin/bot/*` и UI-сценарии админки продолжают расширяться.

## 13. Roadmap

1. Довести API-интеграцию `adminpanel/frontend <-> adminpanel/backend/login`.
2. Закрыть оставшиеся admin UI потоки (role-aware UX, ошибки, logout UX, refresh token стратегия при необходимости).
3. Провести e2e по admin security-сценариям (lockout, idle logout, max lifetime, 2FA bootstrap).
4. Дофинализировать продуктовый frontend и общий release checklist.

## 14. Быстрые ссылки

- `README_SHORT.md`
- `backend/README.md`
- `adminpanel/backend/login/README.md`
- `adminpanel/frontend/README.md`
- `tgbots/bot_moderator/README.md`
- `tgbots/bot_support/README.md`
- `backend/docs/openapi.yaml`
- `backend/docs/decisions/`
