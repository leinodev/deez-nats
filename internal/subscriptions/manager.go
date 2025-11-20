package subscriptions

import (
	"sync"
)

type sub struct {
	Sub   Subscription
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

func (m *Tracker) Track(s Subscription) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.subs = append(m.subs, sub{
		Sub:   s,
		Dirty: false,
	})
}

func (m *Tracker) Drain() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, sub := range m.subs {
		m.subs[i].Dirty = true
		_ = sub.Sub.Drain()
	}
}

func (m *Tracker) Unsubscribe() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for idx, sub := range m.subs {
		if !sub.Dirty {
			continue
		}
		_ = sub.Sub.Unsubscribe()
		// Remove from subs
		m.subs = append(m.subs[:idx], m.subs[idx+1:]...)
	}
}
