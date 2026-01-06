// Package sessions provides infrastructure implementations for session storage
package sessions

import (
	"sync"
	"time"

	"github.com/device-management-toolkit/console/redfish/internal/entity"
	"github.com/device-management-toolkit/console/redfish/internal/usecase/sessions"
)

// InMemoryRepository is an in-memory implementation of sessions.Repository.
type InMemoryRepository struct {
	sessions      map[string]*entity.Session
	tokenIndex    map[string]string // token -> sessionID
	mu            sync.RWMutex
	cleanupTicker *time.Ticker
	done          chan bool
}

// NewInMemoryRepository creates a new in-memory session repository.
func NewInMemoryRepository(cleanupInterval time.Duration) *InMemoryRepository {
	repo := &InMemoryRepository{
		sessions:      make(map[string]*entity.Session),
		tokenIndex:    make(map[string]string),
		cleanupTicker: time.NewTicker(cleanupInterval),
		done:          make(chan bool),
	}

	// Start background cleanup goroutine
	go repo.cleanupLoop()

	return repo
}

// cleanupLoop periodically removes expired sessions.
func (r *InMemoryRepository) cleanupLoop() {
	for {
		select {
		case <-r.cleanupTicker.C:
			if _, err := r.DeleteExpired(); err != nil {
				// Log error but continue cleanup cycle
				_ = err
			}
		case <-r.done:
			return
		}
	}
}

// Stop stops the cleanup goroutine.
func (r *InMemoryRepository) Stop() {
	r.cleanupTicker.Stop()

	r.done <- true
}

// Create stores a new session.
func (r *InMemoryRepository) Create(session *entity.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.sessions[session.ID] = session
	r.tokenIndex[session.Token] = session.ID

	return nil
}

// Get retrieves a session by ID.
func (r *InMemoryRepository) Get(id string) (*entity.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.sessions[id]
	if !exists {
		return nil, sessions.ErrSessionNotFound
	}

	if session.IsExpired() {
		return nil, sessions.ErrSessionExpired
	}

	return session, nil
}

// GetByToken retrieves a session by token.
func (r *InMemoryRepository) GetByToken(token string) (*entity.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sessionID, exists := r.tokenIndex[token]
	if !exists {
		return nil, sessions.ErrSessionNotFound
	}

	session, exists := r.sessions[sessionID]
	if !exists {
		return nil, sessions.ErrSessionNotFound
	}

	if session.IsExpired() {
		return nil, sessions.ErrSessionExpired
	}

	return session, nil
}

// Delete removes a session.
func (r *InMemoryRepository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[id]
	if !exists {
		return sessions.ErrSessionNotFound
	}

	delete(r.tokenIndex, session.Token)
	delete(r.sessions, id)

	return nil
}

// List returns all active sessions.
func (r *InMemoryRepository) List() ([]*entity.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	activeSessions := make([]*entity.Session, 0, len(r.sessions))

	for _, session := range r.sessions {
		if !session.IsExpired() {
			activeSessions = append(activeSessions, session)
		}
	}

	return activeSessions, nil
}

// DeleteExpired removes all expired sessions.
func (r *InMemoryRepository) DeleteExpired() (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	count := 0

	for id, session := range r.sessions {
		if session.IsExpired() {
			delete(r.tokenIndex, session.Token)
			delete(r.sessions, id)

			count++
		}
	}

	return count, nil
}
