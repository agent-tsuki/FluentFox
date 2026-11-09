package audio

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/fluentfox/api/internal/common"
	"github.com/fluentfox/api/pkg/exceptions"
)

func castKey(profile, sentence string) string {
	return common.AUDIO_OBJECT_PREFIX + profile + "/" + clipHash(sentence) + ".wav"
}

func TestCast(t *testing.T) {
	ctx := context.Background()
	main := newTestService(newFakeStore(), mainVoice(&fakeSynth{}))
	rows := &fakeCastStore{voices: []CastVoice{{Key: "f1", Engine: "aivisspeech", StyleID: 1, Credit: "AivisSpeech:まお"}}} //nolint:gosec // a credit line, not a secret
	cast := NewCast(main, rows, func(int) Synthesizer { return &fakeSynth{} }, zap.NewNop())

	t.Run("a key with a row speaks only its engine style", func(t *testing.T) {
		svc := cast.Voice(ctx, "f1")
		if len(svc.voices) != 1 || svc.voices[0].Profile != "aivisspeech-1" {
			t.Fatalf("voices = %+v, want only aivisspeech-1", svc.voices)
		}
		if cast.Voice(ctx, "f1") != svc {
			t.Error("the same voice should keep its service (and its known-clips cache)")
		}
	})

	t.Run("a key without a row is spoken by the main voice", func(t *testing.T) {
		if cast.Voice(ctx, "m9") != main {
			t.Error("want the main service")
		}
	})

	t.Run("the cast is read once until invalidated", func(t *testing.T) {
		reads := rows.reads
		cast.Voices(ctx)
		cast.Voices(ctx)
		if rows.reads != reads {
			t.Fatalf("read the cast %d more times, want cached", rows.reads-reads)
		}
		rows.voices[0].StyleID = 2
		cast.Invalidate()
		if p := cast.Voice(ctx, "f1").voices[0].Profile; p != "aivisspeech-2" {
			t.Fatalf("profile = %q after the change, want aivisspeech-2", p)
		}
	})

	t.Run("a database error leaves every key on the main voice", func(t *testing.T) {
		broken := NewCast(main, &fakeCastStore{err: errors.New("db down")}, func(int) Synthesizer { return &fakeSynth{} }, zap.NewNop())
		if broken.Voice(ctx, "f1") != main {
			t.Error("want the main service")
		}
	})
}

