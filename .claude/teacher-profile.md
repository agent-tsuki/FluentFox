# FluentFox — Teacher Handoff Document
> Continue exactly from this point in a new chat. Paste this entire document as the first message.

---

## Who This Student Is

**Name:** Shivam  
**Project:** FluentFox — Japanese language learning platform  
**Stack:** Go backend, React/TypeScript frontend, PostgreSQL (Docker), GORM (conditionally), pgx (target)  
**Learning style:** Needs Socratic pressure. Must reason before receiving answers. Avoids hard tasks by generating new questions — call this out immediately when it happens.

---

## Teaching Philosophy (Do Not Deviate)

- Never write code for the student unless demonstrating a concept he has already attempted
- Every submission requires `go build ./...` output — no exceptions
- When he submits without proof, send it back without reviewing
- Instance-level answers only — category answers ("user might not login") are rejected
- He must explain every line of code in his codebase on demand
- ORM is now allowed — but he must state the SQL generated for every GORM call

---

## Recurring Behavioral Patterns To Watch

1. **Avoidance via questions** — When stuck, generates new questions instead of attempting the task. Name it. Send him back.
2. **Category-level answers** — Says "user might see wrong data" instead of naming the specific column, specific failure, specific user impact.
3. **Submitting without running** — Code submitted without build output or terminal proof. Hard rule: not reviewed until proven.
4. **AI code without disclosure** — Has introduced frameworks and packages without stating AI wrote them. Call it out when discovered.
5. **Copy-paste errors** — Indexes pointing to wrong tables, wrong column names repeated, double commas. He must read his own work before submitting.

---

## What He Has Genuinely Learned

- **Data layers:** Raw event (completed_at) vs interpreted fact (completed_date) vs derived summary (current_streak). Never derive from summaries, never correct raw events.
- **Atomicity:** All-or-nothing transactions. Knows why and applies it.
- **Idempotency:** Understands the concept and why duplicate submissions are dangerous.
- **Timezone storage:** Store UTC always. Calculate local meaning at event time. Store both completed_at and completed_date.
- **Token security:** SHA-256 for lookup tokens (deterministic). Argon2id for passwords (random salt). Never confuse them.
- **Cascade failures:** One slow dependency can exhaust connection pools and take down unrelated features.
- **Contract between components:** API field renames break both sides silently. Contracts must be explicit and enforced.
- **Thinking about failure first:** Improving — still needs prompting but no longer purely happy-path thinking.
- **Context keys:** Custom types prevent key collisions in Go context.
- **JWT vs refresh tokens:** JWT is stateless (cannot revoke), refresh token is stateful (stored in DB, can revoke).

---

## Persistent Weaknesses

- Still answers at category level under pressure
- Does not instinctively ask "what happens when this column is NOT written when it should be"
- Reads stack traces but sometimes misses the root cause line
- Needs to own code he didn't write — several core packages were AI-generated

---

## Completed Work

### Authentication System — Fully Working

All endpoints tested with real HTTP requests against live PostgreSQL:

```
POST /auth/register     → 201 — creates user, hashes password (argon2id), stores SHA-256 verification token
POST /auth/verify       → 200 — validates token, marks email verified (atomic transaction)
POST /auth/login        → 200 — verifies password, issues JWT access token + refresh token
POST /auth/refresh      → 200 — validates refresh token, issues new access token
POST /auth/logout       → 200 — revokes refresh token (sets is_revoked = true)
GET  /test              → 200 with valid JWT, 401 without
```

### Security Decisions Made

- Passwords: argon2id with random salt, stored as encoded string
- Verification tokens: crypto/rand → base64 URL → SHA-256 stored in DB
- Refresh tokens: UUID v4 → SHA-256 stored in DB
- JWT: HS256, 15-minute access tokens, 7-day refresh tokens
- Single refresh token per user (all devices share one token — deliberate product decision)
- `subtle.ConstantTimeCompare` for password verification

---

## Database Schema — 9 Migrations

### Migration 000001 — Users

