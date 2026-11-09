package audio

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/fluentfox/api/internal/common"
)

type fakeStore struct {
	mu      sync.Mutex
	objects map[string][]byte
	heads   int
	puts    int
}

func newFakeStore(keys ...string) *fakeStore {
	s := &fakeStore{objects: map[string][]byte{}}
	for _, k := range keys {
		s.objects[k] = []byte("stored")
	}
	return s
}

func (s *fakeStore) Exists(_ context.Context, key string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.heads++
	_, ok := s.objects[key]
	return ok, nil
}

func (s *fakeStore) Put(_ context.Context, key, _, _ string, body []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.puts++
	s.objects[key] = body
	return nil
}

type fakeSynth struct {
	mu    sync.Mutex
	calls []string
	fail  bool
}

func (f *fakeSynth) Synthesize(_ context.Context, text string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, text)
	if f.fail {
		return nil, errors.New("engine down")
	}
	return []byte("RIFF....WAVE" + text), nil
}

func (f *fakeSynth) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

type fakeCastStore struct {
	mu     sync.Mutex
	voices []CastVoice
	reads  int
	saved  []CastVoice
	err    error
}

func (f *fakeCastStore) ListCastVoices(context.Context) ([]CastVoice, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	return append([]CastVoice(nil), f.voices...), f.err
}

func (f *fakeCastStore) SaveCastVoice(_ context.Context, v *CastVoice) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, *v)
	for i := range f.voices {
		if f.voices[i].Key == v.Key {
			f.voices[i] = *v
			return nil
		}
	}
	f.voices = append(f.voices, *v)
	return nil
}

type fakeClipIndex struct {
	mu       sync.Mutex
	keys     map[string]bool
	recorded []ClipRecord
	lookups  int
}

func newFakeClipIndex(keys ...string) *fakeClipIndex {
	f := &fakeClipIndex{keys: map[string]bool{}}
	for _, k := range keys {
		f.keys[k] = true
	}
	return f
}

func (f *fakeClipIndex) ExistingClips(_ context.Context, keys []string) (map[string]bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lookups++
	found := map[string]bool{}
	for _, k := range keys {
		if f.keys[k] {
			found[k] = true
		}
	}
	return found, nil
}

func (f *fakeClipIndex) RecordClip(_ context.Context, c *ClipRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.keys[c.ObjectKey] = true
	f.recorded = append(f.recorded, *c)
	return nil
}

type fakeAdmins struct{ admin bool }

func (f fakeAdmins) IsSuperAdmin(context.Context, uuid.UUID) (bool, error) { return f.admin, nil }

type fakeEngine struct {
	speakers []EngineSpeaker
	model    *uuid.UUID
	synth    *fakeSynth
	down     bool
}

func (e *fakeEngine) Name() string     { return "aivisspeech" }
func (e *fakeEngine) Configured() bool { return true }
func (e *fakeEngine) Version(context.Context) (string, error) {
	if e.down {
		return "", errors.New("engine down")
	}
	return "1.2.0", nil
}
func (e *fakeEngine) Speakers(context.Context) ([]EngineSpeaker, error) {
	if e.down {
		return nil, errors.New("engine down")
	}
	return e.speakers, nil
}
func (e *fakeEngine) ModelOf(context.Context, string) (*uuid.UUID, error) { return e.model, nil }
func (e *fakeEngine) Synth(int) Synthesizer                               { return e.synth }

type fakeJobStore struct {
	mu       sync.Mutex
	queue    []*Job
	texts    []CourseText
	saves    int
	finished map[uuid.UUID]common.AudioJobStatus
	errors   map[uuid.UUID]string
	// cancelAfter cancels the running job once this many saves have happened (0: never).
	cancelAfter int
	running     *Job
}

func (f *fakeJobStore) RequeueStaleJobs(context.Context, time.Duration) error { return nil }

func (f *fakeJobStore) ClaimJob(context.Context) (*Job, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.queue) == 0 {
		return nil, nil
	}
	j := f.queue[0]
	f.queue = f.queue[1:]
	j.Status = common.AudioJobRunning
	f.running = j
	return j, nil
}

func (f *fakeJobStore) SaveJobProgress(context.Context, *Job) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saves++
	if f.cancelAfter > 0 && f.saves >= f.cancelAfter {
		return errJobStopped
	}
	return nil
}

func (f *fakeJobStore) FinishJob(_ context.Context, j *Job, status common.AudioJobStatus, jobErr string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.finished == nil {
		f.finished, f.errors = map[uuid.UUID]common.AudioJobStatus{}, map[uuid.UUID]string{}
	}
	f.finished[j.ID], f.errors[j.ID] = status, jobErr
	return nil
}

func (f *fakeJobStore) CourseTexts(context.Context) ([]CourseText, error) { return f.texts, nil }
