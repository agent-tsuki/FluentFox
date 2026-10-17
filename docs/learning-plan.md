# FoxSensei learning system: final plan (v1)

Spaced-repetition learning for hiragana, katakana and N5 kanji. The user only answers;
the server grades every answer from correctness, answer time and other signals, schedules
the next review with FSRS, and reports progress and weaknesses.

Status: **plan, awaiting approval**. Nothing here is implemented yet.

---

## 0. Decisions

| Topic | Decision |
|---|---|
| Scope v1 | Hiragana (~104), katakana (~130), N5 kanji (79). About 390 cards per user. N4, vocabulary, grammar and writing cards come later. |
| Scheduler | Server only, FSRS-6 via `github.com/open-spaced-repetition/go-fsrs/v4`. `ts-fsrs` is removed from the frontend. |
| Cards | Kana: 1 card (`recognition`). Kanji: 2 cards (`meaning`, `reading`). |
| Grading | Correctness first; time only moves between Hard / Good / Easy, relative to the user's own usual time per question type. MCQ max Good. |
| Low-information modes | Speed round, match grid, tracing: analytics only. A miss pulls the card's due date earlier (never later). |
| Guests | May **browse** hiragana, katakana and N5 content (charts, lesson pages, kanji pages). **Every quiz, test, practice, lesson check and review needs login.** Nothing is stored for guests. |
| Old browser progress | Discarded. No import from localStorage. |
| Schema | goose versioned SQL migrations for **all** tables, including the existing user tables. GORM AutoMigrate is removed. |
| Content | Data the server grades with lives in Postgres, loaded by `cmd/sync-content` from `content/*.json`. Static files the browser loads (stroke JSON) live in Cloudflare R2. |
| Kanji parts | Information only (no part cards, no unlock gating). |
| Existing structs | `internal/fsrs` and `internal/quiz` are left untouched. New tables live beside them. |
| Settings | All limits, the daily goal and target retention are per-user settings. |
| Feedback | Questions carry their correct answer for instant ✓/✗. Answers are sent in batches and re-checked by the server. |
| Reviews | One "Review all" queue across tracks (`track` filter kept in the API). |
| Mastery | Stability ≥ 21 days. A kanji is mastered only when both cards are. |
| Streak | A day counts when the daily goal is met (default 20 answers), or when everything due was cleared with at least 5 answers. |
| Retention display | Shown on the dashboard; can be hidden under Settings → Advanced. |
| Delivery | Backend (FluentFox) and frontend (FluentFoxUI) built together, phase by phase. |

---

## 1. Architecture

```
FluentFoxUI ──► /content/*   (public: kana + N5)     internal/content
            ──► /learn/*     (login + verified)      internal/learn      ──► pkg/srs (go-fsrs wrapper)
            ──► /progress/*  (login + verified)      internal/progress
                                                     Postgres (goose migrations)
            ──► R2 stroke JSON (strokes_base_url)
```

Packages follow CLAUDE.md: `routes.go`, `handler.go`, one service struct per use case,
consumer-side repo interfaces, `dto.go`; constants in `internal/common/constants.go`,
enums in `internal/common/types.go`; wiring in `cmd/api/main.go` / `httpserver/server.go`.

```
internal/content/   KanaContentService, KanjiContentService
internal/learn/
  grading.go        pure Grade(input, baseline, settings) → Result   (no I/O)
  questions.go      QuestionBuilder (type choice, distractors)
  services          StartLessonService, StartReviewService, StartPracticeService,
                    StartTestService, SubmitAnswersService, CompleteSessionService,
                    LearningSettingsService, UnitProgressService, KanjiNoteService
internal/progress/  OverviewService, ForecastService, CalendarService, WeaknessService,
                    ConfusionService, ItemHistoryService, DashboardService
pkg/srs/            Scheduler (NewCard, Next, Retrievability) over go-fsrs, no domain types
migrations/         goose .sql files, embedded in the binary
content/            kana/hiragana.json, kana/katakana.json, kanji/n5.json (seed source)
```

---

## 2. Migrations (goose, `migrations/`)

