# FluentFox API

Production-ready Go REST API for the FluentFox Japanese learning platform.

## Stack

| Concern | Choice |
|---|---|
| Language | Go 1.23+ |
| Router | chi/v5 |
| Database | PostgreSQL (Neon) |
| DB Driver | pgx/v5 |
| Migrations | golang-migrate |
| SRS | go-fsrs/v3 |
| Auth | JWT dual-token (access + refresh) |
| Passwords | bcrypt |
| Logging | uber/zap |
| Validation | go-playground/validator/v10 |
| Config | godotenv |
| Email | Resend API |
| Storage | Cloudflare R2 (S3-compatible) |
| Deployment | Railway + Neon |
| Dev server | air (hot reload) |

## First Run

```bash
# 1. Clone and enter the project
git clone <repo-url> fluentfox-api
cd fluentfox-api

# 2. Copy the example env file and fill in values
cp .env.example .env.development
# Edit .env.development — at minimum set DATABASE_URL, JWT secrets

# 3. Start the dev stack (Postgres + API with hot reload)
make dev

# 4. In another terminal, apply all migrations
make migration-up

# 5. Seed the database with characters, kanji, and XP config
make seed

# 6. (Optional) Sync MDX content into the database
make sync-content

# 7. Hit the health check
curl http://localhost:8080/health
# → {"status":"ok","env":"development"}
```

## Common Commands

```bash
make dev            # Start dev stack with hot reload
make dev-down       # Stop containers (preserves DB data)
make test           # Run all tests against isolated test DB
make lint           # Run golangci-lint
make tidy           # go mod tidy + verify
make migration-up     # Apply pending migrations
make migration-downdown   # Roll back one migration
make migration-new name=add_something  # Create new migration pair
make seed           # Run seed SQL files
make sync-content   # Parse MDX and upsert to DB
make build          # Build production Docker image
```

## Project Structure

```
cmd/
  api/              HTTP server entry point
  sync-content/     MDX → PostgreSQL content sync CLI

internal/
  auth/             JWT auth, registration, password reset
  user/             User profile, settings, password change
  chapter/          Grammar chapters and concepts
  srs/              Spaced repetition (go-fsrs)
  quiz/             Quiz sessions and answers
  progress/         Chapter and vocabulary progress tracking
  streak/           Daily streak tracking
  xp/               XP, levelling, leaderboard
  shop/             In-app shop (streak freezes, etc.)
  admin/            Admin dashboard and moderation
  middleware/        Auth, CORS, logging, rate limiting, RBAC
  mdxparser/        MDX file parser (used by sync-content only)

pkg/
  cache/            In-memory content cache
  database/         pgxpool setup and transaction helper
  mailer/           Email interface + Resend implementation
  storage/          Storage interface + Cloudflare R2 implementation
  response/         Standard JSON envelope helpers
  token/            JWT generation/validation + SHA-256 hash
  validator/        go-playground/validator wrapper

config/             Typed config struct + env loading
db/
  migrations/       Ordered SQL migration files (000001–000036)
  seeds/            Idempotent seed data (characters, kanji, XP config)
content/
  grammar/          MDX chapter files by JLPT level
docker/
  Dockerfile.dev    Hot-reload dev image
  Dockerfile.prod   Multi-stage production image (<30MB)
```

## Architecture

Every domain in `internal/` follows the three-layer pattern:

- **handler.go** — HTTP only. Parses request, validates, calls service, writes response.
- **service.go** — Business logic only. Orchestrates repository calls.
- **repository.go** — SQL only. Reads/writes plain Go types.
- **model.go** — Three separate struct sets: DB model, request DTO, response DTO.

Dependencies flow downward: handler → service → repository. Nothing reaches up.

## Environment

See `.env.example` for all required variables with descriptions.

- `APP_ENV=development` loads `.env.development`
- `APP_ENV=test` loads `.env.test`
- `APP_ENV=production` reads from the real environment (Railway injects these)

## Deployment

Railway detects `railway.json` and builds with `docker/Dockerfile.prod`.
The `/health` endpoint checks DB connectivity — Railway uses it to confirm deployment.
