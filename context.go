package natsrpc

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type NatsRPCContext interface {
	context.Context
	RequestJSON(value interface{}) error
	RespondJSON(val interface{}) error
	ResponseWritten() bool
	Ok(value interface{}) error
	Error(err string) error
	GetPath() string

	mustError(err string) error
}

type natsRPCContextImpl struct {
	ctx             context.Context
	msg             *nats.Msg
	route           *Route
	responseWritten bool
}

func NewNatsRPCContext(ctx context.Context, msg *nats.Msg, route *Route) NatsRPCContext {
	return &natsRPCContextImpl{
		ctx:   ctx,
		msg:   msg,
		route: route,
	}
}

// Default context methods

func (c *natsRPCContextImpl) Done() <-chan struct{} {
	return c.ctx.Done()
}
func (c *natsRPCContextImpl) Deadline() (deadline time.Time, ok bool) {
	return c.ctx.Deadline()
}
func (c *natsRPCContextImpl) Err() error {
	return c.ctx.Err()
}
func (c *natsRPCContextImpl) Value(key any) any {
	return c.ctx.Value(key)
}

// Customm methods
func (c *natsRPCContextImpl) ResponseWritten() bool {
	return c.responseWritten
}
func (c *natsRPCContextImpl) RequestJSON(value interface{}) error {
	return json.Unmarshal(c.msg.Data, value)
}
func (c *natsRPCContextImpl) RespondJSON(value interface{}) error {
	if c.ResponseWritten() {
		return fmt.Errorf("cannot answer one request more than once")
	}
	if c.Err() != nil {
		return fmt.Errorf("cannot answer: %v", c.Err())
	}
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	c.responseWritten = true
	return c.msg.Respond(data)
}
func (c *natsRPCContextImpl) Ok(value interface{}) error {
	val := NatsRPCResponse[interface{}]{
		Error: nil,
		Data:  &value,
	}
	return c.RespondJSON(val)
}
func (c *natsRPCContextImpl) Error(err string) error {
	val := NatsRPCResponse[interface{}]{
		Error: &NatsError{
			Message: err,
		},
	}
	return c.RespondJSON(val)
}
func (c *natsRPCContextImpl) GetPath() string {
	return c.route.Path
}

func (c *natsRPCContextImpl) mustError(err string) error {
	val := NatsRPCResponse[interface{}]{
		Error: &NatsError{
			Message: err,
		},
	}
	data, e := json.Marshal(val)
	if e != nil {
		return e
	}
	return c.msg.Respond(data)
}
