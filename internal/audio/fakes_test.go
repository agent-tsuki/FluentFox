package audio

import (
	"context"
	"errors"
	"sync"
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
