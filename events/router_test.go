package events

import (
	"slices"
	"testing"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestEventRouterDFSCore(t *testing.T) {
	defaultOpts := CoreEventHandlerOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
	}

	router := newEventRouter[*nats.Msg, nats.AckOpt, CoreEventHandlerOptions, MiddlewareFunc[*nats.Msg, nats.AckOpt]]("app", defaultOpts)

	router.AddEventHandler("health", func(ctx EventContext[*nats.Msg, nats.AckOpt]) error {
		return nil
	})

	entityGroup := router.Group("entity")
	entityGroup.Use(func(next HandlerFunc[*nats.Msg, nats.AckOpt]) HandlerFunc[*nats.Msg, nats.AckOpt] {
		return next
	})

	entityGroup.AddEventHandler("created", func(ctx EventContext[*nats.Msg, nats.AckOpt]) error {
		return nil
	})

	logGroup := entityGroup.Group("log")
	logGroup.AddEventHandler("added", func(ctx EventContext[*nats.Msg, nats.AckOpt]) error {
		return nil
	}, func(opts *CoreEventHandlerOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
		opts.Queue = "test-queue"
	})

	routes := router.dfs()

	subjects := make([]string, 0, len(routes))
	for _, route := range routes {
		subjects = append(subjects, route.Name)
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
		opts := route.Options
		if opts.Marshaller == nil {
			t.Fatalf("marshaller must not be nil for route %s", route.Name)
		}
		if route.Name == "app.entity.log.added" {
			if opts.Queue != "test-queue" {
				t.Fatalf("unexpected queue: %s", opts.Queue)
			}
		}
	}
}

func TestEventRouterDFSJetStream(t *testing.T) {
	defaultOpts := JetStreamEventHandlerOptions{
		Marshaller: marshaller.DefaultJsonMarshaller,
	}

	router := newEventRouter[jetstream.Msg, any, JetStreamEventHandlerOptions, MiddlewareFunc[jetstream.Msg, any]]("app", defaultOpts)

	router.AddEventHandler("health", func(ctx EventContext[jetstream.Msg, any]) error {
		return nil
	})

	entityGroup := router.Group("entity")
	entityGroup.Use(func(next HandlerFunc[jetstream.Msg, any]) HandlerFunc[jetstream.Msg, any] {
		return next
	})

	entityGroup.AddEventHandler("created", func(ctx EventContext[jetstream.Msg, any]) error {
		return nil
	})

	logGroup := entityGroup.Group("log")
	logGroup.AddEventHandler("added", func(ctx EventContext[jetstream.Msg, any]) error {
		return nil
	}, func(opts *JetStreamEventHandlerOptions) {
		opts.Marshaller = marshaller.DefaultJsonMarshaller
	})

	routes := router.dfs()

	subjects := make([]string, 0, len(routes))
	for _, route := range routes {
		subjects = append(subjects, route.Name)
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
		opts := route.Options
		if opts.Marshaller == nil {
			t.Fatalf("marshaller must not be nil for route %s", route.Name)
		}
	}
}
