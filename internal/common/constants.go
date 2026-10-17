package common

import (
	"time"

	"github.com/google/uuid"

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

// Learning sessions (docs/learning-plan.md sections 4.2 and 7).
const (
	// Review cards this stable may be asked as typed questions.
	TYPED_MIN_STABILITY_DAYS float64 = 3
	// One question in this many is a listening or look-alike question.
	VARIETY_QUESTION_ONE_IN int = 4
	MCQ_DISTRACTORS         int = 3
	MAX_ANSWERS_PER_BATCH   int = 50
	// A day also counts toward the streak when everything due was cleared
	// with at least this many answers.
	GOAL_CLEARED_MIN_ANSWERS int = 5
)

// IntRange is an inclusive range; the learning settings ranges match the
// CHECK constraints in migrations/000006_learning_settings.sql.
type IntRange struct {
	Min int
	Max int
}

type FloatRange struct {
	Min float64
	Max float64
}

const (
	MIN_DESIRED_RETENTION   float64 = 0.70
	MAX_DESIRED_RETENTION   float64 = 0.97
	MAX_NEW_KANA_PER_DAY    int     = 100
	MAX_NEW_KANJI_PER_DAY   int     = 50
	MIN_MAX_REVIEWS_PER_DAY int     = 10
	MAX_MAX_REVIEWS_PER_DAY int     = 1000
	MIN_SESSION_SIZE        int     = 5
	MAX_SESSION_SIZE        int     = 100
	MIN_LESSON_SIZE         int     = 1
	MAX_LESSON_SIZE         int     = 20
	MIN_DAILY_GOAL_ANSWERS  int     = 5
	MAX_DAILY_GOAL_ANSWERS  int     = 200
	MAX_DAY_STARTS_AT_HOUR  int     = 23
)

var (
	DESIRED_RETENTION_RANGE   = FloatRange{Min: MIN_DESIRED_RETENTION, Max: MAX_DESIRED_RETENTION}
	NEW_KANA_PER_DAY_RANGE    = IntRange{Min: 0, Max: MAX_NEW_KANA_PER_DAY}
	NEW_KANJI_PER_DAY_RANGE   = IntRange{Min: 0, Max: MAX_NEW_KANJI_PER_DAY}
	MAX_REVIEWS_PER_DAY_RANGE = IntRange{Min: MIN_MAX_REVIEWS_PER_DAY, Max: MAX_MAX_REVIEWS_PER_DAY}
	SESSION_SIZE_RANGE        = IntRange{Min: MIN_SESSION_SIZE, Max: MAX_SESSION_SIZE}
	LESSON_SIZE_RANGE         = IntRange{Min: MIN_LESSON_SIZE, Max: MAX_LESSON_SIZE}
	DAILY_GOAL_ANSWERS_RANGE  = IntRange{Min: MIN_DAILY_GOAL_ANSWERS, Max: MAX_DAILY_GOAL_ANSWERS}
	DAY_STARTS_AT_HOUR_RANGE  = IntRange{Min: 0, Max: MAX_DAY_STARTS_AT_HOUR}
)

// desired_retention is numeric(3,2).
const DESIRED_RETENTION_DECIMALS int = 2

// Path order of the tracks: the next unit is looked for in this order.
var LEARNING_TRACKS = []LearningTrack{LearningTrackHiragana, LearningTrackKatakana, LearningTrackKanjiN5}

var LEARNING_TRACK_ITEM_TYPE = map[LearningTrack]SRSItemType{
	LearningTrackHiragana: SRSItemKana,
	LearningTrackKatakana: SRSItemKana,
	LearningTrackKanjiN5:  SRSItemKanji,
}

var LEARNING_TRACK_KANA_SCRIPT = map[LearningTrack]KanaScript{
	LearningTrackHiragana: KanaScriptHiragana,
	LearningTrackKatakana: KanaScriptKatakana,
}

var LEARNING_TRACK_KANJI_LEVEL = map[LearningTrack]JLPTLevel{
	LearningTrackKanjiN5: JLPTLevelN5,
}

// The first facet of an item is its primary one: creating that card counts
// the item as newly learned.
var SRS_ITEM_FACETS = map[SRSItemType][]CardFacet{
	SRSItemKana:  {CardFacetRecognition},
	SRSItemKanji: {CardFacetMeaning, CardFacetReading},
}

var QUESTION_TYPE_FACET = map[QuestionType]CardFacet{
	QuestionKanaMCQRomaji:     CardFacetRecognition,
	QuestionKanaMCQKana:       CardFacetRecognition,
	QuestionKanaTyped:         CardFacetRecognition,
	QuestionKanaListen:        CardFacetRecognition,
	QuestionKanaBuild:         CardFacetRecognition,
	QuestionKanaLookalike:     CardFacetRecognition,
	QuestionKanaSpeed:         CardFacetRecognition,
	QuestionKanaMatch:         CardFacetRecognition,
	QuestionKanaTrace:         CardFacetRecognition,
	QuestionKanjiMeaningMCQ:   CardFacetMeaning,
	QuestionKanjiMeaningTyped: CardFacetMeaning,
	QuestionKanjiLookalike:    CardFacetMeaning,
	QuestionKanjiReadingMCQ:   CardFacetReading,
	QuestionKanjiReadingTyped: CardFacetReading,
	QuestionKanjiListen:       CardFacetReading,
}

// Card ids are derived from (user, item, facet) so a lesson can hand out the
// id of a card that is only created when its first answer arrives.
var SRS_CARD_ID_NAMESPACE = uuid.MustParse("6f0c7a52-3d1e-4b8a-9c57-2a64e1f0b9d3")

// SessionKindRule says what answers in a session do. Sessions that do not
// schedule are analytics only, apart from the safety net. CreatesCards lets a
// session hand out ids of cards that are only created by their first answer.
type SessionKindRule struct {
	Schedules    bool
	CreatesCards bool
	// Empty when the session is not a test.
	Test TestKind
}

var SESSION_KIND_RULES = map[SessionKind]SessionKindRule{
	SessionKindLesson:    {Schedules: true, CreatesCards: true},
	SessionKindReview:    {Schedules: true},
	SessionKindPractice:  {CreatesCards: true},
	SessionKindUnitQuiz:  {CreatesCards: true, Test: TestKindUnitQuiz},
	SessionKindSkipTest:  {CreatesCards: true, Test: TestKindSkip},
	SessionKindFinalTest: {CreatesCards: true, Test: TestKindFinal},
}

var TEST_KIND_SESSION_KIND = map[TestKind]SessionKind{
	TestKindUnitQuiz: SessionKindUnitQuiz,
	TestKindSkip:     SessionKindSkipTest,
	TestKindFinal:    SessionKindFinalTest,
}

// Pass marks (share of questions right) match the frontend course configs:
// kana PASS_MARK and SKIP_PASS_MARK, and the kanji level config.
const (
	KANA_UNIT_QUIZ_PASS_MARK  float64 = 0.8
	KANJI_UNIT_QUIZ_PASS_MARK float64 = 0.9
	SKIP_TEST_PASS_MARK       float64 = 0.9
	FINAL_TEST_PASS_MARK      float64 = 0.8
)

var TEST_PASS_MARKS = map[TestKind]map[SRSItemType]float64{
	TestKindUnitQuiz: {SRSItemKana: KANA_UNIT_QUIZ_PASS_MARK, SRSItemKanji: KANJI_UNIT_QUIZ_PASS_MARK},
	TestKindSkip:     {SRSItemKana: SKIP_TEST_PASS_MARK, SRSItemKanji: SKIP_TEST_PASS_MARK},
	TestKindFinal:    {SRSItemKana: FINAL_TEST_PASS_MARK, SRSItemKanji: FINAL_TEST_PASS_MARK},
}

// TEST_SCOPE_ALL is the scope of a final test, and of a skip test over the
// whole track.
const TEST_SCOPE_ALL string = "all"

const PERCENT float64 = 100

// Weak practice ranks cards by the first two terms of the weakness score
// (docs/learning-plan.md section 7.4): recall chance and lapse rate.
const (
	WEAK_RECALL_WEIGHT float64 = 0.4
	WEAK_LAPSE_WEIGHT  float64 = 0.3
)

// A look-alike pair is sent as two characters joined by this, e.g. "ぬ:め".
const PRACTICE_PAIR_SEPARATOR string = ":"

// Matches varchar(1000) on kanji_note.text, which counts characters.
const KANJI_NOTE_MAX_CHARS int = 1000

var LEARNING_TRACK_UNIT_TYPE = map[LearningTrack]UnitType{
	LearningTrackHiragana: UnitTypeKana,
	LearningTrackKatakana: UnitTypeKana,
	LearningTrackKanjiN5:  UnitTypeKanji,
}

// Speed and match practice ask kana with these analytics-only types; kanji
// get ordinary multiple choice.
var PRACTICE_QUICK_KANA_TYPES = map[PracticeMode]QuestionType{
	PracticeModeSpeed: QuestionKanaSpeed,
	PracticeModeMatch: QuestionKanaMatch,
}

// The facet and question type a look-alike drill uses per item type; kanji
// look-alikes are told apart by meaning.
var LOOKALIKE_FACETS = map[SRSItemType]CardFacet{
	SRSItemKana:  CardFacetRecognition,
	SRSItemKanji: CardFacetMeaning,
}

var LOOKALIKE_QUESTION_TYPES = map[SRSItemType]QuestionType{
	SRSItemKana:  QuestionKanaLookalike,
	SRSItemKanji: QuestionKanjiLookalike,
}

// Mastery levels (docs/learning-plan.md section 7.4), lowest first. A card
// in a learning step is Learning whatever its stability.
var MASTERY_LEVELS = []MasteryLevel{MasteryNew, MasteryLearning, MasteryFamiliar, MasteryStrong, MasteryMastered}

const (
	FAMILIAR_MIN_STABILITY_DAYS float64 = 1
	STRONG_MIN_STABILITY_DAYS   float64 = 7
	MASTERED_MIN_STABILITY_DAYS float64 = 21
)

// Progress views (docs/learning-plan.md section 7.4).
const (
	RETENTION_WINDOW         time.Duration = 30 * 24 * time.Hour
	FORECAST_DEFAULT_DAYS    int           = 14
	CALENDAR_DEFAULT_DAYS    int           = 365
	CALENDAR_MAX_DAYS        int           = 366
	WEAK_ITEMS_DEFAULT_LIMIT int           = 20
	CONFUSIONS_DEFAULT_LIMIT int           = 10
	ITEM_HISTORY_ANSWERS     int           = 20
	// Far enough into a study day to be inside it whatever its length.
	STUDY_DAY_MIDPOINT time.Duration = 12 * time.Hour
	// One decimal place for percentages.
	PERCENT_ROUNDING float64 = 10
)

// Weakness score: WEAK_RECALL_WEIGHT and WEAK_LAPSE_WEIGHT above, plus how
// slow the user is on the card and how often it is mixed up with another.
const (
	WEAK_TIME_WEIGHT      float64 = 0.2
	WEAK_CONFUSION_WEIGHT float64 = 0.1
	// Cards with fewer reviews say too little to rank.
	WEAK_MIN_REPS int = 2
	// Answer time this many times the user's usual reads as slow.
	WEAK_SLOW_RATIO float64 = 1.5
	// Recall chance shown as a reason only below this.
	WEAK_LOW_RECALL         float64 = 0.8
	WEAK_GROUP_MIN_REVIEWS  int     = 10
	WEAK_GROUP_LAPSE_FACTOR float64 = 1.5
)

// Kana groups checked as a whole besides their units; the basic group is
// covered by its rows.
var WEAK_KANA_GROUPS = []KanaGroup{KanaGroupDakuten, KanaGroupCombo, KanaGroupForeign}

var QUESTION_TYPE_SKILL = map[QuestionType]Skill{
	QuestionKanaMCQRomaji:     SkillKanaReading,
	QuestionKanaMCQKana:       SkillKanaReading,
	QuestionKanaTyped:         SkillKanaReading,
	QuestionKanaBuild:         SkillKanaReading,
	QuestionKanaLookalike:     SkillKanaReading,
	QuestionKanaSpeed:         SkillKanaReading,
	QuestionKanaMatch:         SkillKanaReading,
	QuestionKanjiMeaningMCQ:   SkillKanjiMeaning,
	QuestionKanjiMeaningTyped: SkillKanjiMeaning,
	QuestionKanjiLookalike:    SkillKanjiMeaning,
	QuestionKanjiReadingMCQ:   SkillKanjiReading,
	QuestionKanjiReadingTyped: SkillKanjiReading,
	QuestionKanaListen:        SkillListening,
	QuestionKanjiListen:       SkillListening,
}

// Dashboard (frontend DashboardData). v1 teaches N5, so that is the target
// level until the user picks one.
const (
	DASHBOARD_STUDY_LOG_SIZE int       = 10
	DASHBOARD_WEEK_DAYS      int       = 7
	DEFAULT_TARGET_LEVEL     JLPTLevel = JLPTLevelN5
	MILESTONE_FIRST_KANJI    int       = 20
	MILESTONE_SHORT_STREAK   int       = 7
	MILESTONE_LONG_STREAK    int       = 30
	STUDY_LOG_DATE_LAYOUT    string    = "Jan 2, 2006"
)
