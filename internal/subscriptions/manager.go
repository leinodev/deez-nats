package subscriptions

import (
	"sync"

	"github.com/leinodev/deez-nats/internal/provider"
)

type sub struct {
	Sub   provider.TransportSubscription
	Dirty bool
}

type Tracker struct {
	mu   sync.Mutex
	subs []sub
}

func NewTracker() *Tracker {
	return &Tracker{
		subs: make([]sub, 0),
	}
}

func (m *Tracker) Track(s provider.TransportSubscription) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subs = append(m.subs, sub{
		Sub:   s,
		Dirty: false,
	})
}

func (m *Tracker) Drain() {
	m.mu.Lock()
	subs := m.subs
	m.subs = nil
	m.mu.Unlock()

	for i, sub := range subs {
		subs[i].Dirty = true
		_ = sub.Sub.Drain()
	}
}

func (m *Tracker) Unsubscribe() {
	m.mu.Lock()
	subs := m.subs
	m.subs = nil
	m.mu.Unlock()

	for _, sub := range subs {
		if !sub.Dirty {
			continue
		}
		_ = sub.Sub.Unsubscribe()
	}
}
