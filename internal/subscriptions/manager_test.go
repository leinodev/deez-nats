package subscriptions

import "testing"

type trackingSubscription struct {
	drains       int
	unsubscribes int
}

func (s *trackingSubscription) Drain() error {
	s.drains++
	return nil
}

func (s *trackingSubscription) Unsubscribe() error {
	s.unsubscribes++
	return nil
}

func TestTrackerUnsubscribeAllDrainedSubscriptions(t *testing.T) {
	tracker := NewTracker()
	subscriptions := []*trackingSubscription{{}, {}, {}}
	for _, subscription := range subscriptions {
		tracker.Track(subscription)
	}

	tracker.Drain()
	tracker.Unsubscribe()
	tracker.Unsubscribe()

	for i, subscription := range subscriptions {
		if subscription.drains != 1 {
			t.Errorf("subscription %d: drains = %d, want 1", i, subscription.drains)
		}
		if subscription.unsubscribes != 1 {
			t.Errorf("subscription %d: unsubscribes = %d, want 1", i, subscription.unsubscribes)
		}
	}
	if len(tracker.subs) != 0 {
		t.Fatalf("tracked subscriptions = %d, want 0", len(tracker.subs))
	}
}
