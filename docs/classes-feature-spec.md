# Classes, channel profiles, permissions and teacher invites — shared spec

Source of truth for the four implementation agents. Backend = `FluentFox`
(Go/Huma), frontend = `FluentFoxUi` (React). JSON is **snake_case**, wrapped in
the usual envelope `{ "status": "success", "data": ... }`.

## Product rules

- `/classes` lists curated Japanese courses (YouTube, Udemy, Website) by JLPT level.
- A YouTube course is a playlist; its detail page shows the playlist videos and a
  **channel introduction** (profile picture, banner, intro video, teaching style,
  best for / not for, pros, cons, verdict, YouTube logo mark).
- A course is visible to the public only when `course.is_published` AND its
  `channel.is_published`. Hidden course → 404 on detail.
- A level with zero visible courses is "locked" (coming soon) — computed by the API.
- Channel display fields have a **YouTube value** (written only by sync) and a
  **custom value** (written only by editors). Public value = custom ?? youtube.
  Sync never overwrites custom values. "Reset to YouTube" clears the custom value.
- Teacher edits go live immediately. Every admin write is recorded in `class_change_log`.
- Ratings/reviews are **out of scope** (the mock ones were invented). The UI drops that section.
- Super-admin = `users.is_admin = true`: has every permission implicitly.
- Learner profiles (`/profile`) stay editable only by their owner — admins get no access.

## Permissions (`internal/common/permissions.go`, already written)

`classes.channel.edit`, `classes.channel.media`, `classes.course.edit`,
`classes.publish`, `classes.sync`, `permissions.manage`.
A grant has `resource_type` `""` (all), `"channel"` or `"course"` plus `resource_id`.
A channel grant also covers that channel's courses. Checked against the DB on
every request (not the JWT). Nobody can grant to themselves; only super-admins
can grant `permissions.manage`.

### Go contract between agents

Package `internal/access` (agent A) exposes a concrete type
`*access.Checker` with:

```go
func (c *Checker) Has(ctx context.Context, userID uuid.UUID, perm common.Permission, res common.ResourceRef) (bool, error)
// true when user is admin, or has the grant for all resources, or for res,
// or (res is a course) for the course's channel. Course→channel lookup
// is done by access via its own small query on class_courses.
func (c *Checker) Require(ctx context.Context, perm common.Permission, res common.ResourceRef) error
// reads the user id from the request context (middleware claims);
// returns exceptions.ErrForbidden when not allowed.
```

The classes domain (agent B) declares its own consumer interface:

```go
type PermissionChecker interface {
    Require(ctx context.Context, perm common.Permission, res common.ResourceRef) error
}
```

and `httpserver/server.go` passes `*access.Checker` into it. Until agent A
lands, agent B may test with a fake.

## Migrations (fixed numbers — do not renumber)

- `000014_permissions.sql` — agent A: `user_permissions`
- `000015_staff_invites.sql` — agent A: `staff_invites`
- `000016_classes.sql` — agent B: `class_channels`, `class_courses`, `class_videos`, `class_change_log`
- `000017_classes_seed.sql` — agent B: seed data (see bottom)

Note 000014 must not reference class tables (they come later). `user_permissions.resource_id` is TEXT, no FK.

### Tables

`user_permissions(id uuid pk, user_id uuid fk users on delete cascade, permission text, resource_type text not null default '', resource_id text not null default '', granted_by uuid null, created_at timestamptz, unique(user_id, permission, resource_type, resource_id))`

`staff_invites(id uuid pk, email text (normalized lower), token_hash bytea unique (sha256), invited_by uuid, grants jsonb [{permission, resource_type, resource_id}], expires_at, accepted_at null, accepted_by uuid null, revoked_at null, created_at)`. Expiry 7 days, single use.

`class_channels(id uuid pk, slug text unique, platform text, url text, youtube_channel_id text null, handle text null,
 name_youtube, name_custom, avatar_url_youtube, avatar_url_custom, banner_url_youtube, banner_url_custom,
 description_youtube, description_custom, intro_video_id_youtube, intro_video_id_custom  (all text null),
 subscriber_count bigint null, video_count bigint null, accent_color text null,
 teaching_style text, best_for text[], not_for text[], pros text[], cons text[], verdict text,
 is_published bool default false, last_synced_at null, created_at, updated_at)`

`class_courses(id uuid pk, slug unique, channel_id fk, title, level text (N5..N1), platform text, url, playlist_id null,
 thumbnail_url null, summary text, is_free bool, price_note text, what_you_learn jsonb (string[]), curriculum jsonb ([{title, topics[]}]),
 tags text[], is_published bool default false, is_featured bool default false, sort_order int default 0,
 total_duration_sec int default 0, last_synced_at null, created_at, updated_at)`

