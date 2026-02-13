# TGApp — Короткий README

Монорепозиторий Telegram Dating Mini App.

Состоит из трех основных частей:
- `backend/` — Go API + встроенный bot-процесс
- `tgbots/bot_moderator/` — отдельный Telegram moderation bot
- `frontend/` — клиентский интерфейс (сейчас в статусе прототипа с mock API)

## Что уже есть

- Авторизация (JWT access + refresh sessions)
- Профиль, лента, свайпы, квоты, anti-abuse механики
- Базовые платежные потоки и идемпотентность транзакций
- Админ-эндпоинты метрик/anti-abuse
- Модерационный бот с ролями `OWNER/ADMIN/MODERATOR/NONE`

## Что в работе

- Полная готовность `frontend/` (интеграция с реальным backend API)
- Полная готовность админки

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

## 3) Frontend

```bash
cd frontend
npm i
npm run dev
```

## Минимально важные ENV

Backend:
- `POSTGRES_DSN`
- `REDIS_ADDR`
- `JWT_SECRET`
- `S3_ENDPOINT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_BUCKET`
- `ADMIN_BOT_TOKEN` (для `/admin/bot/*`)

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

