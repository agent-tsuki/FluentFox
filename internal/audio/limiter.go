package audio

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// generationLimiter caps how many new clips one user can have generated per
// window, so the endpoint cannot be used as a free text-to-speech service.
// Clips that already exist are never counted. In memory, so each API instance
// counts on its own.
type generationLimiter struct {
	max    int
	window time.Duration
	now    func() time.Time

	mu    sync.Mutex
	users map[uuid.UUID]*generationWindow
}

type generationWindow struct {
	count int
	ends  time.Time
}

func newGenerationLimiter(maxPerWindow int, window time.Duration) *generationLimiter {
	return &generationLimiter{max: maxPerWindow, window: window, now: time.Now, users: make(map[uuid.UUID]*generationWindow)}
}

// take reserves one generation for the user, or reports that the window is full.
func (l *generationLimiter) take(userID uuid.UUID) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	w, ok := l.users[userID]
	if !ok || now.After(w.ends) {
		l.dropExpired(now)
		w = &generationWindow{ends: now.Add(l.window)}
		l.users[userID] = w
	}
	if w.count >= l.max {
		return false
	}
	w.count++
	return true
}

// Called with mu held when a window opens, so the map only holds active users.
func (l *generationLimiter) dropExpired(now time.Time) {
	for id, w := range l.users {
		if now.After(w.ends) {
			delete(l.users, id)
		}
	}
}
