package content

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/fluentfox/api/internal/common"
)

// Listening text is written with its readings: words are separated by
// spaces, and every run of kanji (or digits) carries its reading in
// brackets right after it, e.g. "7時[しちじ] に 起[お]きます。". Clients show
// it as written, in hiragana only, or in romaji, so a kanji without a
// reading would leave a hole in two of the three.

var (
	errRubyEmpty       = errors.New("empty text")
	errRubyUnclosed    = errors.New("unclosed [")
	errRubyStray       = errors.New("] without [")
	errRubyNoBase      = errors.New("reading with no kanji or digits before it")
	errRubyBadReading  = errors.New("reading must be kana only")
	errRubyMissingRead = errors.New("kanji or digits without a reading")
)

func isRubyBase(r rune) bool {
	return unicode.Is(unicode.Han, r) || unicode.IsDigit(r)
}

func isKana(r rune) bool {
	return unicode.In(r, unicode.Hiragana, unicode.Katakana) || r == 'ー'
}

// checkRubyText reports the first problem with a listening text's markup.
func checkRubyText(text string) error {
	if strings.TrimSpace(text) == "" {
		return errRubyEmpty
	}
	runes := []rune(text)
	baseLen := 0 // kanji/digits since the last reading, which still need one
	for i := 0; i < len(runes); i++ {
		switch r := runes[i]; {
		case r == '[':
			if baseLen == 0 {
				return fmt.Errorf("%w at %d", errRubyNoBase, i)
			}
			end, err := readingEnd(runes, i)
			if err != nil {
				return err
			}
			i, baseLen = end, 0
		case r == ']':
			return fmt.Errorf("%w at %d", errRubyStray, i)
		case isRubyBase(r):
			baseLen++
		case baseLen > 0:
			return fmt.Errorf("%w before %d", errRubyMissingRead, i)
		}
	}
	if baseLen > 0 {
		return errRubyMissingRead
	}
	return nil
}

// readingEnd finds the ] closing the reading that opens at runes[open],
// checking that the reading is kana only and not empty.
func readingEnd(runes []rune, open int) (int, error) {
	for end := open + 1; end < len(runes); end++ {
		switch r := runes[end]; {
		case r == ']' && end == open+1:
			return 0, fmt.Errorf("%w at %d", errRubyBadReading, open)
		case r == ']':
			return end, nil
		case r == '[':
			return 0, fmt.Errorf("%w at %d", errRubyUnclosed, open)
		case !isKana(r):
			return 0, fmt.Errorf("%w at %d", errRubyBadReading, end)
		}
	}
	return 0, fmt.Errorf("%w at %d", errRubyUnclosed, open)
}

var (
	errPassageKind     = errors.New("kind must be dialogue or monologue")
	errPassageSpeakers = errors.New("a dialogue has 2 to 4 speakers, a monologue at most one (its reader)")
	errSpeakerVoice    = errors.New("a speaker's voice must be a cast key such as f1 or m2")
	errPassageNoLines  = errors.New("no lines")
	errLineSpeaker     = errors.New("line speaker must be one of the dialogue's speakers, and absent in a monologue")
	errLineTranslation = errors.New("line has no translation")
	errLineSay         = errors.New("say must be plain text (no readings) with the line's sentences")
	errPassageTitle    = errors.New("no title")
)

// checkPassage reports the first problem with a passage from a listening file.
func checkPassage(p *ListeningFilePassage) error {
	if strings.TrimSpace(p.Title) == "" {
		return errPassageTitle
	}
	if err := checkSpeakers(p); err != nil {
		return err
	}
	if len(p.Lines) == 0 {
		return errPassageNoLines
	}
	for i := range p.Lines {
		if err := checkLine(p.Kind, len(p.Speakers), &p.Lines[i]); err != nil {
			return fmt.Errorf("line %d: %w", i+1, err)
		}
	}
	return nil
}

var voiceKey = regexp.MustCompile(common.AUDIO_VOICE_KEY_PATTERN)

func checkSpeakers(p *ListeningFilePassage) error {
	lowest, highest := 0, 1
	switch p.Kind {
	case ListeningDialogue:
		lowest, highest = minDialogueSpeakers, maxSpeakers
	case ListeningMonologue:
	default:
		return errPassageKind
	}
	if len(p.Speakers) < lowest || len(p.Speakers) > highest {
		return errPassageSpeakers
	}
	for i, s := range p.Speakers {
		if err := checkRubyText(s.Name); err != nil {
			return fmt.Errorf("speaker %d: %w", i, err)
		}
		if !voiceKey.MatchString(s.Voice) {
			return fmt.Errorf("speaker %d %q: %w", i, s.Voice, errSpeakerVoice)
		}
	}
	return nil
}

func checkLine(kind ListeningPassageKind, speakers int, line *ListeningFileLine) error {
	if kind == ListeningDialogue {
		if line.Speaker == nil || *line.Speaker < 0 || *line.Speaker >= speakers {
			return errLineSpeaker
		}
	} else if line.Speaker != nil {
		return errLineSpeaker
	}
	if err := checkRubyText(line.Text); err != nil {
		return fmt.Errorf("%q: %w", line.Text, err)
	}
	if strings.TrimSpace(line.Translation) == "" {
		return errLineTranslation
	}
	return checkSay(line)
}

// checkSay keeps a line's say speakable the way the line is: plain text, and
// as many sentences (one clip each, which the highlight follows).
func checkSay(line *ListeningFileLine) error {
	if line.Say == nil {
		return nil
	}
	say := *line.Say
	if strings.TrimSpace(say) == "" || strings.ContainsAny(say, "[] ") || sentenceEnds(say) != sentenceEnds(line.Text) {
		return fmt.Errorf("%q: %w", say, errLineSay)
	}
	return nil
}

func sentenceEnds(s string) int {
	return strings.Count(s, "。") + strings.Count(s, "？") + strings.Count(s, "！") + strings.Count(s, "?") + strings.Count(s, "!")
}