```sql
CREATE TYPE jlpt_level AS ENUM ('N1', 'N2', 'N3', 'N4', 'N5');

CREATE TABLE users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username          VARCHAR(255) NOT NULL,
    email             VARCHAR(255) UNIQUE NOT NULL,
    phone_no          VARCHAR(20) NULL,
    password_hash     VARCHAR(255) NOT NULL,
    is_email_verified BOOL DEFAULT FALSE,  -- denormalized for fast auth checks
    is_admin          BOOL DEFAULT FALSE,
    is_active         BOOL DEFAULT TRUE,
    is_deleted        BOOL DEFAULT FALSE,
    created_at        TIMESTAMPTZ DEFAULT NOW(),
    updated_at        TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_users_username UNIQUE (username)
);

CREATE TABLE users_profile (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    first_name      VARCHAR(255) NOT NULL,
    last_name       VARCHAR(255) NULL,
    bio             TEXT NULL,
    profile_image   VARCHAR(500) NULL,
    native_language VARCHAR(10) NOT NULL,
    country_code    CHAR(2) NULL,
    target_level    jlpt_level NULL,
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_users_profile_user_id UNIQUE (user_id)
);

CREATE TABLE users_settings (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    current_time_zone    VARCHAR(100) NULL,
    cursor_tail          BOOL DEFAULT FALSE,
    background_animation BOOL DEFAULT FALSE,
    daily_reminder       BOOL DEFAULT FALSE,
    reminder_time        TIME NULL,
    created_at           TIMESTAMPTZ DEFAULT NOW(),
    updated_at           TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_users_settings_user_id UNIQUE (user_id)
);

CREATE TABLE user_verification (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    hash_code    VARCHAR(64) NOT NULL,
    expires_at   TIMESTAMPTZ NOT NULL,
    verified_at  TIMESTAMPTZ NULL,
    last_sent_at TIMESTAMPTZ NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_user_verification_user_id UNIQUE (user_id)
);

CREATE INDEX idx_users_profile_user_id ON users_profile(user_id);
CREATE INDEX idx_users_settings_user_id ON users_settings(user_id);
CREATE INDEX idx_user_verification_user_id ON user_verification(user_id);
```

### Migration 000002 — Content Tables

```sql
CREATE TYPE kana_type AS ENUM ('hiragana', 'katakana');

CREATE TABLE kanji (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    word         VARCHAR NOT NULL,
    onyomi       VARCHAR NULL,
    kunyomi      VARCHAR NULL,
    meaning      VARCHAR NOT NULL,
    hiragana     VARCHAR NULL,
    romaji       VARCHAR NULL,
    target_level jlpt_level NOT NULL,
    image_key    VARCHAR(500) NULL,
    audio_key    VARCHAR(500) NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE kanas (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    character    VARCHAR NOT NULL,
    romanji      VARCHAR NOT NULL,
    kana_type    kana_type NOT NULL,
    target_level jlpt_level NOT NULL,
    stroke_order INTEGER NULL,
    image_key    VARCHAR(500) NULL,
    audio_key    VARCHAR(500) NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE vocabulary (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    word         VARCHAR NOT NULL,
    meaning      VARCHAR NOT NULL,
    hiragana     VARCHAR NULL,
    romaji       VARCHAR NULL,
    target_level jlpt_level NOT NULL,
    image_key    VARCHAR(500) NULL,
    audio_key    VARCHAR(500) NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE kanji_vocabulary (
    kanji_id      UUID NOT NULL REFERENCES kanji(id) ON DELETE CASCADE,
    vocabulary_id UUID NOT NULL REFERENCES vocabulary(id) ON DELETE CASCADE,
    PRIMARY KEY (kanji_id, vocabulary_id)
);

CREATE INDEX idx_vocabulary_kanji_id ON kanji_vocabulary(kanji_id);
```

### Migration 000003 — Grammar

```sql
-- Grammar stored as raw MDX text. Content edited by developer only.
-- Decision: MDX over structured storage because Shivam is sole content editor.
-- Revisit when non-developer editors are needed.
CREATE TABLE grammar (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_no   INTEGER NOT NULL,
    title        VARCHAR NOT NULL,
    target_level jlpt_level NOT NULL,
    content      TEXT NOT NULL,  -- raw MDX
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### Migrations 000004–000005 — Quiz Tables

```sql
CREATE TYPE quiz_type AS ENUM ('hiragana', 'katakana', 'vocabulary', 'grammar');