| File | Contents |
|---|---|
| `000001_users.sql` | Existing tables as AutoMigrate creates them today: `users`, `users_profile`, `users_settings`, `user_verification`, `refresh_tokens`, `oauth_identities`, `password_resets`. Uses `IF NOT EXISTS` so it is safe on databases AutoMigrate already built. Includes the refresh-token dedupe before the unique index. |
| `000002_content_enums.sql` | `kana_script`, `kana_group`, `jlpt_level`, `reading_type`, `kun_affix` |
| `000003_kana.sql` | `kana_unit`, `kana`, `kana_look_alike` |
| `000004_kanji.sql` | `kanji_group`, `kanji_unit`, `kanji`, `kanji_kun_reading`, `kanji_vocab`, `kanji_part`, `kanji_part_link`, `kanji_look_alike` |
| `000005_learning_enums.sql` | `srs_item_type`, `card_facet`, `learning_track`, `session_kind`, `question_type`, `grade_reason`, `guide_level`, `answer_mode`, `unit_type`, `test_kind` |
| `000006_learning_settings.sql` | `learning_settings` |
| `000007_srs.sql` | `srs_card`, `study_session`, `review_log` |
| `000008_progress.sql` | `user_confusion`, `daily_activity`, `unit_progress`, `test_result`, `kanji_note` |

`make migrate` runs `goose up`. `cmd/migrate` switches from AutoMigrate to goose with embedded
files. The schema was produced by AutoMigrate until now, so `000001` must be checked against a
`pg_dump --schema-only` of a database AutoMigrate built.

---

## 3. Enums

```
kana_script     hiragana | katakana
kana_group      basic | dakuten | combo | foreign
jlpt_level      N5 | N4 | N3 | N2 | N1
reading_type    on | kun
kun_affix       prefix | suffix
srs_item_type   kana | kanji
card_facet      recognition | meaning | reading | writing      (writing unused in v1)
learning_track  hiragana | katakana | kanji_n5
session_kind    lesson | review | practice | unit_quiz | skip_test | final_test
question_type   kana_mcq_romaji | kana_mcq_kana | kana_typed | kana_listen | kana_build |
                kana_lookalike | kana_speed | kana_match | kana_trace |
                kanji_meaning_mcq | kanji_meaning_typed | kanji_reading_mcq |
                kanji_reading_typed | kanji_listen | kanji_lookalike
grade_reason    fast | normal | slow | mcq_cap | typo | hint | replay | wrong |
                revealed | too_fast | idle | stroke_errors | analytics_only
guide_level     full | faded | none
answer_mode     choice | typed | mixed
unit_type       kana | kanji
test_kind       unit_quiz | skip | final
```

Card `state` is `smallint` 0–3 (new, learning, review, relearning) to match go-fsrs.
`rating` is `smallint` 1–4 (Again, Hard, Good, Easy).

---

## 4. Tables

### 4.1 Content

```
kana_unit
  id uuid PK, script kana_script, group_key kana_group, slug text, title text,
  summary text, position int, youtube_id text NULL, video_credit text NULL
  UNIQUE(script, slug)

kana
  id uuid PK, script kana_script, slug text, character text, romaji text,
  alt_romaji text[] DEFAULT '{}', group_key kana_group, unit_id uuid FK → kana_unit,
  position int, stroke_count int
  UNIQUE(script, slug), UNIQUE(script, character)

kana_look_alike
  kana_id uuid FK, other_kana_id uuid FK, note text NULL
  PK(kana_id, other_kana_id)

kanji_group
  id uuid PK, level jlpt_level, slug text, title, summary, position   UNIQUE(level, slug)

kanji_unit
  id uuid PK, group_id uuid FK, level jlpt_level, slug text, title, summary, position
  UNIQUE(level, slug)

kanji
  id uuid PK, char text UNIQUE, level jlpt_level, unit_id uuid FK, position int,
  meanings text[], synonyms text[] DEFAULT '{}', onyomi text[] DEFAULT '{}',
  key_reading_type reading_type, key_reading text, reading_tip text NULL,
  mnemonic text NULL, mnemonic_draft bool DEFAULT true,
  strokes int, grade int NULL, freq int NULL

kanji_kun_reading   id, kanji_id FK, kana, okurigana NULL, affix kun_affix NULL, position
kanji_vocab         id, kanji_id FK, word, reading, meaning, is_key bool, position
kanji_part          id, char UNIQUE, name
kanji_part_link     kanji_id FK, part_id FK, position          PK(kanji_id, part_id)
kanji_look_alike    kanji_id FK, other_kanji_id FK, note NULL  PK(kanji_id, other_kanji_id)
```

