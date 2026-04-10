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
