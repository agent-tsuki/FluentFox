# Learning system: implementation progress

Plan: `docs/learning-plan.md`. This file says what is done and what remains, so work can
resume from here at any time.

Branches: backend `claude/task-gllti3` (FluentFox), frontend `claude/task-gllti3` (FluentFoxUI).

## Status

| Phase | Part | Status |
|---|---|---|
| 1 | goose, `000001_users` baseline, `cmd/migrate` switch, Makefile | done (000001 identical to AutoMigrate; up/down/up verified on Postgres 16) |
| 1 | migrations `000002`–`000009` | done |
| 2 | backend content: `content/*.json`, `cmd/sync-content`, `internal/content`, `/content/*` | done, tested; sync verified twice on real migrations |
| 2 | frontend content on `/content/*`, kana chart grid built from API | done (FluentFoxUI `b333f6d`, `1089d6c`) |
| 2 | upload stroke JSON to R2, set `STROKES_BASE_URL` | **not started** |
| 3 | `pkg/srs` (go-fsrs v4), `internal/learn/grading.go`, enums + constants | done, tested |
| 4 | backend: settings, summary, question builder, lessons, reviews, answers, complete | done, tested (FluentFox `4745719`…`5ad7450`); end-to-end checked on Postgres incl. concurrent duplicate batches |
| 4 | frontend: answer timer + queue, lesson / Review all / result screens, how-grading page, leech hint, `ts-fsrs` removed, guest prompts, Vitest (75 tests) | done (FluentFoxUI up to `7142efb`); not yet clicked through against a live backend |
| 5 | backend: practice, tests (unit/skip/final), safety net, unit progress, kanji notes | done, tested (FluentFox `dd1cdc9`…`eefc0b4`); end-to-end checked on Postgres |
| 5 | frontend: practice / tests / path state / notes on the API, local progress record removed | done (FluentFoxUI up to `b71c0a5`, 130 tests); not yet clicked through against a live backend |
| 6 | backend: progress APIs (overview, forecast, calendar, weak, weak-groups, confusions, items, dashboard) + `GET /learn/tests/history` | done, tested (FluentFox `382ce85`…`66d6ee3`); end-to-end checked on Postgres (Asia/Tokyo user) |
| 6 | frontend progress pages, dashboard, settings page | in progress |

## Next steps (in order)

1. Run backend + frontend together and click through a lesson, a review and the result
   screen (nothing has been tested in a browser yet).
2. Phase 6 backend progress APIs, then the frontend progress pages, dashboard and settings.
3. Upload stroke JSON to R2.

## Decisions taken during implementation

- Go 1.26 (go-fsrs v4 needs it). Dockerfiles use `golang:1.26-alpine`; golangci-lint must be
  v2.6.2+ built with Go 1.26.
- The frontend builds the kana chart grid from the API's groups/units/kana.
- Grading defaults where the plan was silent: typing 300 ms/char before user data; log-time
  spread 0.3 (floor 0.1); full guide caps at Hard, faded at Good (reason `hint`); on/kun mix-up
  and wrong build tiles use reason `wrong`; any idle time or > 30 s gives `idle` and Good.
- Card ids are derived from user + item + facet, so lesson questions carry a `card_id` before
  the card exists; the card is created on its first check answer.
- No work available (limit reached, nothing due) returns an empty session, not an error.
- `reviews_done` counts every answer (daily goal); `max_reviews_per_day` counts only answers
  on cards already in progress. Hiragana and katakana share one new-item limit.
- Answer time uses the client's `shown_at` + `duration_ms`, clamped between session start and
  server time. Completing a session twice returns the same result.
- Typed meanings ignore brackets, articles and punctuation before the typo check; readings are
  compared in hiragana. Romaji → kana conversion happens in the frontend input only.
- `GradeCounts` on the result screen count cards (first graded answer), not answers.
- Practice and test answers create an unscheduled card (state 0) when none exists, because
  every `review_log` row needs a card. State-0 cards are ignored by lessons and new limits.
- Safety net applies to wrong unrated answers in practice and tests, and to speed / match /
  trace misses in any session.
- Pass marks: kana unit quiz 0.8, kanji unit quiz 0.9, skip test 0.9, final test 0.8. Tests
  ask every facet of every item in scope (N5 kanji final = 158 questions).
- Unit progress takes `?track=` because kana slugs repeat across scripts.

## Known gaps and follow-ups

Backend:
- `/progress/weak` ranks per card, so a kanji can appear twice (meaning and reading).
- Dashboard `skillBalance`: grammar, vocabulary and speaking are always 0; `studyLog.xp` is null.
- Practice and test answers count toward `max_reviews_per_day`; consider excluding them.
- No Docker integration tests committed for the learn SQL (checked by hand on Postgres).
- On/kun mix-up rule ("wrong reading, then fixed → Hard") never fires: the request does not
  carry the first attempt.
- A resent answer returns its stored result, but `usual_ms` is recalculated.
- Content is read from the database on every request; could be cached.
- Existing user tables keep `user_id` as `text` with no FKs (as AutoMigrate built them).
- `make migrate` sets only `DATABASE_URL`; `DB_MAX_CONNS` / `DB_MIN_CONNS` must also be set.
  `cmd/sync-content` uses `config.Load()`, so it needs the full API env.
- Kana group titles are hard-coded in `internal/content/kana_groups.go`.
- 20 N5 look-alikes point to non-N5 kanji and are skipped; 11 link once N4 is synced.
- `kanji_look_alike` has no position column; order comes from `created_at`.
- `internal/kanji/model.go` (old, unused) maps to the same table name `kanji`.
- README still mentions golang-migrate / `migration-up` in older sections.

Frontend:
- Until phase 6: kana mastery colours and kanji stages show "New"; kanji "Your progress"
  section removed.
- Tested-out units, skip/final best scores, certificate name and Print step have no API yet;
  stored per user in localStorage (`ff-path-v2:<user>:<course>`), so not synced across
  devices. Needs a backend endpoint for test history / skips.
- Answers not saved: kana outside the session's cards (whole-chart speed round), kanji
  flashcards (self-graded), kanji writing.
- Kana tests build their own options from the server's item list; ー and ッ are asked via
  loanwords and sent as `kana_mcq_kana`.
- N4 practice/quizzes/tests show "coming soon" (API serves N5 only).
- Old localStorage progress is left in storage (not imported, not wiped).
- `KanjiQuizRunner` keeps its own typed input instead of `TypedAnswerInput`.
- PDF generation still reads bundled kana stroke files.
- Unsent answers stored on the device are not tied to a user (server rejects, queue drops).

Regenerating content: `node scripts/content/convert-frontend.mjs [uiPath]`, then
`make sync-content`.