All content tables have `created_at` / `updated_at`. `cmd/sync-content` upserts by natural
key (`script+slug`, `char`, `level+slug`), so re-running it is safe. Content is never deleted
by sync.

### 4.2 Learning

```
learning_settings
  user_id uuid PK FK → users ON DELETE CASCADE
  desired_retention   numeric(3,2) DEFAULT 0.90  CHECK 0.70–0.97
  new_kana_per_day    int DEFAULT 15   CHECK 0–100
  new_kanji_per_day   int DEFAULT 10   CHECK 0–50
  max_reviews_per_day int DEFAULT 200  CHECK 10–1000
  session_size        int DEFAULT 20   CHECK 5–100
  lesson_size         int DEFAULT 5    CHECK 1–20
  daily_goal_answers  int DEFAULT 20   CHECK 5–200
  day_starts_at_hour  smallint DEFAULT 4  CHECK 0–23
  answer_mode         answer_mode DEFAULT 'mixed'
  show_retention      bool DEFAULT true
  fsrs_weights        double precision[] NULL     (per-user optimizer output, later)
  updated_at
  -- row created lazily with defaults on first read; timezone comes from users_settings

srs_card
  id uuid PK, user_id uuid FK CASCADE, item_type srs_item_type, item_id uuid,
  facet card_facet, state smallint, due timestamptz, stability double precision,
  difficulty double precision, reps int, lapses int, learning_step smallint,
  scheduled_days int, last_review_at timestamptz NULL, is_leech bool DEFAULT false,
  introduced_at timestamptz, created_at, updated_at
  UNIQUE(user_id, item_type, item_id, facet)
  INDEX(user_id, due) WHERE state <> 0

study_session
  id uuid PK, user_id uuid FK CASCADE, kind session_kind, track learning_track NULL,
  scope text NULL (unit slug / group / all), started_at, completed_at NULL,
  answer_count int, correct_count int, active_ms bigint

review_log                                   (append only)
  id uuid PK, user_id uuid FK CASCADE, card_id uuid FK, session_id uuid FK,
  client_event_id uuid, question_type question_type, answered_at timestamptz,
  -- sent by the client
  shown_at timestamptz, duration_ms int, idle_ms int, first_input_ms int NULL,
  attempts smallint, answer_text varchar(64) NULL, chosen_option text NULL,
  options text[] NULL, hints_used smallint, answer_revealed bool, audio_replays smallint,
  audio_ms int NULL, stroke_errors smallint NULL, stroke_count smallint NULL,
  guide_level guide_level NULL,
  -- computed by the server
  is_correct bool, typo_distance smallint NULL, active_ms int,
  rating smallint NULL CHECK (rating BETWEEN 1 AND 4), grade_reason grade_reason,
  counts_for_srs bool, state_before smallint, retrievability_before double precision NULL,
  stability_before, stability_after, difficulty_before, difficulty_after double precision NULL,
  elapsed_days int, scheduled_days int, due_after timestamptz NULL
  UNIQUE(user_id, client_event_id)
  INDEX(user_id, question_type, answered_at DESC)
  INDEX(card_id, answered_at)

user_confusion
  user_id, item_type srs_item_type, target_item_id uuid, chosen_item_id uuid,
  times int, last_at timestamptz
  PK(user_id, item_type, target_item_id, chosen_item_id)

daily_activity
  user_id, local_date date, reviews_done int, new_learned int, correct_count int,
  again_count int, active_ms bigint, due_at_day_start int, goal_met bool
  PK(user_id, local_date)

unit_progress
  user_id, unit_type unit_type, unit_id uuid, watched_at, traced_at, written_at,
  practised_at (all timestamptz NULL), quiz_best_pct smallint NULL, completed_at NULL
  PK(user_id, unit_type, unit_id)

test_result
  id uuid PK, user_id FK, session_id FK, kind test_kind, track learning_track,
  scope text, score int, total int, passed bool, active_ms bigint, taken_at

kanji_note
  user_id FK, kanji_id FK, text varchar(1000), updated_at   PK(user_id, kanji_id)
```

