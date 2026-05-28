package natsevents

import "testing"

func TestJetStreamPushDurable(t *testing.T) {
	t.Parallel()
	if got := jetStreamPushDurable("", "events.users.created", 2); got != "" {
		t.Fatalf("empty base: got %q", got)
	}
	if got := jetStreamPushDurable("gosearch-users", "events.users.created", 1); got != "gosearch-users" {
		t.Fatalf("single route: got %q", got)
	}
	if got := jetStreamPushDurable("gosearch-users", "events.users.created", 2); got != "gosearch-users_events_users_created" {
		t.Fatalf("multi route: got %q", got)
	}
}

func TestSanitizeJetStreamDurable(t *testing.T) {
	t.Parallel()
	if got := sanitizeJetStreamDurable("  gosearch-users  "); got != "gosearch-users" {
		t.Fatalf("got %q", got)
	}
	if got := sanitizeJetStreamDurable("bad name!"); got != "bad_name" {
		t.Fatalf("got %q", got)
	}
}