func TestClipIndex(t *testing.T) {
	ctx := context.Background()
	admin := Requester{UserID: uuid.New(), Admin: true}

	t.Run("an indexed clip is served without asking storage", func(t *testing.T) {
		store := newFakeStore()
		svc := newTestService(store, mainVoice(&fakeSynth{})).WithClipIndex(newFakeClipIndex(mainKey("はい")))
		resp, err := svc.Find(ctx, []string{"はい"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Items[0].Segments[0].URL == "" || store.heads != 0 {
			t.Fatalf("url = %q, storage heads = %d; want a url and no storage request", resp.Items[0].Segments[0].URL, store.heads)
		}
	})

	t.Run("a clip found only in storage is added to the index", func(t *testing.T) {
		index := newFakeClipIndex()
		svc := newTestService(newFakeStore(mainKey("はい")), mainVoice(&fakeSynth{})).WithClipIndex(index)
		if _, err := svc.Find(ctx, []string{"はい"}); err != nil {
			t.Fatal(err)
		}
		if len(index.recorded) != 1 || index.recorded[0].ObjectKey != mainKey("はい") || index.recorded[0].CreatedBy != nil {
			t.Fatalf("recorded = %+v", index.recorded)
		}
	})

	t.Run("a new clip is recorded with its maker and size", func(t *testing.T) {
		index := newFakeClipIndex()
		svc := newTestService(newFakeStore(), mainVoice(&fakeSynth{})).WithClipIndex(index)
		if _, err := svc.Speak(ctx, admin, []string{"いいえ"}); err != nil {
			t.Fatal(err)
		}
		if len(index.recorded) != 1 || index.recorded[0].Bytes == 0 || *index.recorded[0].CreatedBy != admin.UserID {
			t.Fatalf("recorded = %+v", index.recorded)
		}
	})

	t.Run("cast voices share the index", func(t *testing.T) {
		index := newFakeClipIndex(castKey("aivisspeech-1", "はい"))
		main := newTestService(newFakeStore(), mainVoice(&fakeSynth{})).WithClipIndex(index)
		cast := NewCast(main, &fakeCastStore{voices: []CastVoice{{Key: "f1", Engine: "aivisspeech", StyleID: 1}}},
			func(int) Synthesizer { return &fakeSynth{} }, zap.NewNop())
		resp, err := cast.Voice(ctx, "f1").Find(ctx, []string{"はい"})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(resp.Items[0].Segments[0].URL, "aivisspeech-1/") {
			t.Fatalf("url = %q", resp.Items[0].Segments[0].URL)
		}
	})
}

func TestVoiceSet(t *testing.T) {
	ctx := context.Background()
	model := uuid.New()
	engine := &fakeEngine{
		speakers: []EngineSpeaker{{Name: "まお", SpeakerUUID: "s1", Styles: []EngineStyle{{ID: 888753760, Name: "ノーマル"}}}},
		model:    &model,
	}
	newService := func(admin bool) (*VoiceSetService, *fakeCastStore, *Cast) {
		rows := &fakeCastStore{}
		cast := NewCast(newTestService(newFakeStore(), mainVoice(&fakeSynth{})), rows, func(int) Synthesizer { return &fakeSynth{} }, zap.NewNop())
		return NewVoiceSetService(fakeAdmins{admin: admin}, engine, rows, cast, zap.NewNop()), rows, cast
	}
	user := uuid.New()

	t.Run("only a super-admin may change a voice", func(t *testing.T) {
		svc, rows, _ := newService(false)
		if _, err := svc.Set(ctx, user, "f1", 888753760); !errors.Is(err, exceptions.ErrForbidden) {
			t.Fatalf("err = %v, want forbidden", err)
		}
		if len(rows.saved) != 0 {
			t.Fatal("saved a voice for a non-admin")
		}
	})

	t.Run("a style the engine lacks is refused", func(t *testing.T) {
		svc, _, _ := newService(true)
		if _, err := svc.Set(ctx, user, "f1", 42); !errors.Is(err, exceptions.ErrAudioStyleNotFound) {
			t.Fatalf("err = %v, want style not found", err)
		}
	})

	t.Run("alt is not a cast key", func(t *testing.T) {
		svc, _, _ := newService(true)
		if _, err := svc.Set(ctx, user, common.AUDIO_VOICE_ALT, 888753760); !errors.Is(err, exceptions.ErrAudioVoiceKeyUnknown) {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("saves the style with its model and credit, and takes effect at once", func(t *testing.T) {
		svc, rows, cast := newService(true)
		cast.Voices(ctx) // cached before the change
		v, err := svc.Set(ctx, user, "f1", 888753760)
		if err != nil {
			t.Fatal(err)
		}
		if v.Credit != "AivisSpeech:まお" || v.ModelUUID == nil || *v.ModelUUID != model || *v.UpdatedBy != user || len(rows.saved) != 1 {
			t.Fatalf("saved %+v", v)
		}
		if p := cast.Voice(ctx, "f1").voices[0].Profile; p != "aivisspeech-888753760" {
			t.Fatalf("profile = %q, want the new voice", p)
		}
	})
}

func TestJobService(t *testing.T) {
	ctx := context.Background()
	svc := NewJobService(fakeAdmins{admin: true}, nil, zap.NewNop())
	if _, err := svc.Create(ctx, uuid.New(), CreateJobRequest{Kind: common.AudioJobInstallModel}); !errors.Is(err, exceptions.ErrAudioJobInvalid) {
		t.Fatalf("err = %v, want an install job without a model refused", err)
	}
	denied := NewJobService(fakeAdmins{}, nil, zap.NewNop())
	if _, err := denied.List(ctx, uuid.New()); !errors.Is(err, exceptions.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func newRunner(jobs *fakeJobStore, synth *fakeSynth, installed *[]string) *JobRunner {
	main := newTestService(newFakeStore(), mainVoice(synth))
	cast := NewCast(main, &fakeCastStore{voices: []CastVoice{{Key: "f1", Engine: "aivisspeech", StyleID: 1}}},
		func(int) Synthesizer { return synth }, zap.NewNop())
	r := NewJobRunner(jobs, cast, func(_ context.Context, id string) error {
		*installed = append(*installed, id)
		return nil
	}, zap.NewNop())
	r.wait = 0
	return r
}

func TestJobRunner(t *testing.T) {
	ctx := context.Background()
	texts := []CourseText{{Voice: "f1", Text: "はい。"}, {Voice: "f1", Text: "はい。"}, {Voice: "main", Text: "いぬ"}, {Voice: "main", Text: "ねこ"}}

	t.Run("generates every missing clip of every voice", func(t *testing.T) {
		job := &Job{ID: uuid.New(), Kind: common.AudioJobGenerate}
		jobs := &fakeJobStore{queue: []*Job{job}, texts: texts}
		synth := &fakeSynth{}
		ran, err := newRunner(jobs, synth, &[]string{}).RunNext(ctx)
		if err != nil || !ran {
			t.Fatalf("ran = %v, err = %v", ran, err)
		}
		if jobs.finished[job.ID] != common.AudioJobDone || job.Total != 3 || job.Done != 3 || synth.count() != 3 {
			t.Fatalf("status %s, total %d, done %d, synth calls %d", jobs.finished[job.ID], job.Total, job.Done, synth.count())
		}
	})

	t.Run("generates only the job's voice", func(t *testing.T) {
		key := "f1"
		job := &Job{ID: uuid.New(), Kind: common.AudioJobGenerate, VoiceKey: &key}
		jobs := &fakeJobStore{queue: []*Job{job}, texts: texts}
		if _, err := newRunner(jobs, &fakeSynth{}, &[]string{}).RunNext(ctx); err != nil {
			t.Fatal(err)
		}
		if job.Total != 1 {
			t.Fatalf("total = %d, want the one f1 text", job.Total)
		}
	})

	t.Run("a cancelled job stops and is not marked finished", func(t *testing.T) {
		job := &Job{ID: uuid.New(), Kind: common.AudioJobGenerate}
		jobs := &fakeJobStore{queue: []*Job{job}, texts: texts, cancelAfter: 1}
		synth := &fakeSynth{}
		if _, err := newRunner(jobs, synth, &[]string{}).RunNext(ctx); err != nil {
			t.Fatal(err)
		}
		if _, ok := jobs.finished[job.ID]; ok || synth.count() != 0 {
			t.Fatalf("finished = %v, synth calls %d; want stopped before generating", jobs.finished, synth.count())
		}
	})

	t.Run("a voice that keeps failing ends the job as failed", func(t *testing.T) {
		job := &Job{ID: uuid.New(), Kind: common.AudioJobGenerate}
		jobs := &fakeJobStore{queue: []*Job{job}, texts: texts}
		if _, err := newRunner(jobs, &fakeSynth{fail: true}, &[]string{}).RunNext(ctx); err != nil {
			t.Fatal(err)
		}
		if jobs.finished[job.ID] != common.AudioJobFailed || job.Failed != 3 || jobs.errors[job.ID] == "" {
			t.Fatalf("status %s, failed %d, error %q", jobs.finished[job.ID], job.Failed, jobs.errors[job.ID])
		}
	})

	t.Run("installs a model", func(t *testing.T) {
		model := uuid.New()
		job := &Job{ID: uuid.New(), Kind: common.AudioJobInstallModel, ModelUUID: &model}
		jobs := &fakeJobStore{queue: []*Job{job}}
		var installed []string
		if _, err := newRunner(jobs, &fakeSynth{}, &installed).RunNext(ctx); err != nil {
			t.Fatal(err)
		}
		if jobs.finished[job.ID] != common.AudioJobDone || len(installed) != 1 || installed[0] != model.String() {
			t.Fatalf("status %s, installed %v", jobs.finished[job.ID], installed)
		}
	})

	t.Run("an empty queue runs nothing", func(t *testing.T) {
		ran, err := newRunner(&fakeJobStore{}, &fakeSynth{}, &[]string{}).RunNext(ctx)
		if err != nil || ran {
			t.Fatalf("ran = %v, err = %v", ran, err)
		}
	})
}

func TestSpokenText(t *testing.T) {
	if got := spokenText("私[わたし] は 学生[がくせい] です。"); got != "私は学生です。" {
		t.Fatalf("got %q", got)
	}
}

func TestModelOfSpeaker(t *testing.T) {
	list := `{"a59cb814-0083-4369-8542-f51a29e72af7":{"manifest":{"name":"まお"},"speakers":[{"speaker":{"speaker_uuid":"s1"},"speaker_info":{"portrait":"...."}}]},
	"22e8ed77-94fe-4ef2-871f-a86f94e9a579":{"speakers":[{"speaker":{"speaker_uuid":"s2"}}]}}`
	id, err := modelOfSpeaker(strings.NewReader(list), "s2")
	if err != nil || id == nil || id.String() != "22e8ed77-94fe-4ef2-871f-a86f94e9a579" {
		t.Fatalf("id = %v, err = %v", id, err)
	}
	none, err := modelOfSpeaker(strings.NewReader(list), "s9")
	if err != nil || none != nil {
		t.Fatalf("id = %v, err = %v; want none", none, err)
	}
}

// delayedSynth takes `delay` per sentence, or fails when its context ends first.
type delayedSynth struct{ delay time.Duration }

func (s delayedSynth) Synthesize(ctx context.Context, text string) ([]byte, error) {
	select {
	case <-time.After(s.delay):
		return []byte("RIFF....WAVE" + text), nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestWithTimeouts(t *testing.T) {
	ctx := context.Background()
	admin := Requester{UserID: uuid.New(), Admin: true}
	slow := Voice{Profile: "aivisspeech-1", Synth: delayedSynth{delay: 50 * time.Millisecond}}

	t.Run("a render longer than the wait still finishes, is stored, and is made once", func(t *testing.T) {
		store := newFakeStore()
		calls := &countingSynth{inner: delayedSynth{delay: 50 * time.Millisecond}}
		svc := newTestService(store, Voice{Profile: "aivisspeech-1", Synth: calls}).WithTimeouts(10*time.Millisecond, time.Second)
		for range 2 {
			resp, err := svc.Speak(ctx, admin, []string{"はい"})
			if err != nil {
				t.Fatal(err)
			}
			if !resp.Pending {
				t.Fatal("want the request answered before the slow render ends")
			}
		}
		time.Sleep(150 * time.Millisecond)
		resp, err := svc.Find(ctx, []string{"はい"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Items[0].Segments[0].URL == "" || calls.count() != 1 {
			t.Fatalf("url %q, renders %d; want the clip stored from a single render", resp.Items[0].Segments[0].URL, calls.count())
		}
	})

	t.Run("a longer timeout lets the slow engine finish, and its voices share it", func(t *testing.T) {
		main := newTestService(newFakeStore(), mainVoice(&fakeSynth{})).WithTimeouts(time.Second, time.Second)
		svc := main.WithVoices([]Voice{slow})
		resp, err := svc.Speak(ctx, admin, []string{"はい"})
		if err != nil {
			t.Fatal(err)
		}
		if resp.Pending {
			t.Fatal("want the clip made within the longer timeout")
		}
	})
}

type countingSynth struct {
	mu    sync.Mutex
	n     int
	inner Synthesizer
}

func (c *countingSynth) Synthesize(ctx context.Context, text string) ([]byte, error) {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return c.inner.Synthesize(ctx, text) //nolint:wrapcheck // a test double
}

func (c *countingSynth) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}