---

## 5. Grading (`internal/learn/grading.go`)

Pure function, no I/O, table-driven tests.

```
if !correct || revealed || attempts > 1 (except on/kun mix-up) → Again (wrong / revealed)
cap := type max, lowered by hint, typo, audio replays, stroke errors, guide level
active := duration − idle − audio
if active < MinPlausibleAnswer || active > MaxActiveAnswer || tab was hidden → min(Good, cap)
z := (ln(active) − blended median) / (1.4826 × MAD)       // user's last 50 correct, same type
rating := Good; z > SlowZ → Hard; z < FastZ → Easy
return min(rating, cap), reason
```

Blended median: `w × user median + (1 − w) × ln(type default)`, `w = n / (n + BaselineColdStart)`.
Typed answers subtract `chars × user's median ms per character` before scoring.

| Question type | Updates schedule | Max | Extra rules |
|---|---|---|---|
| kana_mcq_romaji, kana_mcq_kana | first answer of the day | Good | slow → Hard |
| kana_typed | first answer of the day | Easy | hint → max Hard |
| kana_listen | first answer of the day | Good | time starts when audio ends; replays > 1 → max Hard |
| kana_build | first answer of the day | Good | 1 wrong tile → Hard, 2 or more → Again |
| kana_lookalike | first answer of the day | Good | wrong → Again and confusion counted |
| kana_speed, kana_match, kana_trace | **no** (analytics) | — | miss → `due = min(due, now + 1 day)`; confusion counted |
| kanji_meaning_mcq, kanji_reading_mcq | first answer of the day | Good | slow → Hard |
| kanji_meaning_typed | first answer of the day | Easy | typo accepted → max Good; typo matching another kanji's meaning → Again |
| kanji_reading_typed | first answer of the day | Easy | on/kun mix-up then fixed → Hard; wrong kana → Again (no typo tolerance) |
| kanji_listen | first answer of the day | Good | as kana_listen |
| kanji_lookalike | first answer of the day | Good | as kana_lookalike |

Cards in learning or relearning steps update on every answer. Otherwise only the first graded
answer per card per local day updates FSRS; later answers have `counts_for_srs = false`.

Labels shown to users: Easy → **Strong**, Good → **Good**, Hard → **Shaky**, Again → **Missed**.

### Constants (`internal/common/constants.go`)

```
MinPlausibleAnswer  = 250 * time.Millisecond
MaxActiveAnswer     = 30 * time.Second
BaselineWindow      = 50
BaselineColdStart   = 20
SlowZ               = 1.5
FastZ               = -1.0
LeechLapses         = 8
SafetyNetDue        = 24 * time.Hour
MeaningTypoShort    = 4   // ≤ 4 letters: 0 edits
MeaningTypoMedium   = 7   // 5–7 letters: 1 edit, ≥ 8 letters: 2 edits
BuildTilesAgain     = 2
StrokeErrorRatioAgain = 0.25

Default "usual time" per type:
  kana MCQ 2s · kana typed 1.5s + typing · listen 2s after audio · build 1s per tile
  kanji MCQ 3s · meaning typed 2s + typing · reading typed 2.5s + typing · lookalike 3s
```

---

## 6. Scheduling (`pkg/srs`)

- go-fsrs v4 `DefaultParam()`, with `RequestRetention` from `learning_settings.desired_retention`.
- `EnableFuzz = true`, `EnableShortTerm = true`, learning steps 1 m → 10 m, relearning step 10 m.
- `MaximumInterval = 365 days` (named constant).
- Per-user `fsrs_weights` used when present (optimizer job is out of scope for v1).
- Retrievability for progress views is computed in Go from stability and elapsed time.
- Skip-test pass: cards created in `review` state, stability 7 days, default difficulty, due
  spread at random over the next 1–7 days.
- Leech: `is_leech = true` at 8 lapses; card stays scheduled.

---

## 7. APIs

