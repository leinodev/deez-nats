package natsrpc

import (
	"log"
	"time"
)

type OptionFunc func(*NatsRPCOptions)
type HandlerOptionFunc func(*RPCHandlerOptions)
type RPCCallOptionFunc func(*RPCCallOptions)

// Defaults

func GetDefaultOptions() NatsRPCOptions {
	return NatsRPCOptions{
		BaseName: "default",
		RPCHandlerOpts: RPCHandlerOptions{
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

type NatsRPCOptions struct {
	BaseName string

	RPCHandlerOpts RPCHandlerOptions
	RPCCallOpts    RPCCallOptions
}
type RPCHandlerOptions struct {
	HandlerTimeout time.Duration
}
type RPCCallOptions struct {
	Timeout time.Duration
}

func WithHandlerTimeout(timeout time.Duration) HandlerOptionFunc {
	return func(ho *RPCHandlerOptions) {
		ho.HandlerTimeout = timeout
	}
}
func WithRPCTimeout(timeout time.Duration) RPCCallOptionFunc {
	return func(ho *RPCCallOptions) {
		ho.Timeout = timeout
	}
}

func WithBaseName(name string) OptionFunc {
	return func(o *NatsRPCOptions) {
		o.BaseName = name
	}
}
