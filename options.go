package natsrpc

import (
	"log"
	"time"
)

type OptionFunc func(*Options)
type HandlerOptionFunc func(*HandlerOptions)
type RPCCallOptionFunc func(*RPCCallOptions)

// Defaults

func GetDefaultOptions() Options {
	return Options{
		BaseName: "default",
		HandlerOpts: HandlerOptions{
			HandlerTimeout: time.Minute,
		},
		RPCCallOpts: RPCCallOptions{
			Timeout: time.Minute,
		},
	}
}

func DefaultErrHandler(c NatsRPCContext, err error) {
	log.Printf("error in handler %s: %s", c.GetPath(), err.Error())
}

// Options

type Options struct {
	BaseName string

	HandlerOpts HandlerOptions
	RPCCallOpts RPCCallOptions
}
type HandlerOptions struct {
	HandlerTimeout time.Duration
}
type RPCCallOptions struct {
	Timeout time.Duration
}

func WithHandlerTimeout(timeout time.Duration) HandlerOptionFunc {
	return func(ho *HandlerOptions) {
		ho.HandlerTimeout = timeout
	}
}
func WithRPCTimeout(timeout time.Duration) RPCCallOptionFunc {
	return func(ho *RPCCallOptions) {
		ho.Timeout = timeout
	}
}

func WithBaseName(name string) OptionFunc {
	return func(o *Options) {
		o.BaseName = name
	}
}
