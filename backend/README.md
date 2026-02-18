# Telegram Dating Mini App Backend

Основной Go backend для Telegram Dating Mini App.

## Quick start

1. Подготовить env:
   ```bash
   cp .env.example .env
   ```
2. Поднять локальные зависимости:
   ```bash
   docker compose -f docker/docker-compose.yml up -d
   ```
3. Применить миграции:
   ```bash
   make migrate-up
   ```
4. Запустить API:
   ```bash
   make run-api
   ```
5. Проверить здоровье:
   ```bash
   curl http://localhost:8080/healthz
   ```

## Admin web auth (`/admin/*`)

`/admin/*` теперь защищены отдельной web-auth схемой для админки:
- access token должен быть `Bearer` JWT с claims `uid`, `sid`, `role`;
- `sid` валидируется в таблице `admin_sessions` (серверная сессия);
- при каждом запросе сессия `touch`-ится и продлевается по idle timeout.

Что нужно включить:
- применить миграцию `backend/migrations/000012_add_admin_web_auth.up.sql`;
- задать `ADMIN_WEB_JWT_SECRET` (должен совпадать с `LOGIN_JWT_SECRET` из `adminpanel/backend/login`);
- при необходимости настроить `ADMIN_WEB_SESSION_IDLE_TIMEOUT` (по умолчанию `30m`).

Если `ADMIN_WEB_JWT_SECRET` пустой, `/admin/*` будут отвечать ошибкой `ADMIN_AUTH_UNAVAILABLE`.

## Важные ENV

- `POSTGRES_DSN`
- `REDIS_ADDR`
- `JWT_SECRET`
- `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`
- `ADMIN_BOT_TOKEN` (для `/admin/bot/*`)
- `ADMIN_WEB_JWT_SECRET` (для `/admin/*`)
- `ADMIN_WEB_SESSION_IDLE_TIMEOUT`

## Commands

- `make fmt` - format code
- `make test` - run tests
- `make run-api` - run HTTP API server
- `make run-bot` - run built-in bot process
- `make migrate-up` - apply SQL migrations
- `make migrate-down` - rollback SQL migrations
