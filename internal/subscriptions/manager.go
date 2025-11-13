package subscriptions

import (
	"sync"

	"github.com/nats-io/nats.go"
)

type Manager struct {
	mu   sync.Mutex
	subs []*nats.Subscription
}

func NewManager() *Manager {
	return &Manager{
		subs: make([]*nats.Subscription, 0),
	}
}

func (m *Manager) Track(sub *nats.Subscription) {
	if sub == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.subs = append(m.subs, sub)
}

func (m *Manager) Cleanup() {
	m.mu.Lock()
	subs := m.subs
	m.subs = nil
	m.mu.Unlock()

	for _, sub := range subs {
		_ = sub.Drain()
	}
}

func (m *Manager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.subs)
}
