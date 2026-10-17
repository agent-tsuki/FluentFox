package common

// JLPTLevel maps to the jlpt_level PostgreSQL enum.
type JLPTLevel string

const (
	JLPTLevelN1 JLPTLevel = "N1"
	JLPTLevelN2 JLPTLevel = "N2"
	JLPTLevelN3 JLPTLevel = "N3"
	JLPTLevelN4 JLPTLevel = "N4"
	JLPTLevelN5 JLPTLevel = "N5"
)

// KanaType maps to the kana_type PostgreSQL enum.
type KanaType string

const (
	KanaTypeHiragana KanaType = "hiragana"
	KanaTypeKatakana KanaType = "katakana"
)

// QuizType maps to the quiz_type PostgreSQL enum.
type QuizType string

const (
	QuizTypeHiragana  QuizType = "hiragana"
	QuizTypeKatakana  QuizType = "katakana"
	QuizTypeVocab     QuizType = "vocabulary"
	QuizTypeGrammar   QuizType = "grammar"
)

// SRSContentType maps to the srs_content_type PostgreSQL enum.
type SRSContentType string

const (
	SRSContentHiragana  SRSContentType = "hiragana"
	SRSContentKatakana  SRSContentType = "katakana"
	SRSContentVocab     SRSContentType = "vocabulary"
	SRSContentKanji     SRSContentType = "kanji"
)

// SRSItemType maps to the srs_item_type PostgreSQL enum.
type SRSItemType string

const (
	SRSItemKana  SRSItemType = "kana"
	SRSItemKanji SRSItemType = "kanji"
)

// CardFacet maps to the card_facet PostgreSQL enum. Writing is unused in v1.
type CardFacet string

const (
	CardFacetRecognition CardFacet = "recognition"
	CardFacetMeaning     CardFacet = "meaning"
	CardFacetReading     CardFacet = "reading"
	CardFacetWriting     CardFacet = "writing"
)

// LearningTrack maps to the learning_track PostgreSQL enum.
type LearningTrack string

const (
	LearningTrackHiragana LearningTrack = "hiragana"
	LearningTrackKatakana LearningTrack = "katakana"
	LearningTrackKanjiN5  LearningTrack = "kanji_n5"
)

// SessionKind maps to the session_kind PostgreSQL enum.
type SessionKind string

const (
	SessionKindLesson    SessionKind = "lesson"
	SessionKindReview    SessionKind = "review"
	SessionKindPractice  SessionKind = "practice"
	SessionKindUnitQuiz  SessionKind = "unit_quiz"
	SessionKindSkipTest  SessionKind = "skip_test"
	SessionKindFinalTest SessionKind = "final_test"
)

// QuestionType maps to the question_type PostgreSQL enum.
type QuestionType string

const (
	QuestionKanaMCQRomaji     QuestionType = "kana_mcq_romaji"
	QuestionKanaMCQKana       QuestionType = "kana_mcq_kana"
	QuestionKanaTyped         QuestionType = "kana_typed"
	QuestionKanaListen        QuestionType = "kana_listen"
	QuestionKanaBuild         QuestionType = "kana_build"
	QuestionKanaLookalike     QuestionType = "kana_lookalike"
	QuestionKanaSpeed         QuestionType = "kana_speed"
	QuestionKanaMatch         QuestionType = "kana_match"
	QuestionKanaTrace         QuestionType = "kana_trace"
	QuestionKanjiMeaningMCQ   QuestionType = "kanji_meaning_mcq"
	QuestionKanjiMeaningTyped QuestionType = "kanji_meaning_typed"
	QuestionKanjiReadingMCQ   QuestionType = "kanji_reading_mcq"
	QuestionKanjiReadingTyped QuestionType = "kanji_reading_typed"
	QuestionKanjiListen       QuestionType = "kanji_listen"
	QuestionKanjiLookalike    QuestionType = "kanji_lookalike"
)

// GradeReason maps to the grade_reason PostgreSQL enum.
type GradeReason string

const (
	GradeReasonFast          GradeReason = "fast"
	GradeReasonNormal        GradeReason = "normal"
	GradeReasonSlow          GradeReason = "slow"
	GradeReasonMCQCap        GradeReason = "mcq_cap"
	GradeReasonTypo          GradeReason = "typo"
	GradeReasonHint          GradeReason = "hint"
	GradeReasonReplay        GradeReason = "replay"
	GradeReasonWrong         GradeReason = "wrong"
	GradeReasonRevealed      GradeReason = "revealed"
	GradeReasonTooFast       GradeReason = "too_fast"
	GradeReasonIdle          GradeReason = "idle"
	GradeReasonStrokeErrors  GradeReason = "stroke_errors"
	GradeReasonAnalyticsOnly GradeReason = "analytics_only"
)

// GuideLevel maps to the guide_level PostgreSQL enum.
type GuideLevel string

const (
	GuideLevelFull  GuideLevel = "full"
	GuideLevelFaded GuideLevel = "faded"
	GuideLevelNone  GuideLevel = "none"
)

// AnswerMode maps to the answer_mode PostgreSQL enum.
type AnswerMode string

const (
	AnswerModeChoice AnswerMode = "choice"
	AnswerModeTyped  AnswerMode = "typed"
	AnswerModeMixed  AnswerMode = "mixed"
)

// UnitType maps to the unit_type PostgreSQL enum.
type UnitType string

const (
	UnitTypeKana  UnitType = "kana"
	UnitTypeKanji UnitType = "kanji"
)

// TestKind maps to the test_kind PostgreSQL enum.
type TestKind string

const (
	TestKindUnitQuiz TestKind = "unit_quiz"
	TestKindSkip     TestKind = "skip"
	TestKindFinal    TestKind = "final"
)

// GradeLabel is the word users see for a rating; it is not stored.
type GradeLabel string

const (
	GradeLabelStrong GradeLabel = "strong"
	GradeLabelGood   GradeLabel = "good"
	GradeLabelShaky  GradeLabel = "shaky"
	GradeLabelMissed GradeLabel = "missed"
)

// KanaScript maps to the kana_script PostgreSQL enum.
type KanaScript string

const (
	KanaScriptHiragana KanaScript = "hiragana"
	KanaScriptKatakana KanaScript = "katakana"
)

// KanaGroup maps to the kana_group PostgreSQL enum.
type KanaGroup string

const (
	KanaGroupBasic   KanaGroup = "basic"
	KanaGroupDakuten KanaGroup = "dakuten"
	KanaGroupCombo   KanaGroup = "combo"
	KanaGroupForeign KanaGroup = "foreign"
)

// ReadingType maps to the reading_type PostgreSQL enum.
type ReadingType string

const (
	ReadingTypeOn  ReadingType = "on"
	ReadingTypeKun ReadingType = "kun"
)

// KunAffix maps to the kun_affix PostgreSQL enum.
type KunAffix string

const (
	KunAffixPrefix KunAffix = "prefix"
	KunAffixSuffix KunAffix = "suffix"
)