`class_videos(id uuid pk, course_id fk on delete cascade, youtube_video_id, title, thumbnail_url, duration_sec int, position int, unique(course_id, position))`

`class_change_log(id uuid pk, actor_id uuid, entity_type text, entity_id uuid, field text, old_value text null, new_value text null, created_at)`

## Public API (agent B) — `Metadata: humautil.Public()`

### `GET /classes?level=N5&platform=youtube&free=true`
```json
{ "items": [CourseCard], "meta": { "levels": [{"level":"N5","count":4,"locked":false}, ...all 5],
  "stats": {"course_count":4,"channel_count":4} } }
```
CourseCard:
```json
{ "slug":"misa-n5-complete","title":"...","summary":"...","level":"N5","platform":"youtube",
  "is_free":true,"is_featured":false,"thumbnail_url":"...","video_count":180,"total_duration_sec":36000,
  "tags":["Grammar"],
  "channel": {"slug":"japanese-ammo-with-misa","name":"Japanese Ammo with Misa","avatar_url":"...","accent_color":"#E53935","subscriber_count":660000} }
```
Ordered: featured first, then sort_order, then title.

### `GET /classes/{slug}` → CourseDetail
```json
{ "slug","title","summary","level","platform","url","playlist_id","is_free","price_note","thumbnail_url",
  "tags":[],"what_you_learn":[],"curriculum":[{"title","topics":[]}],
  "total_duration_sec", "last_synced_at",
  "videos":[{"youtube_video_id","title","thumbnail_url","duration_sec","position"}],
  "channel": ChannelProfile,
  "more_from_channel":[CourseCard], "similar":[CourseCard] }
```
ChannelProfile:
```json
{ "slug","name","platform","url","handle","youtube_channel_id","avatar_url","banner_url","description",
  "intro_video_id","subscriber_count","video_count","accent_color",
  "teaching_style","best_for":[],"not_for":[],"pros":[],"cons":[],"verdict" }
```

## Access API (agent A)

- `GET /admin/me/permissions` (any signed-in user) →
  `{ "is_super_admin": bool, "grants": [{"permission","resource_type","resource_id"}] }`
- `GET /admin/permissions?user_email=` (permissions.manage) → `[{ "id","user_id","user_email","username","permission","resource_type","resource_id","granted_by","created_at" }]`
- `POST /admin/permissions` body `{ "user_email","permission","resource_type","resource_id" }` (permissions.manage)
- `DELETE /admin/permissions/{id}` (permissions.manage)
- `POST /admin/invites` body `{ "email", "grants":[{permission,resource_type,resource_id}] }` (permissions.manage) → Invite; sends email via MailService with link `{AppURL}/invite/{token}`
- `GET /admin/invites` → `[Invite]`; `POST /admin/invites/{id}/resend` (new token, new expiry); `DELETE /admin/invites/{id}` (revoke)
  Invite: `{ "id","email","status":"pending|accepted|expired|revoked","grants":[...],"invited_by","expires_at","accepted_at","created_at" }`
- `GET /invites/{token}` (Public) → `{ "email_masked":"m***@gmail.com","status","grants":[{"permission","resource_type","resource_id","resource_name"}],"expires_at" }`
  (resource_name looked up from class_channels/class_courses by id; null if missing)
