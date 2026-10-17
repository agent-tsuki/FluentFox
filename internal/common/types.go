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
