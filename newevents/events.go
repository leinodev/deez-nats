package newevents

import "errors"

var (
	ErrJetStreamPullRequiresDurable = errors.New("jetstream pull consumer requires durable name")
	ErrEmptySubject                 = errors.New("empty subject")
)
