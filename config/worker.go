package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/spf13/viper"

	"github.com/fluentfox/api/internal/common"
)

// WorkerConfig is what the audio worker (cmd/audioworker) needs: the
// database, R2, the voice engine and ffmpeg. The VOICEVOX_* and FFMPEG_BIN
// values must match the API's, or the worker would make clips in other
// folders (or another format) than the API looks in.
type WorkerConfig struct {
	DatabaseURL string
	DBMaxConns  int32
	DBMinConns  int32

	R2AccountID  string
	R2AccessKey  string
	R2SecretKey  string
	R2BucketName string
	R2PublicURL  string

	VoicevoxURL       string
	VoicevoxEngine    string
	VoicevoxSpeakerID int
	VoicevoxCredit    string
	FFmpegBin         string
}

// LoadWorker reads the worker's settings from the environment. It needs
// none of the API's other settings (JWT, mail, OAuth).
func LoadWorker() *WorkerConfig {
	v := viper.New()
	v.AutomaticEnv()
	v.SetDefault("VOICEVOX_SPEAKER_ID", common.VOICEVOX_DEFAULT_SPEAKER)
	v.SetDefault("VOICEVOX_CREDIT", common.VOICEVOX_DEFAULT_CREDIT)
	v.SetDefault("VOICEVOX_ENGINE", common.VOICEVOX_DEFAULT_ENGINE)
	cfg := &WorkerConfig{
		DatabaseURL:       require(v, "DATABASE_URL"),
		DBMaxConns:        int32(requireInt(v, "DB_MAX_CONNS")), //nolint:gosec // a small pool size, as in LoadDatabase
		DBMinConns:        int32(requireInt(v, "DB_MIN_CONNS")), //nolint:gosec // a small pool size, as in LoadDatabase
		R2AccountID:       require(v, "R2_ACCOUNT_ID"),
		R2AccessKey:       require(v, "R2_ACCESS_KEY"),
		R2SecretKey:       require(v, "R2_SECRET_KEY"),
		R2BucketName:      require(v, "R2_BUCKET_NAME"),
		R2PublicURL:       require(v, "R2_PUBLIC_URL"),
		VoicevoxURL:       strings.TrimRight(require(v, "VOICEVOX_URL"), "/"),
		VoicevoxEngine:    strings.ToLower(v.GetString("VOICEVOX_ENGINE")),
		VoicevoxSpeakerID: v.GetInt("VOICEVOX_SPEAKER_ID"),
		VoicevoxCredit:    v.GetString("VOICEVOX_CREDIT"),
		FFmpegBin:         v.GetString("FFMPEG_BIN"),
	}
	if !regexp.MustCompile(common.VOICEVOX_ENGINE_PATTERN).MatchString(cfg.VoicevoxEngine) {
		panic(fmt.Sprintf("config: VOICEVOX_ENGINE %q must match %s", cfg.VoicevoxEngine, common.VOICEVOX_ENGINE_PATTERN))
	}
	return cfg
}
