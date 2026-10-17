# Learning system: implementation progress

Plan: `docs/learning-plan.md`. This file says what is done and what remains, so work can
resume from here at any time.

Branches: backend `claude/task-gllti3` (FluentFox), frontend `claude/task-gllti3` (FluentFoxUI).

## Status

| Phase | Part | Status |
|---|---|---|
| 1 | goose, `000001_users` baseline, `cmd/migrate` switch, Makefile | done (000001 schema identical to AutoMigrate; up/down/up verified on Postgres 16) |
| 1 | migrations `000002`–`000008` (all tables from the plan) | done |
| 2 | backend content: `content/*.json`, `cmd/sync-content`, `internal/content`, `/content/*` | in progress |
| 2 | frontend content services on `/content/*` | done (FluentFoxUI `b333f6d`; falls back to bundled JSON until the backend serves content) |
| 2 | upload stroke JSON to R2, set `strokes_base_url` | not started |
| 3 | `pkg/srs` (go-fsrs v4 wrapper), `internal/learn/grading.go`, learning enums + constants | in progress |
| 4 | frontend `useAnswerTimer`, answer queue, `learnService` / `progressService` clients, auth guards | done (FluentFoxUI `b333f6d`; not yet used by any screen) |
| 4 | backend sessions: settings, question builder, lessons, reviews, answers, complete | not started |
| 4 | frontend lesson / review / result screens, remove `ts-fsrs` | not started |
| 5 | practice modes, safety net, unit / skip / final tests, unit progress, notes | not started |
| 6 | progress APIs (overview, forecast, calendar, weakness, confusions, items, dashboard) | not started |
| 6 | frontend progress pages, dashboard, settings | not started |
| — | tests for each finished part (written after the code) | not started |

## Notes and follow-ups

- Existing user tables keep `user_id` as `text` with no FKs (that is what AutoMigrate built).
  New tables use `uuid` FKs to `users(id)`. Aligning the old columns needs its own migration.
- `make migrate` sets only `DATABASE_URL`; `config.LoadDatabase` also needs `DB_MAX_CONNS`
  and `DB_MIN_CONNS` in the environment.
- README still mentions golang-migrate / `migration-up` in older sections.

Frontend (FluentFoxUI) follow-ups:
- Align TS types in `src/types/learn.ts`, `progress.ts`, `content.ts` with the backend DTOs once
  those exist. Frontend assumes: lower-case level in URLs (`n5`), kana response has
  `sections: KanaSection[]`, R2 layout `<base>/kana/<script>.json` and
  `<base>/kanji/v1/strokes/<file>.json`, labels `strong | good | shaky | missed`.
- `useJapaneseSpeech` exposes no `onend`; wire audio start/stop into `useAnswerTimer`.
- Unit pages embed quiz/practice panels inline, so guests can still reach them; gate them.
- `RequireAuth` redirects to home with the login modal; no "Sign in to practise" prompt yet.
- PDF generation still reads bundled kana stroke files.
- Stored unsent answers are not tied to a user; the server rejects another user's answers and
  the queue drops them on 4xx.
- No frontend test runner exists, so the timer and queue have no committed tests.
