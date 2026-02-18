# Admin Login Backend (`telegram -> 2fa -> password`)

Отдельный сервис для входа в админку по этапам:
1. Валидация Telegram auth payload.
2. Проверка TOTP-кода (Google Authenticator).
3. Проверка пароля и выпуск JWT.

## Что уже заложено

- Telegram auth для сайта (Telegram Login Widget) и WebApp payload.
- Челлендж-сессия входа (`admin_login_challenges`) с TTL.
- TOTP setup с QR-кодом (`admin_totp_setup_tokens`), start/confirm.
- Проверка пароля по `bcrypt`.
- Шифрование `totp_secret` в БД (AES-GCM, ключ из `LOGIN_TOTP_SECRET_KEY`).
- Блокировка аккаунта после N неудачных попыток.
- Серверные сессии (`admin_sessions`):
  - логаут при неактивности (`LOGIN_SESSION_IDLE_TIMEOUT`, default `30m`),
  - максимальная жизнь сессии (`LOGIN_SESSION_MAX_LIFETIME`, default `12h`).
- JWT access token с `sid` (привязан к серверной сессии).

## Быстрый запуск

```bash
cd /Users/ivankudzin/cursor/tgapp/adminpanel/backend/login
cp .env.example .env
export $(grep -v '^#' .env | xargs)

go mod tidy
go run ./cmd/login-api
```

Сервер по умолчанию поднимется на `:8082`.

Ключ для шифрования 2FA секрета (пример генерации):

```bash
openssl rand -base64 32
```

## Миграции

Файлы:
- `migrations/000001_admin_login.up.sql`
- `migrations/000001_admin_login.down.sql`

Применить можно любым вашим инструментом миграций (goose/migrate/ручной SQL).

## Интеграция с основным backend (`/admin/*`)

Чтобы JWT из этого сервиса принимался в `backend`:
- `backend` должен быть с миграцией `backend/migrations/000012_add_admin_web_auth.up.sql`;
- `backend` переменная `ADMIN_WEB_JWT_SECRET` должна совпадать с `LOGIN_JWT_SECRET`;
- `ADMIN_WEB_SESSION_IDLE_TIMEOUT` и `LOGIN_SESSION_IDLE_TIMEOUT` лучше держать одинаковыми;
- запросы в `backend /admin/*` идут с `Authorization: Bearer <access_token>`.

`backend` дополнительно валидирует `sid` через таблицу `admin_sessions`, поэтому logout/idle/max-lifetime отрабатывают централизованно на уровне БД.

## Добавление пользователя вручную

1. Сгенерировать hash пароля:

```bash
go run ./cmd/password-hash -password 'YourStrongPassword'
```

2. Вставить пользователя в БД:

```sql
INSERT INTO admin_users (telegram_id, username, display_name, role, password_hash, totp_enabled, is_active)
VALUES (123456789, 'admin_user', 'Main Admin', 'owner', '$2a$10$...', FALSE, TRUE);
```

## API

### 1) Telegram step

`POST /v1/auth/telegram/start`

```json
{
  "init_data": "id=...&username=...&auth_date=...&hash=..."
}
```

`init_data` для сайта можно передавать как querystring из Telegram Login Widget.

Успех:

```json
{
  "challenge_id": "uuid",
  "next_step": "2fa",
  "username": "admin_user"
}
```

### 2) 2FA step

`POST /v1/auth/2fa/verify`

```json
{
  "challenge_id": "uuid",
  "code": "123456"
}
```

### 3) Password step

`POST /v1/auth/password/verify`

```json
{
  "challenge_id": "uuid",
  "password": "YourStrongPassword"
}
```

Успех:

```json
{
  "access_token": "...",
  "token_type": "Bearer",
  "expires_at": "2026-02-17T12:34:56Z",
  "admin": {
    "id": 1,
    "telegram_id": 123456789,
    "username": "admin_user",
    "display_name": "Main Admin",
    "role": "owner"
  }
}
```

### Проверка текущей сессии (touch idle timeout)

`GET /v1/auth/me` с `Authorization: Bearer ...`

### Logout

`POST /v1/auth/logout` с `Authorization: Bearer ...`

### TOTP setup (QR для Google Authenticator)

Эндпойнты защищены заголовком `X-Bootstrap-Key` (`LOGIN_BOOTSTRAP_KEY`).

`POST /v1/admin/2fa/setup/start`

```json
{
  "telegram_id": 123456789,
  "account_name": "admin_user"
}
```

Успех:

```json
{
  "setup_id": "uuid",
  "telegram_id": 123456789,
  "otpauth_url": "otpauth://totp/...",
  "secret": "BASE32SECRET",
  "qr_code_data_url": "data:image/png;base64,...",
  "expires_at": "2026-02-17T12:34:56Z"
}
```

`POST /v1/admin/2fa/setup/confirm`

```json
{
  "setup_id": "uuid",
  "code": "123456"
}
```

## Параметры, которые удобно менять

- `LOGIN_SESSION_IDLE_TIMEOUT` (`30m`) — авто-логаут при неактивности.
- `LOGIN_SESSION_MAX_LIFETIME` (`12h`) — максимум жизни сессии.
- `LOGIN_JWT_TTL` (`12h`) — TTL access token (если выше max lifetime, будет ограничен max lifetime).
- `LOGIN_TOTP_SECRET_KEY` — ключ шифрования `totp_secret` (32 байта: plain/base64/hex).
- `LOGIN_MAX_FAILED_ATTEMPTS` (`5`) — блокировка после N ошибок.
- `LOGIN_LOCK_DURATION` (`15m`) — длительность блокировки.
- `LOGIN_TELEGRAM_AUTH_MAX_AGE` (`5m`) — максимальный возраст Telegram auth payload.

## Dev-режим

`LOGIN_DEV_MODE=true` позволяет передавать в `init_data` просто Telegram ID (`"123456789"`), чтобы быстрее тестировать поток до подключения реального Telegram auth payload.
