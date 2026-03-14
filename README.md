# The Deranged Hermits — Website

MTG club website for The Deranged Hermits, hosting proxy-friendly Legacy and Premodern tournaments in Athens, Greece.

## Quick Start

### Prerequisites
- Docker & Docker Compose
- (or) Go 1.23+ and PostgreSQL 16+

### Run with Docker Compose

```bash
docker compose up --build
```

The site will be available at **http://localhost:8080**.

### Create an Admin User

```bash
# With Docker:
docker compose exec app ./seed admin your-password

# Without Docker:
go run ./cmd/seed admin your-password
```

### Run Locally (without Docker)

1. Start PostgreSQL and create a database:
   ```bash
   createdb hermits
   psql hermits < migrations/001_init.sql
   ```

2. Set environment variables:
   ```bash
   export DATABASE_URL="postgres://localhost:5432/hermits?sslmode=disable"
   export SESSION_SECRET="some-secret"
   export API_KEY="your-api-key"
   ```

3. Run the server:
   ```bash
   go run ./cmd/server
   ```

## Configuration

| Variable | Description | Default |
|---|---|---|
| `DATABASE_URL` | PostgreSQL connection string | `postgres://hermits:hermits@localhost:5432/hermits?sslmode=disable` |
| `PORT` | HTTP listen port | `8080` |
| `SESSION_SECRET` | Secret for session cookies | `change-me-in-production` |
| `API_KEY` | Bearer token for `/api` routes | (empty — API disabled) |
| `BASE_URL` | Public URL of the site | `http://localhost:8080` |
| `SMTP_HOST` | SMTP server hostname | (empty — email disabled) |
| `SMTP_PORT` | SMTP server port | `587` |
| `SMTP_USER` | SMTP username | |
| `SMTP_PASS` | SMTP password | |
| `SMTP_FROM` | From address for emails | `noreply@derangedhermits.com` |

## JSON API

All `/api` endpoints require `Authorization: Bearer <API_KEY>` header.

```bash
# List upcoming events
curl -H "Authorization: Bearer your-api-key" http://localhost:8080/api/events?upcoming=true

# Create an event
curl -X POST http://localhost:8080/api/events \
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

## Project Structure

```
cmd/server/main.go          — HTTP server entrypoint
cmd/seed/main.go            — CLI to create admin users
internal/config/             — Environment-based configuration
internal/db/                 — Database queries (events, subscribers, auth)
internal/handlers/           — HTTP handlers (home, events, subscribe, admin, API)
internal/mail/               — SMTP email sending
internal/middleware/          — Auth middleware (sessions + API key)
migrations/                  — SQL schema
templates/                   — Go HTML templates
static/                      — CSS, images
```

## Hero Image

Place the hero image at `static/art.png` for the homepage hero background.

## License

Copyright (C) 2026 Dylan Stephano-Shachter

This program is free software: you can redistribute it and/or modify it under the terms of the GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

See [LICENSE.txt](LICENSE.txt) for the full license text.
