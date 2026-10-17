# Learning system: implementation progress

Plan: `docs/learning-plan.md`. This file says what is done and what remains, so work can
resume from here at any time.

Branches: backend `claude/task-gllti3` (FluentFox), frontend `claude/task-gllti3` (FluentFoxUI).

## Status

| Phase | Part | Status |
|---|---|---|
| 1 | goose, `000001_users` baseline, `cmd/migrate` switch, Makefile | in progress |
| 1 | migrations `000002`–`000008` (all tables from the plan) | in progress |
| 2 | backend content: `content/*.json`, `cmd/sync-content`, `internal/content`, `/content/*` | in progress |
| 2 | frontend content services on `/content/*` | in progress |
| 2 | upload stroke JSON to R2, set `strokes_base_url` | not started |
| 3 | `pkg/srs` (go-fsrs v4 wrapper), `internal/learn/grading.go`, learning enums + constants | in progress |
| 4 | frontend `useAnswerTimer`, answer queue, `learnService` / `progressService` clients, auth guards | in progress |
| 4 | backend sessions: settings, question builder, lessons, reviews, answers, complete | not started |
| 4 | frontend lesson / review / result screens, remove `ts-fsrs` | not started |
| 5 | practice modes, safety net, unit / skip / final tests, unit progress, notes | not started |
| 6 | progress APIs (overview, forecast, calendar, weakness, confusions, items, dashboard) | not started |
| 6 | frontend progress pages, dashboard, settings | not started |
| — | tests for each finished part (written after the code) | not started |
