package natsrpc_test

import (
	"testing"
	"time"

	"github.com/TexHik620953/natsrpc-go"
	"github.com/nats-io/nats.go"
)

func TestRPCString(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Error(err)
		return
	}

	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))
	if err != nil {
		t.Error(err)
		return
	}

	nrpc.AddRPC("test", func(c natsrpc.NatsRPCContext) error {
		return c.Ok("Some string")
	}, natsrpc.WithHandlerTimeout(time.Second*10))

	err = nrpc.StartWithContext(t.Context())
	if err != nil {
		t.Error(err)
		return
	}

	var resp string

	err = nrpc.CallRPC("testapp.test", nil, &resp, natsrpc.WithRPCTimeout(time.Second*5))
	if err != nil {
		t.Error(err)
		return
	}

	if resp != "Some string" {
		t.Error("invalid response")
		return
	}
}

type testStruct struct {
	F32 float32
	F64 float64
	I32 int32
	I64 int64
	S   string
	M   map[string]float32
	A   []float32
}

func TestRPCStruct(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Error(err)
		return
	}

	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))
	if err != nil {
		t.Error(err)
		return
	}

	nrpc.AddRPC("double", func(c natsrpc.NatsRPCContext) error {
		var msg testStruct

		err := c.RequestJSON(&msg)
		if err != nil {
			return err
		}

		msg.F32 *= 2
		msg.F64 *= 2
		msg.I32 *= 2
		msg.I64 *= 2
		msg.S = msg.S + msg.S
		for k, v := range msg.M {
			msg.M[k] = v * 2.0
		}
		for i, v := range msg.A {
			msg.A[i] = v * 2.0
		}

		return c.Ok(&msg)
	}, natsrpc.WithHandlerTimeout(time.Second*10))

	err = nrpc.StartWithContext(t.Context())
	if err != nil {
		t.Error(err)
		return
	}

	var resp testStruct
	err = nrpc.CallRPC("testapp.double", &testStruct{
		F32: 1.7,
		F64: 1.8,
		I32: 556,
		I64: -775,
		S:   "Hello ",
		M:   map[string]float32{"A": 1.1, "B": 1.2},
		A:   []float32{1, 2, 3, 4, 5, 7.5, 42, -9, -177},
	}, &resp, natsrpc.WithRPCTimeout(time.Second*5))
	if err != nil {
		t.Error(err)
		return
	}
	if resp.F32 != 3.4 {
		t.Error("invalid response")
		return
	}

}

func TestSetHandlersAfterStart(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Error(err)
		return
	}

	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))
	if err != nil {
		t.Error(err)
		return
	}

	nrpc.AddRPC("test1", func(c natsrpc.NatsRPCContext) error {
		return c.Ok("Some string")
	}, natsrpc.WithHandlerTimeout(time.Second*10))

	err = nrpc.StartWithContext(t.Context())
	if err != nil {
		t.Error(err)
		return
	}

	nrpc.AddRPC("test", func(c natsrpc.NatsRPCContext) error {
		return c.Ok("Some string")
	}, natsrpc.WithHandlerTimeout(time.Second*10))
}
func TestStartNoHandlers(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Error(err)
		return
	}

	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))
	if err != nil {
		t.Error(err)
		return
	}

	err = nrpc.StartWithContext(t.Context())
	if err != nil {
		t.Error(err)
		return
	}
}
func TestStartSameHandlers(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Error(err)
		return
	}

	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))
	if err != nil {
		t.Error(err)
		return
	}

	nrpc.AddRPC("test", func(c natsrpc.NatsRPCContext) error {
		return c.Ok("Some string")
	}, natsrpc.WithHandlerTimeout(time.Second*10))

	nrpc.AddRPC("test", func(c natsrpc.NatsRPCContext) error {
		return c.Ok("Some string")
	}, natsrpc.WithHandlerTimeout(time.Second*10))

	err = nrpc.StartWithContext(t.Context())
	if err != nil {
		t.Error(err)
		return
	}

}
func TestHandlerTimeout(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Error(err)
		return
	}

	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))
	if err != nil {
		t.Error(err)
		return
	}

	nrpc.AddRPC("test", func(c natsrpc.NatsRPCContext) error {
		<-time.After(time.Second * 3)
		return c.Ok("Some string")
	}, natsrpc.WithHandlerTimeout(time.Second*2))

	err = nrpc.StartWithContext(t.Context())
	if err != nil {
		t.Error(err)
		return
	}

	var resp string

	err = nrpc.CallRPC("testapp.test", nil, &resp, natsrpc.WithRPCTimeout(time.Second*5))
	if err == nil {
		t.Error("no error raised, but should")
		return
	}
}
func TestRPCTimeout(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Error(err)
		return
	}

	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))
	if err != nil {
		t.Error(err)
		return
	}

	nrpc.AddRPC("test", func(c natsrpc.NatsRPCContext) error {
		<-time.After(time.Second * 3)
		return c.Ok("Some string")
	}, natsrpc.WithHandlerTimeout(time.Second*20))

	err = nrpc.StartWithContext(t.Context())
	if err != nil {
		t.Error(err)
		return
	}

	var resp string

	err = nrpc.CallRPC("testapp.test", nil, &resp, natsrpc.WithRPCTimeout(time.Second*2))
	if err == nil {
		t.Error("no error raised, but should")

		return
	}
}

func TestNoResponse(t *testing.T) {
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		t.Error(err)
		return
	}

	nrpc := natsrpc.New(nc, natsrpc.WithBaseName("testapp"))
	if err != nil {
		t.Error(err)
		return
	}

	nrpc.AddRPC("test", func(c natsrpc.NatsRPCContext) error {
		return nil
	}, natsrpc.WithHandlerTimeout(time.Second*20))

	err = nrpc.StartWithContext(t.Context())
	if err != nil {
		t.Error(err)
		return
	}

	var resp string

	err = nrpc.CallRPC("testapp.test", nil, &resp, natsrpc.WithRPCTimeout(time.Second*2))
	if err == nil {
		t.Error(err)
		return
	}
}