Paths follow the existing style (no `/api` prefix). Responses use the existing
`{status, data}` envelope. `/learn/*` and `/progress/*` need login and a verified email.

### 7.1 Content (public for kana and N5)

| Method | Path | Response |
|---|---|---|
| GET | `/content/kana/{script}` | units, groups, kana (shape of the frontend `KanaSection` / course types) |
| GET | `/content/kanji/{level}/index` | `KanjiLevelIndex` |
| GET | `/content/kanji/{level}/detail` | `KanjiLevelDetail` |
| GET | `/content/kanji/chars/{char}` | `KanjiEntry`, 404 if unknown |

Each response includes `strokes_base_url` for R2.

### 7.2 Learn

| Method | Path | Purpose |
|---|---|---|
| GET | `/learn/summary?track=` | due now, due today, new left today, learned, next unit, goal progress |
| POST | `/learn/lessons` | `{track}` → session + up to `lesson_size` new items in path order + check quiz |
| POST | `/learn/reviews` | `{track?}` → session + due cards (relearning first, lowest recall chance, oldest due) |
| POST | `/learn/practice` | `{track, mode: weak|unit|lookalike|speed|match, unit?, pair?}` |
| POST | `/learn/tests` | `{kind: unit_quiz|skip|final, track, scope}` |
| POST | `/learn/sessions/{id}/answers` | batch of 1–50 answers → per-answer result |
| POST | `/learn/sessions/{id}/complete` | result screen: Strong/Good/Shaky/Missed, per-card reason and next due, test score |
| GET / PUT | `/learn/settings` | learning settings |
| PUT | `/learn/units/{type}/{id}/progress` | `{step: watched|traced|written|practised}` |
| GET / PUT | `/learn/kanji/{char}/note` | personal note |

Answer request item:
```
client_event_id, card_id, question_type, shown_at, duration_ms, idle_ms, first_input_ms,
answer_text | chosen_option, options[], attempts, hints_used, answer_revealed,
audio_replays, audio_ms, stroke_errors, stroke_count, guide_level
```
Answer result item:
```
client_event_id, correct, correct_answer, label, reason, active_ms, usual_ms, next_due, requeue
```

Submit rules:
- Idempotent on `client_event_id`: a repeat returns the stored result.
- One transaction per batch: lock card rows in id order (`SELECT … FOR UPDATE`), grade, run
  FSRS, insert `review_log`, update `srs_card`, upsert `daily_activity` (including `goal_met`)
  and `user_confusion`.
- Lesson check answers create the cards (first answer per item) and enter learning steps.
- Answers from a session that is not the caller's, or already completed, are rejected.
- `requeue = true` when the card is in a learning step; the frontend shows it again at the
  end of the session.

### 7.3 Question builder

| Card | Question |
|---|---|
| new / learning | multiple choice |
| review, stability ≥ 3 days, `answer_mode` allows | typed |
| 1 in 4 questions | listening or look-alike |

Wrong options (3): the user's past confusions → look-alikes → same kana row / kanji unit →
random. Options that are also correct (じ/ぢ, ず/づ, shared meanings) are never used.

### 7.4 Progress

| Method | Path | Response |
|---|---|---|
| GET | `/progress/overview` | per track: items per level, % mastered, 30-day retention, average time; today; streak |
| GET | `/progress/forecast?days=14` | reviews due per day per track (overdue counted today) |
| GET | `/progress/calendar?from=&to=` | per day: answers, minutes, accuracy, goal met; current / longest streak, active days |
| GET | `/progress/weak?track=&limit=20` | weakest items with reasons and a practice suggestion |
| GET | `/progress/weak-groups?track=` | groups weaker than the user's average |
| GET | `/progress/confusions?track=&limit=10` | top mix-up pairs |
| GET | `/progress/items/{type}/{id}` | per-card level, recall chance, next due, stats, last 20 answers |
| GET | `/progress/dashboard` | frontend `DashboardData` shape |

Levels: New · Learning (learning/relearning or stability < 1 d) · Familiar (1–7 d) ·
Strong (7–21 d) · Mastered (≥ 21 d). Kanji level = lower of its two cards.

