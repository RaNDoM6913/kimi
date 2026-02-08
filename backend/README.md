# Telegram Dating Mini App Backend (RB)

Backend service for Telegram Dating Mini App.

## Quick start

1. Copy environment template:
   ```bash
   cp .env.example .env
   ```
2. Start local dependencies:
   ```bash
   docker compose -f docker/docker-compose.yml up -d
   ```
3. Run API:
   ```bash
   make run-api
   ```
4. Health check:
   ```bash
   curl http://localhost:8080/healthz
   ```

## Commands

- `make fmt` - format code
- `make test` - run tests
- `make run-api` - run HTTP API server
- `make run-bot` - run moderation bot stub
- `make migrate-up` - apply SQL migrations
- `make migrate-down` - rollback SQL migrations
