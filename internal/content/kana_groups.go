package content

import "github.com/fluentfox/api/internal/common"

type kanaGroupText struct {
	Title   string
	Summary string
}

// The content schema has no kana group table, so the group titles live here,
// copied from content/kana/*.json. A group missing here still appears in the
// response, with its id as the title.
var kanaGroupTexts = map[common.KanaScript]map[common.KanaGroup]kanaGroupText{
	common.KanaScriptHiragana: {
		common.KanaGroupBasic:   {Title: "Basic 46", Summary: "The core hiragana, あ to ん."},
		common.KanaGroupDakuten: {Title: "Dakuten & handakuten", Summary: "Voiced sounds: か → が, は → ぱ."},
		common.KanaGroupCombo:   {Title: "Combinations", Summary: "Small ゃゅょ combinations."},
	},
	common.KanaScriptKatakana: {
		common.KanaGroupBasic:   {Title: "Basic 46", Summary: "The core katakana, ア to ン."},
		common.KanaGroupDakuten: {Title: "Dakuten & handakuten", Summary: "Voiced sounds: カ → ガ, ハ → パ."},
		common.KanaGroupCombo:   {Title: "Combinations", Summary: "Small ャュョ combinations."},
		common.KanaGroupForeign: {Title: "Foreign sounds", Summary: "ー, small ッ and the extra sounds used in loanwords."},
	},
}
