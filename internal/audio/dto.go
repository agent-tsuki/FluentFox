package audio

// maxItems matches common.AUDIO_MAX_TEXTS (huma needs a literal); each text's
// length (common.AUDIO_MAX_TEXT_RUNES) is checked by the service.
type SpeechRequest struct {
	Texts []string `json:"texts" minItems:"1" maxItems:"20" doc:"Japanese texts to speak, up to 300 characters each; each is split into sentences, one clip per sentence"`
	Voice string   `json:"voice,omitempty" enum:"main,alt" doc:"main (default), or alt: the second voice of a dialogue; the main voice answers when there is none"`
}

type SpeechResponse struct {
	Items   []SpeechItem `json:"items"   doc:"One per text, in request order"`
	Credits []string     `json:"credits" doc:"Credit lines to show for the voices used"`
	Pending bool         `json:"pending" doc:"Some segments have no url yet (per-request or hourly limit, or no voice is up); ask again for those texts later"`
}

type SpeechItem struct {
	Segments []SpeechSegment `json:"segments" doc:"The text's sentences, played one after another"`
}

type SpeechSegment struct {
	Text  string `json:"text"`
	URL   string `json:"url,omitempty"   doc:"Public CDN URL of the clip; missing while the clip is not ready"`
	Voice string `json:"voice,omitempty" doc:"Voice profile that spoke it, e.g. voicevox-14 or openjtalk-mei_normal"`

	credit string
}

type SpeechInfo struct {
	Enabled bool   `json:"enabled"`
	BaseURL string `json:"base_url,omitempty" doc:"Main voice's folder: clip URL = base_url + \"/\" + hex(sha256(sentence)) + ext"`
	Ext     string `json:"ext,omitempty"      doc:".mp3, or .wav where the server has no ffmpeg"`
	Credit  string `json:"credit,omitempty"   doc:"Credit line for the main voice"`

	AltBaseURL string `json:"alt_base_url,omitempty" doc:"The second voice's folder (voice: alt), named like base_url; missing when there is no second voice"`
	AltCredit  string `json:"alt_credit,omitempty"   doc:"Credit line for the second voice"`
}