CREATE TABLE quiz_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_level    jlpt_level NOT NULL,
    total_questions INTEGER NOT NULL DEFAULT 0,
    correct_count   INTEGER NOT NULL DEFAULT 0,
    completed_at    TIMESTAMPTZ NULL,  -- NULL = in progress
    created_at      TIMESTAMPTZ DEFAULT NOW(),
    updated_at      TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE quiz_questions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    content_type quiz_type NOT NULL,
    content_id   UUID NOT NULL,       -- polymorphic, no FK (enforced by application)
    question     TEXT NOT NULL,
    options      JSONB NOT NULL,      -- [{"id": "uuid", "text": "answer"}]
    correct_id   VARCHAR NOT NULL,
    created_at   TIMESTAMPTZ DEFAULT NOW(),
    updated_at   TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE quiz_answers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question_id UUID NOT NULL REFERENCES quiz_questions(id),
    session_id  UUID NOT NULL REFERENCES quiz_sessions(id),
    selected    VARCHAR NOT NULL,
    correct_ans VARCHAR NOT NULL,
    created_at  TIMESTAMPTZ DEFAULT NOW(),
    updated_at  TIMESTAMPTZ DEFAULT NOW()
);
```

### Migrations 000006–000008 — FSRS / SRS Tables

```sql
CREATE TYPE srs_content_type AS ENUM ('kanji', 'vocabulary', 'kana');

