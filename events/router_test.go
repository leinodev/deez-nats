package events

import (
	"slices"
	"testing"

	"github.com/leinodev/deez-nats/marshaller"
)

func TestEventRouterDFS(t *testing.T) {
	defaultOpts := EventHandlerOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
		JetStream: JetStreamEventOptions{
			Enabled: true,
		},
	}

	router := newEventRouter("app", defaultOpts)

	router.AddEventHandler("health", func(ctx EventContext) error {
		return nil
	})

	entityGroup := router.Group("entity")
	entityGroup.Use(func(next EventHandleFunc) EventHandleFunc {
		return next
	})

	entityGroup.AddEventHandler("created", func(ctx EventContext) error {
		return nil
	})

	logGroup := entityGroup.Group("log")
	logGroup.AddEventHandler("added", func(ctx EventContext) error {
		return nil
	}, WithHandlerJetStream(
		WithJSEnabled(true),
		WithJSPull(true),
		WithJSDurable("log-added"),
	))

	routes := router.dfs()

	subjects := make([]string, 0, len(routes))
	for _, route := range routes {
		subjects = append(subjects, route.subject)
	}

	expected := []string{
		"app.health",
		"app.entity.created",
		"app.entity.log.added",
	}
	slices.Sort(subjects)
	slices.Sort(expected)
	if len(subjects) != len(expected) {
		t.Fatalf("unexpected number of subjects: %v", subjects)
	}
	for i := range subjects {
		if subjects[i] != expected[i] {
			t.Fatalf("unexpected subjects: got %v want %v", subjects, expected)
		}
	}

	for _, route := range routes {
		if route.options.Marshaller == nil {
			t.Fatalf("marshaller must not be nil for route %s", route.subject)
		}
		if route.subject == "app.entity.log.added" {
			if !route.options.JetStream.Pull {
				t.Fatalf("expected pull consumer for %s", route.subject)
			}
			if route.options.JetStream.Durable != "log-added" {
				t.Fatalf("unexpected durable: %s", route.options.JetStream.Durable)
			}
		}
	}
}
