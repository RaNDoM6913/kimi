# TGApp — Короткий README

Монорепозиторий Telegram Dating Mini App.

Состоит из ключевых частей:
- `backend/` — Go API + встроенный bot-процесс
- `tgbots/bot_moderator/` — отдельный Telegram moderation bot
- `tgbots/bot_support/` — отдельный Telegram support bot
- `frontend/` — клиентский интерфейс (сейчас в статусе прототипа с mock API)
- `adminpanel/backend/login/` — backend логина админки (`telegram -> 2fa -> password`)

## Что уже есть

- Авторизация (JWT access + refresh sessions)
- Профиль, лента, свайпы, квоты, anti-abuse механики
- Базовые платежные потоки и идемпотентность транзакций
- Админ-эндпоинты метрик/anti-abuse
- Защита `/admin/*` через web JWT + серверные admin-сессии
- Отдельный login backend для админки: Telegram auth, TOTP (Google Authenticator + QR), пароль, блокировки
- Модерационный бот с ролями `OWNER/ADMIN/MODERATOR/NONE`

## Что в работе

- Полная готовность `frontend/` (интеграция с реальным backend API)
- Полная интеграция `adminpanel/frontend` c `adminpanel/backend/login`

## Быстрый запуск локально

## 1) Backend

```bash
cd backend
docker compose -f docker/docker-compose.yml up -d
make migrate-up
make run-api
```

Опционально запустить встроенный backend bot:

```bash
make run-bot
```

## 2) Moderator bot

```bash
cd tgbots/bot_moderator
docker compose up -d
make run
```

## 2.1) Support bot

```bash
cd tgbots/bot_support
cp .env.example .env
make run
```

## 3) Frontend

```bash
cd frontend
npm i
npm run dev
```

## 4) Admin login backend

```bash
cd adminpanel/backend/login
cp .env.example .env
go mod tidy
go run ./cmd/login-api
```

## Минимально важные ENV

Backend:
- `POSTGRES_DSN`
- `REDIS_ADDR`
- `JWT_SECRET`
- `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`
- `ADMIN_BOT_TOKEN` (для `/admin/bot/*`)
- `ADMIN_WEB_JWT_SECRET` (для `/admin/*`, должен совпадать с `LOGIN_JWT_SECRET`)
- `ADMIN_WEB_SESSION_IDLE_TIMEOUT` (idle timeout для admin web сессий)

Admin login backend (`adminpanel/backend/login`):
- `LOGIN_POSTGRES_DSN`
- `LOGIN_JWT_SECRET`
- `LOGIN_TOTP_SECRET_KEY`
- `LOGIN_SESSION_IDLE_TIMEOUT` (по умолчанию `30m`)
- `LOGIN_SESSION_MAX_LIFETIME` (по умолчанию `12h`)
- `LOGIN_TELEGRAM_BOT_TOKEN`
- `LOGIN_BOOTSTRAP_KEY`

Moderator bot:
- `BOT_TOKEN`
- `OWNER_TG_ID`
- `DATABASE_URL`
- `ADMIN_API_URL`, `ADMIN_BOT_TOKEN`, `ADMIN_MODE`

## Полезные команды проверки

Backend:

```bash
cd backend
go list ./...
go test ./...
go vet ./...
gofmt -l .
```

Moderator bot:

```bash
cd tgbots/bot_moderator
go list ./...
go test ./...
go vet ./...
gofmt -l .
```

## Где смотреть детали

- Полная документация: `README_FULL.md`
- Backend API: `backend/docs/openapi.yaml`
- Backend архитектурные решения: `backend/docs/decisions/`
- Login backend админки: `adminpanel/backend/login/README.md`
