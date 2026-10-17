package common

import (
	"time"

	"github.com/fluentfox/api/pkg/srs"
)

const USERNAME_RETRY int = 3
const CHARSET string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const USERNAME_SUFFIX_LEN int = 7
const AUTH_TOKEN_LEN int = 32
const AUTH_TOKEN_EXPIRE_HOUR int = 6

const (
	PASSWORD_RESET_CODE_LENGTH        int           = 8
	PASSWORD_RESET_CODE_EXPIRES_AFTER time.Duration = 10 * time.Minute
	PASSWORD_RESET_LINK_EXPIRES_AFTER time.Duration = time.Hour
	PASSWORD_RESET_CODE_COOLDOWN      time.Duration = 60 * time.Second
	PASSWORD_RESET_SEND_WINDOW        time.Duration = 24 * time.Hour
	PASSWORD_RESET_DAILY_SEND_LIMIT   int64         = 5
	PASSWORD_RESET_MAX_CODE_ATTEMPTS  int           = 5
)

// Together these must stay well under the server's 15s WriteTimeout, because
// the send runs inside the request.
const (
	EMAIL_SEND_TIMEOUT      time.Duration = 8 * time.Second
	EMAIL_MX_LOOKUP_TIMEOUT time.Duration = 1 * time.Second
)

// The signup email counts toward the daily limit.
const (
	VERIFICATION_RESEND_COOLDOWN  time.Duration = 60 * time.Second
	VERIFICATION_SEND_WINDOW      time.Duration = 24 * time.Hour
	VERIFICATION_DAILY_SEND_LIMIT int64         = 5
)

// The index names must match the gorm tags on users.User.
const (
	PG_UNIQUE_VIOLATION         string = "23505"
	USERS_EMAIL_UNIQUE_INDEX    string = "idx_users_email"
	USERS_USERNAME_UNIQUE_INDEX string = "idx_users_username"
)

// Each argon2id hash uses 64 MB; this caps total hashing memory at 256 MB.
const (
	ARGON2_MAX_CONCURRENT_HASHES int = 4
	ARGON2_HASH_PARTS            int = 6
)

// HS256 can be brute-forced offline from any issued token if the secret is short.
const JWT_SECRET_MIN_BYTES int = 32

// Answer grading thresholds (docs/learning-plan.md section 5). Answer times
// are compared in log space because they are right-skewed.
const (
	MIN_PLAUSIBLE_ANSWER time.Duration = 250 * time.Millisecond
	MAX_ACTIVE_ANSWER    time.Duration = 30 * time.Second
	BASELINE_WINDOW      int           = 50
	BASELINE_COLD_START  float64       = 20
	SLOW_Z               float64       = 1.5
	FAST_Z               float64       = -1.0
	LEECH_LAPSES         int           = 8
	SAFETY_NET_DUE       time.Duration = 24 * time.Hour
	// MAD × 1.4826 estimates the standard deviation of a normal distribution.
	MAD_TO_SIGMA float64 = 1.4826
	// Spread used while the user has few answers, and a floor so a run of
	// identical times cannot make every small difference look extreme.
	DEFAULT_LOG_MAD float64 = 0.3
	MIN_LOG_MAD     float64 = 0.1
	// Used until the user has typed answers of their own.
	DEFAULT_TYPING_PER_CHAR  time.Duration = 300 * time.Millisecond
	FREE_AUDIO_REPLAYS       int           = 1
	BUILD_TILES_AGAIN        int           = 2
	STROKE_ERROR_RATIO_AGAIN float64       = 0.25
)

// English meaning answers: up to MEANING_TYPO_SHORT letters allow no edits,
// up to MEANING_TYPO_MEDIUM allow one, longer ones allow two.
const (
	MEANING_TYPO_SHORT        int = 4
	MEANING_TYPO_MEDIUM       int = 7
	MEANING_TYPO_MEDIUM_EDITS int = 1
	MEANING_TYPO_LONG_EDITS   int = 2
)

// UsualTime is the default time to answer before typing; typed answers add
// typing time per character and build questions add UsualTimePerTile.
type QuestionTypeRule struct {
	UpdatesSchedule  bool
	MaxRating        srs.Rating
	UsualTime        time.Duration
	UsualTimePerTile time.Duration
	Typed            bool
}

// Default "usual time" per question type, before the user has a baseline.
const (
	USUAL_KANA_MCQ           time.Duration = 2 * time.Second
	USUAL_KANA_TYPED         time.Duration = 1500 * time.Millisecond
	USUAL_LISTEN_AFTER_AUDIO time.Duration = 2 * time.Second
	USUAL_BUILD_PER_TILE     time.Duration = time.Second
	USUAL_LOOKALIKE          time.Duration = 3 * time.Second
	USUAL_KANJI_MCQ          time.Duration = 3 * time.Second
	USUAL_MEANING_TYPED      time.Duration = 2 * time.Second
	USUAL_READING_TYPED      time.Duration = 2500 * time.Millisecond
)

var QUESTION_TYPE_RULES = map[QuestionType]QuestionTypeRule{
	QuestionKanaMCQRomaji:     {UpdatesSchedule: true, MaxRating: srs.RatingGood, UsualTime: USUAL_KANA_MCQ},
	QuestionKanaMCQKana:       {UpdatesSchedule: true, MaxRating: srs.RatingGood, UsualTime: USUAL_KANA_MCQ},
	QuestionKanaTyped:         {UpdatesSchedule: true, MaxRating: srs.RatingEasy, UsualTime: USUAL_KANA_TYPED, Typed: true},
	QuestionKanaListen:        {UpdatesSchedule: true, MaxRating: srs.RatingGood, UsualTime: USUAL_LISTEN_AFTER_AUDIO},
	QuestionKanaBuild:         {UpdatesSchedule: true, MaxRating: srs.RatingGood, UsualTimePerTile: USUAL_BUILD_PER_TILE},
	QuestionKanaLookalike:     {UpdatesSchedule: true, MaxRating: srs.RatingGood, UsualTime: USUAL_LOOKALIKE},
	QuestionKanaSpeed:         {},
	QuestionKanaMatch:         {},
	QuestionKanaTrace:         {},
	QuestionKanjiMeaningMCQ:   {UpdatesSchedule: true, MaxRating: srs.RatingGood, UsualTime: USUAL_KANJI_MCQ},
	QuestionKanjiMeaningTyped: {UpdatesSchedule: true, MaxRating: srs.RatingEasy, UsualTime: USUAL_MEANING_TYPED, Typed: true},
	QuestionKanjiReadingMCQ:   {UpdatesSchedule: true, MaxRating: srs.RatingGood, UsualTime: USUAL_KANJI_MCQ},
	QuestionKanjiReadingTyped: {UpdatesSchedule: true, MaxRating: srs.RatingEasy, UsualTime: USUAL_READING_TYPED, Typed: true},
	QuestionKanjiListen:       {UpdatesSchedule: true, MaxRating: srs.RatingGood, UsualTime: USUAL_LISTEN_AFTER_AUDIO},
	QuestionKanjiLookalike:    {UpdatesSchedule: true, MaxRating: srs.RatingGood, UsualTime: USUAL_LOOKALIKE},
}

// Only N5 kanji content ships in v1; other levels answer 404 until they are synced.
var KANJI_CONTENT_LEVELS = []JLPTLevel{JLPTLevelN5}

// The kana chart (frontend KanaType) calls the combo group's kana "compound".
const KANA_CHART_TYPE_COMPOUND string = "compound"
