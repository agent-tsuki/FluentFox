package learn

import "testing"

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{name: "equal strings are zero", a: "water", b: "water", want: 0},
		{name: "empty strings are zero", a: "", b: "", want: 0},
		{name: "against empty is the length", a: "abc", b: "", want: 3},
		{name: "counts substitutions and inserts", a: "kitten", b: "sitting", want: 3},
		{name: "counts a delete and an insert", a: "flaw", b: "lawn", want: 2},
		{name: "swapped letters cost two", a: "abcd", b: "abdc", want: 2},
		{name: "counts runes not bytes", a: "日本", b: "日", want: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Levenshtein(tt.a, tt.b); got != tt.want {
				t.Fatalf("Levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
			if got := Levenshtein(tt.b, tt.a); got != tt.want {
				t.Fatalf("Levenshtein is not symmetric for %q, %q", tt.a, tt.b)
			}
		})
	}
}

func TestMeaningTypoAllowance(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		want     int
	}{
		{name: "four letters allow no edits", expected: "tree", want: 0},
		{name: "five letters allow one edit", expected: "water", want: 1},
		{name: "seven letters allow one edit", expected: "seventy", want: 1},
		{name: "eight letters allow two edits", expected: "mountain", want: 2},
		{name: "length ignores case and extra spaces", expected: "  Big  ", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MeaningTypoAllowance(tt.expected); got != tt.want {
				t.Fatalf("MeaningTypoAllowance(%q) = %d, want %d", tt.expected, got, tt.want)
			}
		})
	}
}

func TestMatchMeaning(t *testing.T) {
	tests := []struct {
		name     string
		answer   string
		accepted []string
		others   []string
		want     MeaningMatch
	}{
		{name: "exact match is correct", answer: "water", accepted: []string{"water"},
			want: MeaningMatch{Correct: true}},
		{name: "match ignores case and spacing", answer: "  Rice   Field ", accepted: []string{"rice field"},
			want: MeaningMatch{Correct: true}},
		{name: "any accepted meaning matches", answer: "sun", accepted: []string{"day", "sun"},
			want: MeaningMatch{Correct: true}},
		{name: "one edit on a medium word is a typo", answer: "watr", accepted: []string{"water"},
			want: MeaningMatch{Correct: true, TypoDistance: 1}},
		{name: "two edits on a long word is a typo", answer: "mointaim", accepted: []string{"mountain"},
			want: MeaningMatch{Correct: true, TypoDistance: 2}},
		{name: "closest accepted meaning sets the distance", answer: "mountin", accepted: []string{"mountain range", "mountain"},
			want: MeaningMatch{Correct: true, TypoDistance: 1}},
		{name: "any edit on a short word is wrong", answer: "tre", accepted: []string{"tree"}},
		{name: "two edits on a medium word is wrong", answer: "wtr", accepted: []string{"water"}},
		{name: "empty answer is wrong", answer: "   ", accepted: []string{"water"}},
		{name: "another kanji's meaning is wrong", answer: "tower", accepted: []string{"power"}, others: []string{"tower"}},
		{name: "typo as close to another kanji's meaning is wrong", answer: "lower", accepted: []string{"power"}, others: []string{"tower"}},
		{name: "typo closer to the accepted meaning is correct", answer: "powr", accepted: []string{"power"}, others: []string{"tower"},
			want: MeaningMatch{Correct: true, TypoDistance: 1}},
		{name: "exact match wins over a shared meaning", answer: "day", accepted: []string{"day"}, others: []string{"day"},
			want: MeaningMatch{Correct: true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MatchMeaning(tt.answer, tt.accepted, tt.others); got != tt.want {
				t.Fatalf("MatchMeaning(%q) = %+v, want %+v", tt.answer, got, tt.want)
			}
		})
	}
}
