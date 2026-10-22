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

# 6. Course content (kana, kanji) is loaded by `make migrate` from content/

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
make build          # Build production Docker image
```

## Project Structure

```
cmd/
  api/              HTTP server entry point

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
- `RESEND_API_KEY` (required) sends all email through the Resend HTTP API.
  `EMAIL_FROM_ADDRESS` must be on a domain verified in Resend. There is no
  SMTP option: Railway and many other hosts block outbound SMTP.
- `ALERT_EMAIL_TO` (optional) gets an email when a request fails with a 500,
  with the error, request id and stack trace. At most one email is sent every
  5 minutes; the next one says how many errors were skipped. Empty turns it off.
  The email is sent in the background, so it is reliable on a long-running host
  (Railway) but may be lost on Vercel, which can stop the function after the response.
- `VPNAPI_API_KEY` (optional) is a vpnapi.io key (free: 1,000 lookups a day). At login the API looks up
  the client's city and whether it is a VPN; a password login from a VPN must
  then also enter a code sent by email. Empty turns both off. The client IP is
  only right when `TRUSTED_PROXIES` covers the host's proxy (Railway:
  `100.64.0.0/10`). On Vercel, the API reads the platform-controlled
  `X-Vercel-Forwarded-For` header when the automatically exposed `VERCEL=1`
  system variable is available; enable **Automatically expose System
  Environment Variables** in the Vercel project if it is not already on.
- `R2_ACCOUNT_ID`, `R2_ACCESS_KEY`, `R2_SECRET_KEY`, `R2_BUCKET_NAME` and
  `R2_PUBLIC_URL` (all optional, but all five together) turn on profile photo
  uploads to Cloudflare R2. Without them `PUT /profile/avatar` answers 503 and
  the built-in avatars still work.
- `STROKES_BASE_URL` (optional) is the public base URL of the kana and kanji
  stroke JSON files in R2. The `/content/*` endpoints return it as
  `strokes_base_url`; it is empty when unset.
- `YOUTUBE_API_KEY` (optional) is a YouTube Data API v3 key used only by the
  classes admin sync (`POST /admin/classes/channels/{id}/sync` and
  `.../courses/{id}/sync-playlist`). Without it those answer 503
  `YOUTUBE_SYNC_UNAVAILABLE`; the public `/classes` pages never call YouTube.
  Channel avatar and banner uploads use the same R2 settings as profile photos.
- `DATABASE_URL` may end with `default_query_exec_mode=simple_protocol`
  (e.g. `...?sslmode=require&default_query_exec_mode=simple_protocol`). pgx
  then skips preparing each new query on a fresh connection, saving one round
  trip per query on serverless cold starts, and it works through Neon's
  `-pooler` (PgBouncer) host. pgx still escapes every parameter. Try it on a
  preview deployment first.

## Migrations

The schema is managed by [goose](https://github.com/pressly/goose). SQL files
live in `migrations/` (`000001_users.sql`, `000002_...`) and are embedded in the
`cmd/migrate` binary. `000001` recreates the user tables exactly as GORM
AutoMigrate used to build them, with `IF NOT EXISTS`, so it is safe on existing
databases. Add a change as a new numbered file with `-- +goose Up` and
`-- +goose Down` sections; never edit an applied file.

- `make migrate` applies pending migrations (`go run ./cmd/migrate up`)
- `make migrate-status` lists applied and pending migrations
- `make migrate-down` rolls back the latest migration
- Any other goose command works too, e.g. `go run ./cmd/migrate down-to 0`

`cmd/migrate` reads `DATABASE_URL`, `DB_MAX_CONNS` and `DB_MIN_CONNS`.

`refresh_tokens.user_id` has a unique index that the login upsert relies on.
Run the migration against a database **before** deploying code that includes
it: the migration first removes duplicate refresh-token rows (keeping each
user's newest), then creates the index. Deploying first makes every login fail
until the migration runs.

- Development: `make migrate`
- Production: runs automatically on every push to `main`, in
  `.github/workflows/deploy-vercel.yml`, before `vercel deploy`. It reads the
  `PROD_DATABASE_URL` repository secret (Neon direct, non-pooler URL). A failed
  migration stops the deploy.

## Deployment

Railway detects `railway.json` and builds with `docker/Dockerfile.prod`.
The `/health` endpoint checks DB connectivity — Railway uses it to confirm deployment.
