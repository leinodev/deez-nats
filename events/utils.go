package events

import (
	"errors"

	"github.com/nats-io/nats.go"
)

var (
	ErrJetStreamPullRequiresDurable = errors.New("jetstream pull consumer requires durable name")
	ErrEmptySubject                 = errors.New("empty subject")
)

func mergeHeaders(defaultHeaders, explicitHeaders nats.Header) nats.Header {
	if len(defaultHeaders) == 0 && len(explicitHeaders) == 0 {
		return nil
	}
	if defaultHeaders == nil {
		return explicitHeaders
	}
	if explicitHeaders == nil {
		return defaultHeaders
	}

	merged := make(nats.Header)

	for k, v := range defaultHeaders {
		merged[k] = append([]string(nil), v...)
	}
	for k, v := range explicitHeaders {
		merged[k] = append([]string(nil), v...)
	}

	return merged
}