- `POST /invites/{token}/accept` (signed in) — user's verified email must equal invite email, else 403 `INVITE_EMAIL_MISMATCH`. Grants inserted + invite marked accepted in one transaction. Errors: `INVITE_NOT_FOUND` 404, `INVITE_EXPIRED` 410, `INVITE_USED` 409, `INVITE_REVOKED` 410.
- Google sign-in must be able to return to `/invite/{token}`: make sure `validOAuthReturnTo` accepts that path (extend minimally if it doesn't).

## Admin classes API (agent B) — signed-in, checked via PermissionChecker

| Method & path | Permission (resource) |
|---|---|
| `GET /admin/classes/channels` | any classes permission; returns only channels the user may edit (all for global/super) |
| `GET /admin/classes/channels/{id}` → AdminChannel | channel.edit or channel.media (channel) |
| `POST /admin/classes/channels` | channel.edit (all) |
| `PATCH /admin/classes/channels/{id}` (text fields, partial) | channel.edit (channel) |
| `POST /admin/classes/channels/{id}/avatar` (raw image body, like profile avatar upload) | channel.media |
| `POST /admin/classes/channels/{id}/banner` | channel.media |
| `PATCH /admin/classes/channels/{id}/intro-video` `{ "video_id" }` (accepts id or YouTube URL) | channel.media |
| `DELETE /admin/classes/channels/{id}/custom/{field}` field ∈ name, avatar_url, banner_url, description, intro_video_id | edit for text fields, media for media fields |
| `PATCH /admin/classes/channels/{id}/visibility` `{ "is_published" }` | classes.publish (channel) |
| `POST /admin/classes/channels/{id}/sync` | classes.sync (channel) |
| `GET /admin/classes/channels/{id}/history` | any permission on channel |
| `GET /admin/classes/courses?channel_id=` | course.edit; includes hidden |
| `GET /admin/classes/courses/{id}`, `POST /admin/classes/courses`, `PATCH /admin/classes/courses/{id}`, `DELETE ...` | course.edit (course) |
| `PATCH /admin/classes/courses/{id}/visibility` `{ "is_published", "is_featured"? }` | classes.publish (course) |
| `POST /admin/classes/courses/{id}/sync-playlist` | classes.sync (course) |

AdminChannel = ChannelProfile + `id`, `is_published`, `last_synced_at`, and for each
overridable field an object `{ "youtube": "...", "custom": "...|null", "effective": "..." }`
under `"fields": { "name", "avatar_url", "banner_url", "description", "intro_video_id" }`.
AdminCourse = CourseDetail fields + `id`, `channel_id`, `is_published`, `is_featured`, `sort_order`.

Images: reuse `profile`'s ObjectStore (R2) + `pkg/imageutil`. Avatar square; banner keeps aspect,
max width 2048. Key `classes/channels/{id}/{avatar|banner}/{uuid}.jpg` (new key per upload).
503 when R2 not configured (same as profile).

YouTube: new optional config `YOUTUBE_API_KEY` (not required; sync returns 503
`YOUTUBE_SYNC_UNAVAILABLE` without it). `channels.list` part=snippet,brandingSettings,statistics
(by id or `forHandle`); `playlistItems.list` + `videos.list` (contentDetails.duration) for playlists.
Never call YouTube on a public request.

## Seed data (000017) — real values verified 2026-10-22

All published, level N5, is_free true unless noted.

| channel slug | name | url | youtube_channel_id | avatar (s240) | intro video | subs |
|---|---|---|---|---|---|---|
| japanese-ammo-with-misa | Japanese Ammo with Misa | https://www.youtube.com/@JapaneseAmmowithMisa | UCBSyd8tXJoEJKIXfrwkPdbA | https://yt3.googleusercontent.com/xOk8YBLsc0lES-K1tM0TxOQf_iwgsgtek9rJrxgLei69lxbEk1WKpxtICsQTxDQE3nPzF50zyQ=s240-c-k-c0x00ffffff-no-rj | WYYaf711IEM | 660000 |
| cure-dolly | Organic Japanese with Cure Dolly | https://www.youtube.com/@organicjapanesewithcuredol49 | UCkdmU8hGK4Fg3LghTVtKltQ | https://yt3.googleusercontent.com/ytc/AIdro_nw8mg4x0-3A56gFtGKYZ6QNfQPwTqqmDFuIhpdmOQU8cQ=s240-c-k-c0x00ffffff-no-rj | TkS5APqAFHA | 106000 |
| tokini-andy | ToKini Andy | https://www.youtube.com/@ToKiniAndy | UCsHzgQRjPzOz_w4HSrNLIVw | https://yt3.googleusercontent.com/Odm1BnpElzpeAsSWhLXe58XTa_5Fk5h5QtQYwGW4UpjEKqknncffawSNwPIESYhcxYVp6HXLKQ=s240-c-k-c0x00ffffff-no-rj | (none) | 233000 |
| nihongo-no-mori | 日本語の森 (Nihongo no Mori) | https://www.youtube.com/@nihongonomori2013 | UCVx6RFaEAg46xfAsD2zz16w | https://yt3.googleusercontent.com/_t0OeOIzQtIRDzEMxlzOTdOSAJhln6BnC3UPFbAadIpXvjmEUWegaJfqB5e75laWE6afeabd=s240-c-k-c0x00ffffff-no-rj | 1SGCu28948U | 932000 |
| japanesepod101 | JapanesePod101 | https://www.japanesepod101.com | — | — | — | — (platform website, is_free false) |
| udemy-miku | Miku Sensei (Udemy) | https://www.udemy.com/course/japanese-for-beginners-n5/ | — | — | — | — (platform udemy, is_free false) |

Course text (title, summary/bio, what_you_learn, curriculum, tags, pros, cons, teaching style)
is taken from `FluentFoxUi/src/api/mock/classes.ts`; seed values go into the `*_youtube`
columns for synced fields and the plain columns for editorial fields. No reviews/ratings.
Videos are not seeded (filled by sync).