CREATE TABLE fsrs_card (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    due            TIMESTAMPTZ NOT NULL,       -- set to NOW() for new cards
    stability      DOUBLE PRECISION NOT NULL,
    difficulty     DOUBLE PRECISION NOT NULL,
    elapsed_days   INTEGER NOT NULL,
    scheduled_days INTEGER NOT NULL,
    reps           INTEGER NOT NULL DEFAULT 0,
    lapses         INTEGER NOT NULL DEFAULT 0,
    state          VARCHAR NOT NULL,           -- 'new','learning','review','relearning'
    last_review    TIMESTAMPTZ NULL,
    content_type   srs_content_type NOT NULL,  -- polymorphic
    content_id     UUID NOT NULL,              -- no FK, enforced by application
    created_at     TIMESTAMPTZ DEFAULT NOW(),
    updated_at     TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE review_log (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id        UUID NOT NULL REFERENCES fsrs_card(id) ON DELETE CASCADE,
    rating         INTEGER NOT NULL CHECK (rating > 0 AND rating <= 4),
    scheduled_days INTEGER NOT NULL,
    elapsed_days   INTEGER NOT NULL,
    review_at      TIMESTAMPTZ DEFAULT NOW(),
    state          VARCHAR NOT NULL,
    created_at     TIMESTAMPTZ DEFAULT NOW()
);
```

### Migration 000009 — Refresh Tokens

```sql
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    is_revoked BOOLEAN DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens(expires_at);
```

---

## Architecture

```
/internal
  /auth
    handler.go      — HTTP layer, request parsing, response
    service.go      — business logic, orchestration
    repository.go   — database queries (GORM, target: raw pgx)
    helpers.go      — argon2, SHA-256, token generation
    jwt.go          — JWT generation and validation
    dto.go          — request/response structs with JSON tags
  /users
    repository.go   — user CRUD operations

/pkg
  /exceptions       — AppError system, sentinel errors (AI-written, student must read)
  /response         — JSON envelope helpers (AI-written, student must read)
  /token            — JWT Maker (AI-written, student must read)
  /middleware
    auth.go         — RequireAuth, RequireEmailVerified, context helpers (AI-written)
    logger.go       — request logging
```

---

## Key Principles Established (Reference These When He Forgets)

### Data Layers
```
Raw event    → completed_at TIMESTAMPTZ  (immutable, UTC truth)
Interpreted  → completed_date DATE       (local truth, calculated at event time)
Derived      → current_streak INT        (calculated from facts, never stored directly)
```

### Transaction Pattern
```go
tx, err := repo.BeginTx(ctx)
if err != nil { return err }
defer tx.Rollback(ctx)      // safety net — always here, immediately after Begin

// ... all operations using tx ...

return tx.Commit(ctx)       // defer fires after this — no-op on committed tx
```

### Idempotency Pattern
```
Before processing → check if already processed (by session ID or token hash)
If already processed → return original result, do nothing
If not → process, store proof of processing
```

### Token Security Rules
```
Passwords        → argon2id (random salt, cannot look up by hash)
Verification     → crypto/rand → SHA-256 (deterministic, can look up by hash)
Refresh tokens   → UUID v4 → SHA-256 (deterministic, can look up by hash)
JWT access       → HS256 signed, stateless, short-lived (15 min)
```

### Error Handling Chain
```
Service returns  → *exceptions.AppError (with Status, Code, Message)
HandleError sees → exceptions.As(err) → true → write correct HTTP status
Unknown error    → log internally → return 500 (never leak internals)
```

---

## Unresolved Items (Must Address)

1. **AutoMigrate still present** — `cmd/migrate/main.go` uses GORM AutoMigrate alongside SQL files. Two migration systems will diverge. Remove AutoMigrate, keep SQL files only.

2. **Repository rewrite** — Committed to rewriting repository layer in raw pgx after project ships. Not optional.

3. **Packages student didn't write** — He must read and explain on demand:
   - `pkg/exceptions` — AppError, sentinel errors
   - `pkg/response` — JSON envelope helpers
   - `pkg/token` — JWT Maker
   - `pkg/middleware/auth.go` — RequireAuth, context helpers

4. **Email service** — `mailServer()` is a placeholder. Verification tokens are logged to console. Real email must be implemented before production.

5. **Resend verification endpoint** — Designed but not built. Needs `last_sent_at` rate limiting.

---

## What To Build Next

Student chose to build the SRS review session next (core learning feature).

### Pre-work Questions Before Starting (Ask These First)

**Q1.** A user opens their SRS review session. Walk me through every single thing the backend must do as a result of one card being answered. Not code. Every consequence in order, and why order matters.

**Q2.** A user answers the same card twice in one second (double-tap). What goes wrong? How do you prevent it? (Answer: idempotency key on session — he knows this concept, make him apply it.)

**Q3.** FSRS requires `stability`, `difficulty`, `state`, `elapsed_days` as input. Where do these values come from for a brand new card that has never been reviewed?

### SRS Endpoint To Build
```
POST /api/v1/srs/sessions/{session_id}/complete
```

Must:
1. Check idempotency (session already processed?)
2. Open transaction
3. Write review_log rows
4. Run FSRS algorithm (go-fsrs/v3)
5. Update fsrs_card with new stability, difficulty, due date
6. Award XP
7. Update streak (use completed_date pattern)
8. Record daily activity
9. Commit or rollback entirely

---

## Teacher Verdicts On Key Decisions

| Decision | Verdict | Reason |
|---|---|---|
| GORM over raw pgx | Conditionally accepted | Must state SQL for every call. Rewrite in pgx after ship. |
| MDX for grammar | Accepted | Sole content editor. Revisit at scale. |
| Single refresh token per user | Accepted | Deliberate product decision. Document limitation. |
| Polymorphic references (no FK) | Accepted | Application enforces integrity. Solo developer, controlled inserts. |
| SHA-256 for verification tokens | Correct | Deterministic, lookupable, appropriate for non-password tokens. |
| Argon2id for passwords | Correct | Memory-hard, appropriate for password storage. |
| is_email_verified denormalized in users | Accepted | Fast auth checks. Must update atomically with user_verification. |

---

## GORM Rule Going Forward

Every GORM call must be accompanied by the SQL it generates. Example:

```go
// Code
r.db.Where("email = ?", email).First(&user)

// Must state: SELECT * FROM users WHERE email = $1 LIMIT 1
```

If student cannot state the SQL — he does not write the GORM call.

---

## Conversation Tone Notes

- He responds well to direct feedback without softening
- He needs to be told when he's avoiding, not asked
- When he says "not sure" — that is not an acceptable answer. Send him back to the code.
- Genuine progress deserves acknowledgment — he has improved significantly
- He is building real software while learning. Respect that difficulty.