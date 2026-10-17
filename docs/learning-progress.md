# Learning system: implementation progress

Plan: `docs/learning-plan.md`. This file says what is done and what remains, so work can
resume from here at any time.

Branches: backend `claude/task-gllti3` (FluentFox), frontend `claude/task-gllti3` (FluentFoxUI).

## Status

| Phase | Part | Status |
|---|---|---|
| 1 | goose, `000001_users` baseline, `cmd/migrate` switch, Makefile | done (000001 schema identical to AutoMigrate; up/down/up verified on Postgres 16) |
| 1 | migrations `000002`–`000008` (all tables from the plan) | done |
| 2 | backend content: `content/*.json`, `cmd/sync-content`, `internal/content`, `/content/*` | done, with tests; sync verified twice against the real migrations (236 kana, 79 kanji, 386 vocab, 230 kun readings, 174 kana look-alikes) |
| 2 | frontend content services on `/content/*` | done (FluentFoxUI `b333f6d`; falls back to bundled JSON until the backend serves content) |
| 2 | upload stroke JSON to R2, set `strokes_base_url` | not started |
| 3 | `pkg/srs` (go-fsrs v4 wrapper), `internal/learn/grading.go`, learning enums + constants | done, with tests (`pkg/srs`, `internal/learn` pass) |
| 4 | frontend `useAnswerTimer`, answer queue, `learnService` / `progressService` clients, auth guards | done (FluentFoxUI `b333f6d`; not yet used by any screen) |
| 4 | backend sessions: settings, question builder, lessons, reviews, answers, complete | in progress |
| 4 | frontend lesson / review / result screens, remove `ts-fsrs` | in progress |
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

- **Go 1.26 (decision needed).** go-fsrs v4 requires Go 1.26, so `go.mod` is now `go 1.26`.
  `docker/Dockerfile.dev` (`golang:1.25-alpine`) and `docker/Dockerfile.prod`
  (`golang:1.23-alpine`) need `golang:1.26`, and golangci-lint must be a build made with
  Go 1.26 (v2.6.2+). CI reads the version from `go.mod`. Alternative: implement FSRS-6
  ourselves inside `pkg/srs` and stay on Go 1.25.
- Grading choices made where the plan was silent: default typing speed 300 ms/char before
  user data; default log-time spread 0.3 (floor 0.1); full guide caps at Hard, faded at Good
  (reason `hint`); on/kun mix-up and wrong build tiles use reason `wrong`; any idle time or
  > 30 s gives `idle` and Good; known cards (skip test) get reps 1 and due spread by whole days.
- Callers of `Grade` must: feed `MatchMeaning` (typo matching another kanji) into `Correct`,
  and limit history to the last `BASELINE_WINDOW` answers.
- New constants in `internal/common/constants.go` use the file's UPPER_SNAKE style.

Content (phase 2) follow-ups:
- Add `UNIQUE(kanji_id, position)` to `kanji_kun_reading` and `kanji_vocab` (new migration).
  Sync currently matches on (kanji_id, position) without it, and never deletes rows, so a
  shortened list leaves old rows behind.
- Kana group titles/summaries are hard-coded in `internal/content/kana_groups.go`; a
  `kana_group` table would be cleaner.
- 20 N5 look-alikes point to kanji outside N5 (目, 田, 犬, 石 ...) and are skipped with a
  warning. 11 will link once N4 is synced and N5 re-synced.
- `kanji_look_alike` has no position column; order comes from `created_at`.
- The kana API returns groups/units/kana, not the frontend `KanaSection` chart grid. The
  frontend must build the grid (or the backend adds it). Frontend currently expects
  `sections: KanaSection[]` and falls back to bundled data.
- Stroke JSON not uploaded to R2 yet; `STROKES_BASE_URL` must be set per environment.
- Re-generate content with `node scripts/content/convert-frontend.mjs [uiPath]`, then
  `make sync-content`. `cmd/sync-content` uses `config.Load()`, so it needs the full API env.
- `internal/kanji/model.go` (old, unused) maps to the same table name `kanji`.

## Next steps (in order)

1. ~~Go 1.26~~ chosen: Dockerfiles now use `golang:1.26-alpine`. Developers need
   golangci-lint v2.6.2+ built with Go 1.26.
2. ~~Migration `000009`~~ done: unique (kanji_id, position) on kun readings and vocab.
3. ~~Kana API shape~~ decided: the frontend builds the chart grid from groups/units/kana.
4. Phase 4 backend (in progress): learning settings, question builder, lessons, reviews, answers, complete.
5. Phase 4 frontend: wire `useAnswerTimer` + answer queue into lesson/review/result screens,
   remove `ts-fsrs` and the localStorage progress services.
6. Phase 5, then phase 6, as in the plan.

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
