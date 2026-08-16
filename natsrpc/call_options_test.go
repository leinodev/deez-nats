package natsrpc

import (
	"sync"
	"testing"

	"github.com/nats-io/nats.go"
)

// Per-call headers must not touch the map owned by defaults: a shallow copy of
// CallOptions shares it, so concurrent calls raced on one map and the process
// died with "concurrent map writes". Run with -race to see the original bug.
func TestCallOptionsDoNotShareDefaultHeaders(t *testing.T) {
	defaults := NewRPCOptions(WithDefaultCallOptions(WithCallHeader("X-Trace", "root")))

	var wait sync.WaitGroup
	for worker := range 32 {
		wait.Add(1)
		go func() {
			defer wait.Done()

			callOpts := defaults.DefaultCallOptions
			callOpts.Headers = cloneHeaders(defaults.DefaultCallOptions.Headers)
			WithCallHeader("X-Worker", string(rune('a'+worker%26)))(&callOpts)
		}()
	}
	wait.Wait()

	if got := defaults.DefaultCallOptions.Headers.Values("X-Worker"); len(got) != 0 {
		t.Fatalf("вызов записал заголовок в общие defaults: %v", got)
	}
	if got := defaults.DefaultCallOptions.Headers.Get("X-Trace"); got != "root" {
		t.Fatalf("общий заголовок defaults изменён: %q", got)
	}
}

// Значения тоже копируются: nats.Header хранит срез, и общий срез вернул бы ту
// же гонку уровнем ниже.
func TestCloneHeadersCopiesValues(t *testing.T) {
	source := nats.Header{"X-Trace": []string{"root"}}

	clone := cloneHeaders(source)
	clone.Add("X-Trace", "child")

	if got := source.Values("X-Trace"); len(got) != 1 {
		t.Fatalf("исходные значения изменены: %v", got)
	}
	if cloneHeaders(nil) != nil {
		t.Fatal("пустые заголовки должны давать nil, а не пустую карту")
	}
}
