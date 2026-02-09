# bot_moderator

Внутренний Telegram-бот модерации для Telegram Dating Mini App.

## Что умеет сейчас
- запуск `go run ./cmd/bot`
- команда `/start`
- меню по роли: `OWNER` / `ADMIN` / `MODERATOR` / `NONE`
- пока роль `OWNER` определяется только через `OWNER_TG_ID` из конфига
- остальные пользователи получают `NONE`

## Быстрый старт
1. Скопировать переменные окружения из `.env.example`.
2. Указать `BOT_TOKEN` и `OWNER_TG_ID`.
3. Запустить:

```bash
make run
```

## Проверка

```bash
make test
```
