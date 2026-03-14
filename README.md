# The Deranged Hermits — Website

MTG club website for The Deranged Hermits, hosting proxy-friendly Legacy and Premodern tournaments in Athens, Greece.

## Quick Start

### Prerequisites
- Docker & Docker Compose

### Run with Docker Compose

```bash
docker compose up --build
```

The site will be available at **http://localhost** (Caddy reverse proxy on ports 80/443).

### Create an Admin User

```bash
docker compose exec app ./seed admin your-password
```

### Configuration

Copy `.env.example` to `.env` and edit as needed:

```bash
cp .env.example .env
```

| Variable | Description | Default |
|---|---|---|
| `DOMAIN` | Domain for Caddy HTTPS | `localhost` |
| `DATABASE_URL` | PostgreSQL connection string | `postgres://hermits:hermits@db:5432/hermits?sslmode=disable` |
| `PORT` | Internal HTTP listen port | `8080` |
| `SESSION_SECRET` | Secret for session signing & CSRF | `change-me-in-production` |
| `API_KEY` | Bearer token for `/api` routes | (empty — API disabled) |
| `BASE_URL` | Public URL of the site (used in emails) | `http://localhost` |
| `SMTP_HOST` | SMTP server hostname | (empty — email disabled) |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USER` | SMTP username | |
| `SMTP_PASS` | SMTP password | |
| `SMTP_FROM` | From address for emails | `noreply@derangedhermits.com` |

For production, generate secrets with `openssl rand -hex 32`.

## Features

- **Event management** — create, edit, delete events from the admin panel
- **Email subscriptions** — double opt-in confirmation email flow
- **Event notifications** — notify confirmed subscribers about new events with per-subscriber unsubscribe tokens
- **iCal downloads** — add events to your calendar
- **JSON API** — programmatic access with Bearer token auth
- **CSRF protection** — double-submit cookie pattern on all browser forms
- **Rate limiting** — per-IP limits on login, subscribe, and API routes
- **Signed sessions** — HMAC-SHA256 signed session cookies
- **Caddy reverse proxy** — automatic HTTPS with Let's Encrypt in production
- **Health check** — `GET /health` endpoint
- **Graceful shutdown** — drains connections on SIGTERM
- **Structured logging** — JSON logs via `log/slog`

## JSON API

All `/api` endpoints require `Authorization: Bearer <API_KEY>` header.

```bash
# List upcoming events
curl -H "Authorization: Bearer your-api-key" http://localhost/api/events?upcoming=true

# Create an event
curl -X POST http://localhost/api/events \
  -H "Authorization: Bearer your-api-key" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Legacy Tournament",
    "format": "Legacy",
    "date": "2026-04-05T16:00:00+03:00",
    "location": "Dragonphoenix Inn",
    "location_url": "https://maps.google.com/...",
    "entry_fee": "10€",
    "description": "Proxy-friendly Legacy tournament. All proxies must be color prints."
  }'
```

## Testing

### Run all tests locally

```bash
./scripts/test.sh
```

This starts a temporary PostgreSQL container on port 5433, runs the full test suite, and tears it down.

### Run unit tests only (no database required)

```bash
go test ./internal/middleware/... ./internal/config/... ./internal/mail/...
```

### Run with an existing database

```bash
export TEST_DATABASE_URL="postgres://hermits_test:hermits_test@localhost:5433/hermits_test?sslmode=disable"
docker compose -f docker-compose.test.yml up -d db
go test -v -count=1 -p 1 ./...
docker compose -f docker-compose.test.yml down
```

Tests also run automatically on pull requests via GitHub Actions.

## Project Structure

```
cmd/server/main.go           — HTTP server entrypoint
cmd/seed/main.go             — CLI to create admin users
internal/config/              — Environment-based configuration
internal/db/                  — Database queries (events, subscribers, auth)
internal/handlers/            — HTTP handlers (home, events, subscribe, admin, API)
internal/mail/                — SMTP email sending
internal/middleware/           — CSRF, rate limiting, auth, session signing
migrations/                   — SQL schema & migrations
templates/                    — Go HTML templates
static/                       — CSS, images
Caddyfile                     — Caddy reverse proxy config
```

## Hero Image

Place the hero image at `static/art.png` for the homepage hero background. Recommended resolution: 1920×640px.

## License

Copyright (C) 2026 Dylan Stephano-Shachter

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

See [LICENSE.txt](LICENSE.txt) for the full license text.