Weakness score (cards introduced, reps ≥ 2):
```
0.4 × (1 − recall chance) + 0.3 × lapse rate + 0.2 × min(time ratio − 1, 1) + 0.1 × confusion share
```
Weak group: ≥ 10 reviews and lapse rate ≥ 1.5 × the user's overall rate (or time well above usual).

Dashboard mapping: studyHours ← `daily_activity.active_ms`; cardsReviewed ← review answers;
streak ← `daily_activity.goal_met`; jlptProgress ← % of N5 items (kana included) mastered;
targetLevel ← `users_profile`; skillBalance ← accuracy per skill; globalRank ← null;
milestones ← fixed list (all basic hiragana learned, all hiragana mastered, all katakana
learned, first 20 kanji, N5 kanji complete, 7-day streak, 30-day streak). Retention
("your recall vs target") included unless `show_retention` is false.

---

## 8. Frontend (FluentFoxUI)

1. **Content from API.** `kanjiService.ts` and the kana mock switch to `/content/*`. Bundled
   kanji JSON and kana data move to the backend `content/` folder as seed source. Stroke JSON
   moves to R2.
2. **Services.** New `learnService.ts` and `progressService.ts`. `kanaProgressService.ts`,
   `kanjiProgressService.ts` and `learningPathService.ts` call the API, keeping their hook
   interfaces. localStorage progress keys are removed and old data is discarded.
   `ts-fsrs` is removed.
3. **`useAnswerTimer()`** in every question component: `shown_at`, `first_input_ms`,
   `idle_ms` (Page Visibility), `audio_ms` (speech `onend`), attempts, hints, reveals,
   replays, `client_event_id` (`crypto.randomUUID()`).
4. **Answer queue.** Flush every 5 answers, on complete and on tab hide (`fetch` keepalive).
   Unsent answers kept in localStorage and resent with the same ids.
5. **Screens.** Session result (Strong / Good / Shaky / Missed with reasons), "How grading
   works", "Review all" with due badge, progress pages (mastery, forecast, calendar, weak
   items with "Practise this", weak groups, mix-ups), Settings → Learning and Advanced
   (show retention), leech hint with mnemonic and look-alike comparison.
6. **Guests.** Content pages open for kana and N5. Every quiz, test, practice, lesson check
   and review route sits behind `RequireAuth`, with a "Sign in to practise and save progress"
   prompt.

---

## 9. Build order

Each phase is one backend PR and, where noted, one frontend PR. Each follows the repo's
`/feature` workflow (design, implement, review, tests).

| Phase | Backend | Frontend |
|---|---|---|
| 1. Migrations | goose, `000001_users` baseline, `cmd/migrate` switch, Makefile | — |
| 2. Content | `000002–000004`, `cmd/sync-content`, `content/*.json`, R2 strokes upload, `/content/*` | content services use API, strokes from R2 |
| 3. Scheduler + grading | `pkg/srs`, `grading.go` with full table tests | — |
| 4. Sessions | `000005–000007`, settings, question builder, lessons, reviews, answers, complete | `useAnswerTimer`, answer queue, lesson / review / result screens, auth guards, remove `ts-fsrs` |
| 5. Practice and tests | practice modes, safety net, unit / skip / final tests, unit progress, notes, `000008` | practice + test flows on API, notes, unit progress |
| 6. Progress | overview, forecast, calendar, weakness, confusions, item history, dashboard | progress pages, dashboard, settings pages |

---

## 10. Tests

- `grading.go`: table tests per question type and signal (wrong, revealed, too fast, idle,
  slow, fast, caps, typo, replays, build tiles, cold start blend).
- Services: hand-written fakes of their repo interfaces (CLAUDE.md pattern).
- Repository / transaction: integration tests against the Docker test database
  (`make test`): idempotent resubmit, concurrent batches on the same card, daily limits
  across the day boundary in a non-UTC timezone, `goal_met`.
- Migrations: `goose up` → `down` → `up` on an empty database, and `up` on a database built
  by the old AutoMigrate.
- Frontend: unit tests for `useAnswerTimer` and the answer queue (retry with same ids).

---

## 11. Later (not v1)

N4 kanji (seed only), vocabulary cards, kanji writing cards, per-user FSRS optimizer job,
leaderboard (`globalRank`), audio files instead of TTS.
