package lifecycle

import (
	"errors"
	"sync"
)

var ErrAlreadyStarted = errors.New("already started")

type Manager struct {
	mu      sync.Mutex
	started bool
}

func NewManager() *Manager {
	return &Manager{
		started: false,
	}
}

func (m *Manager) MarkAsStarted() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return ErrAlreadyStarted
	}

	m.started = true
	return nil
}

func (m *Manager) IsStarted() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.started
}

func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.started = false
}
