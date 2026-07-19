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

	kept := m.subs[:0]
	for _, sub := range m.subs {
		if !sub.Dirty {
			kept = append(kept, sub)
			continue
		}
		_ = sub.Sub.Unsubscribe()
	}
	clear(m.subs[len(kept):])
	m.subs = kept
}
