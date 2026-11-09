// Command audioworker runs the audio jobs a super-admin queues on the admin
// Audio page: generating a voice's missing clips, or installing an AivisHub
// model. It runs on Railway next to the voice engine, so long jobs are not
// cut short by a web request's time limit. On start it also installs the
// models the voice cast uses, so a fresh engine gets them.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/fluentfox/api/config"
	"github.com/fluentfox/api/internal/audio"
	"github.com/fluentfox/api/pkg/database"
	"github.com/fluentfox/api/pkg/storage"
)

const aivisEngine = "aivisspeech"

func main() {
	cfg := config.LoadWorker()
	log, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	if err := run(cfg, log); err != nil {
		log.Fatal("audio worker", zap.Error(err))
	}
}

func run(cfg *config.WorkerConfig, log *zap.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := database.NewDB(cfg.DatabaseURL, cfg.DBMaxConns, cfg.DBMinConns)
	if err != nil {
		return fmt.Errorf("connect to the database: %w", err)
	}
	repo := audio.NewRepository(db)
	runner := audio.NewJobRunner(repo, newCast(cfg, repo, log), func(ctx context.Context, id string) error {
		return audio.EnsureAivisModels(ctx, cfg.VoicevoxURL, []string{id}, log)
	}, log)

	if cfg.VoicevoxEngine == aivisEngine {
		installCastModels(ctx, cfg, repo, log)
	}
	log.Info("audio worker started", zap.String("engine", cfg.VoicevoxEngine))
	runner.Run(ctx)
	log.Info("audio worker stopped")
	return nil
}

// newCast speaks like the API: the env's main voice, the cast from
// audio_voice, the same encoder and the same clip index.
func newCast(cfg *config.WorkerConfig, repo *audio.Repository, log *zap.Logger) *audio.Cast {
	store := storage.NewR2Store(cfg.R2AccountID, cfg.R2AccessKey, cfg.R2SecretKey, cfg.R2BucketName, cfg.R2PublicURL)
	engine := audio.NewEngine(cfg.VoicevoxURL, cfg.VoicevoxEngine)
	main := []audio.Voice{{
		Profile: audio.EngineProfile(cfg.VoicevoxEngine, cfg.VoicevoxSpeakerID),
		Credit:  cfg.VoicevoxCredit,
		Synth:   engine.Synth(cfg.VoicevoxSpeakerID),
	}}
	var encoder audio.Encoder = audio.WAVPassthrough{}
	if cfg.FFmpegBin != "" {
		encoder = audio.NewFFmpegMP3(cfg.FFmpegBin)
	}
	speech := audio.NewSpeechService(store, cfg.R2PublicURL, main, encoder, log).WithClipIndex(repo)
	return audio.NewCast(speech, repo, engine.Synth, log)
}

// A missing model is logged, not fatal: the other voices still work, and an
// admin can queue the install again from the Audio page.
func installCastModels(ctx context.Context, cfg *config.WorkerConfig, repo *audio.Repository, log *zap.Logger) {
	models, err := repo.CastModels(ctx)
	if err != nil {
		log.Warn("list the cast's voice models", zap.Error(err))
		return
	}
	if err := audio.EnsureAivisModels(ctx, cfg.VoicevoxURL, models, log); err != nil {
		log.Warn("install the cast's voice models", zap.Error(err))
	}
}
