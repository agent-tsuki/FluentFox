package audio

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
)

// JobStore is the audio worker's view of the job queue and the course;
// *Repository satisfies it.
type JobStore interface {
	RequeueStaleJobs(ctx context.Context, staleAfter time.Duration) error
	ClaimJob(ctx context.Context) (*Job, error)
	SaveJobProgress(ctx context.Context, j *Job) error
	FinishJob(ctx context.Context, j *Job, status common.AudioJobStatus, jobErr string) error
	CourseTexts(ctx context.Context) ([]CourseText, error)
}

// ModelInstaller installs an AivisHub model into the engine.
type ModelInstaller func(ctx context.Context, modelUUID string) error

// JobRunner is the audio worker: it runs queued jobs one at a time, next to
// the voice engine, so generating a whole course is not bound by a web
// request's time limit.
type JobRunner struct {
	jobs    JobStore
	cast    *Cast
	install ModelInstaller
	wait    time.Duration
	log     *zap.Logger
}

func NewJobRunner(jobs JobStore, cast *Cast, install ModelInstaller, log *zap.Logger) *JobRunner {
	return &JobRunner{jobs: jobs, cast: cast, install: install, wait: common.AUDIO_JOB_ROUND_WAIT, log: log}
}

// Run polls for jobs until ctx ends.
func (r *JobRunner) Run(ctx context.Context) {
	for {
		ran, err := r.RunNext(ctx)
		if err != nil {
			r.log.Warn("audio worker", zap.Error(err))
		}
		if ran {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(common.AUDIO_JOB_POLL):
		}
	}
}

// RunNext runs the next queued job, if any.
func (r *JobRunner) RunNext(ctx context.Context) (bool, error) {
	if err := r.jobs.RequeueStaleJobs(ctx, common.AUDIO_JOB_STALE_AFTER); err != nil {
		return false, fmt.Errorf("audio worker: %w", err)
	}
	job, err := r.jobs.ClaimJob(ctx)
	if err != nil {
		return false, fmt.Errorf("audio worker: %w", err)
	}
	if job == nil {
		return false, nil
	}
	log := r.log.With(zap.String("job", job.ID.String()), zap.String("kind", string(job.Kind)))
	log.Info("audio job started")
	runErr := r.run(ctx, job)
	if errors.Is(runErr, errJobStopped) {
		log.Info("audio job stopped (cancelled)")
		return true, nil
	}
	status, msg := common.AudioJobDone, ""
	if runErr != nil {
		status, msg = common.AudioJobFailed, truncateRunes(runErr.Error(), common.AUDIO_JOB_ERROR_RUNES)
	} else if job.Failed > 0 {
		status, msg = common.AudioJobFailed, fmt.Sprintf("%d texts still have no clip (the engine may be busy or down); run the job again", job.Failed)
	}
	// Finish even when the worker is shutting down, so the job does not wait to go stale.
	if err := r.jobs.FinishJob(context.WithoutCancel(ctx), job, status, msg); err != nil {
		return true, fmt.Errorf("audio worker: %w", err)
	}
	log.Info("audio job finished", zap.String("status", string(status)), zap.Int("done", job.Done), zap.Int("failed", job.Failed))
	return true, nil
}

func (r *JobRunner) run(ctx context.Context, job *Job) error {
	switch job.Kind {
	case common.AudioJobInstallModel:
		if job.ModelUUID == nil {
			return errors.New("no model_uuid")
		}
		job.Total = 1
		if err := r.install(ctx, job.ModelUUID.String()); err != nil {
			return fmt.Errorf("install model: %w", err)
		}
		job.Done = 1
		r.cast.Invalidate()
		return nil
	case common.AudioJobGenerate:
		return r.generate(ctx, job)
	default:
		return fmt.Errorf("unknown job kind %q", job.Kind)
	}
}

// generate makes every missing clip of the job's voice (or of every voice),
// a batch of texts at a time, asking each batch again until all its clips exist.
func (r *JobRunner) generate(ctx context.Context, job *Job) error {
	byVoice, err := r.textsByVoice(ctx, job.VoiceKey)
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(byVoice))
	for key, texts := range byVoice {
		keys = append(keys, key)
		job.Total += len(texts)
	}
	sort.Strings(keys)
	job.Done, job.Failed = 0, 0
	if err := r.jobs.SaveJobProgress(ctx, job); err != nil {
		return err //nolint:wrapcheck // errJobStopped is checked by the caller
	}
	who := Requester{Admin: true}
	if job.CreatedBy != nil {
		who.UserID = *job.CreatedBy
	}
	for _, key := range keys {
		if err := r.generateVoice(ctx, job, who, key, byVoice[key]); err != nil {
			return err
		}
	}
	return nil
}

func (r *JobRunner) textsByVoice(ctx context.Context, only *string) (map[string][]string, error) {
	texts, err := r.jobs.CourseTexts(ctx)
	if err != nil {
		return nil, fmt.Errorf("course texts: %w", err)
	}
	byVoice := map[string][]string{}
	seen := map[CourseText]bool{}
	for _, t := range texts {
		if (only != nil && t.Voice != *only) || seen[t] {
			continue
		}
		seen[t] = true
		byVoice[t.Voice] = append(byVoice[t.Voice], t.Text)
	}
	return byVoice, nil
}

func (r *JobRunner) generateVoice(ctx context.Context, job *Job, who Requester, key string, texts []string) error {
	svc := r.cast.Voice(ctx, key)
	for start := 0; start < len(texts); start += common.AUDIO_MAX_TEXTS {
		batch := texts[start:min(start+common.AUDIO_MAX_TEXTS, len(texts))]
		done, err := r.speakBatch(ctx, job, svc, who, batch)
		if err != nil {
			return err
		}
		if done {
			job.Done += len(batch)
		} else {
			job.Failed += len(batch)
		}
		if err := r.jobs.SaveJobProgress(ctx, job); err != nil {
			return err //nolint:wrapcheck // errJobStopped is checked by the caller
		}
	}
	return nil
}

// speakBatch asks for a batch until every clip exists or the rounds run out.
// Each round also marks the job alive.
func (r *JobRunner) speakBatch(ctx context.Context, job *Job, svc *SpeechService, who Requester, batch []string) (bool, error) {
	for round := 0; round < common.AUDIO_JOB_MAX_ROUNDS; round++ {
		resp, err := svc.Speak(ctx, who, batch)
		if err != nil {
			return false, fmt.Errorf("speak: %w", err)
		}
		if !resp.Pending {
			return true, nil
		}
		if err := r.jobs.SaveJobProgress(ctx, job); err != nil {
			return false, err //nolint:wrapcheck // errJobStopped is checked by the caller
		}
		select {
		case <-ctx.Done():
			return false, fmt.Errorf("speak: %w", ctx.Err())
		case <-time.After(r.wait):
		}
	}
	return false, nil
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
