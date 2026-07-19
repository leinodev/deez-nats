// Hand-written (NOT generated): end-to-end proof that the generated client,
// server, and event pub/sub actually round-trip over the real deez-nats
// protobuf-binary wire (ProtobufWrap + Any), using an in-process NATS server.
package examplepb_test

import (
	"context"
	"testing"
	"time"

	"github.com/leinodev/deez-nats/marshaller"
	"github.com/leinodev/deez-nats/natsevents"
	"github.com/leinodev/deez-nats/natsrpc"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	server "github.com/nats-io/nats-server/v2/server"
	natstest "github.com/nats-io/nats-server/v2/test"

	pb "github.com/leinodev/deez-nats/dnatsgen/examples/contract/gen/go/examplepb"
)

func startServer(t *testing.T) (*server.Server, *nats.Conn) {
	t.Helper()
	opts := natstest.DefaultTestOptions
	opts.Port = -1
	opts.JetStream = true
	opts.StoreDir = t.TempDir()
	s := natstest.RunServer(&opts)
	nc, err := nats.Connect(s.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { nc.Close(); s.Shutdown() })
	return s, nc
}

// --- RPC ---

type walletImpl struct {
	requestIDs chan string
}

func (w walletImpl) GetBalance(ctx natsrpc.RPCContext, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	w.requestIDs <- ctx.RequestHeaders().Get("X-Request-ID")
	return &pb.GetBalanceResponse{Balance: "100.50", CurrencyCode: "SMC", UpdatedAt: 42}, nil
}

func (walletImpl) Transfer(_ natsrpc.RPCContext, req *pb.TransferRequest) (*pb.TransferResponse, error) {
	return &pb.TransferResponse{TransactionId: "tx-" + req.GetIdempotencyKey(), Ok: true}, nil
}

func TestRPCRoundTrip(t *testing.T) {
	_, nc := startServer(t)
	ctx := context.Background()

	requestIDs := make(chan string, 1)
	router := natsrpc.New(nc)
	pb.RegisterWalletServer(router, walletImpl{requestIDs: requestIDs})
	if err := router.StartWithContext(ctx); err != nil {
		t.Fatalf("start rpc: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}

	client := pb.NewWalletClient(nc)

	bal, err := client.GetBalance(
		ctx,
		&pb.GetBalanceRequest{ServerId: "s1", OwnerId: "o1"},
		natsrpc.WithCallHeader("X-Request-ID", "rpc-request-id"),
	)
	if err != nil {
		t.Fatalf("GetBalance: %v", err)
	}
	if bal.GetBalance() != "100.50" || bal.GetCurrencyCode() != "SMC" || bal.GetUpdatedAt() != 42 {
		t.Fatalf("unexpected balance: %+v", bal)
	}

	tr, err := client.Transfer(ctx, &pb.TransferRequest{IdempotencyKey: "abc", Amount: "10"})
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if tr.GetTransactionId() != "tx-abc" || !tr.GetOk() {
		t.Fatalf("unexpected transfer: %+v", tr)
	}
	select {
	case requestID := <-requestIDs:
		if requestID != "rpc-request-id" {
			t.Fatalf("request id = %q, want rpc-request-id", requestID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for RPC request header")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := router.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown rpc router: %v", err)
	}
}

// --- Core events ---

type presenceResult struct {
	event     *pb.PresenceChanged
	requestID string
}

type presenceImpl struct{ ch chan presenceResult }

func (p presenceImpl) Online(ctx natsevents.EventContext[*nats.Msg, nats.AckOpt], ev *pb.PresenceChanged) error {
	p.ch <- presenceResult{event: ev, requestID: ctx.Headers().Get("X-Request-ID")}
	return nil
}
func (p presenceImpl) Offline(_ natsevents.EventContext[*nats.Msg, nats.AckOpt], _ *pb.PresenceChanged) error {
	return nil
}

func TestCoreEventRoundTrip(t *testing.T) {
	_, nc := startServer(t)
	ctx := context.Background()

	ch := make(chan presenceResult, 1)
	events := natsevents.New(nc)
	pb.RegisterPresence(events, presenceImpl{ch})
	if err := events.StartWithContext(ctx); err != nil {
		t.Fatalf("start core events: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}

	pub := pb.NewPresencePublisherWithRouter(events)
	want := &pb.PresenceChanged{PlayerId: "p1", ServerId: "s1", PingMs: 20, Roles: []string{"vip", "beta"}, PingDelta: -3}
	if err := pub.Online(ctx, want, natsevents.WithCoreEmitHeader("X-Request-ID", "core-request-id")); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case result := <-ch:
		got := result.event
		if got.GetPlayerId() != "p1" || got.GetPingMs() != 20 || got.GetPingDelta() != -3 || len(got.GetRoles()) != 2 {
			t.Fatalf("unexpected event: %+v", got)
		}
		if result.requestID != "core-request-id" {
			t.Fatalf("request id = %q, want core-request-id", result.requestID)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for core event")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := events.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown core events: %v", err)
	}
}

// --- JetStream events ---

type auditResult struct {
	event     *pb.AuditEntry
	requestID string
}

type auditImpl struct{ ch chan auditResult }

func (a auditImpl) Logged(ctx natsevents.EventContext[jetstream.Msg, any], ev *pb.AuditEntry) error {
	a.ch <- auditResult{event: ev, requestID: ctx.Headers().Get("X-Request-ID")}
	return nil
}

func TestJetStreamEventRoundTrip(t *testing.T) {
	_, nc := startServer(t)
	ctx := context.Background()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	if _, err := js.CreateStream(ctx, jetstream.StreamConfig{Name: "AUDIT", Subjects: []string{"audit.>"}}); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	ch := make(chan auditResult, 1)
	events := natsevents.NewJetStream(js, natsevents.WithJetStreamStream("AUDIT"))
	pb.RegisterAudit(events, auditImpl{ch})
	if err := events.StartWithContext(ctx); err != nil {
		t.Fatalf("start jetstream events: %v", err)
	}

	pub := pb.NewAuditPublisherWithRouter(events)
	want := &pb.AuditEntry{Id: "a1", ActorId: "o1", Action: pb.Action_ACTION_CREATE, Meta: map[string]string{"k": "v"}, At: 7, Seq: 99}
	if err := pub.Logged(ctx, want, natsevents.WithJetStreamEmitHeader("X-Request-ID", "js-request-id")); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case result := <-ch:
		got := result.event
		if got.GetId() != "a1" || got.GetAction() != pb.Action_ACTION_CREATE || got.GetMeta()["k"] != "v" || got.GetSeq() != 99 {
			t.Fatalf("unexpected audit: %+v", got)
		}
		if result.requestID != "js-request-id" {
			t.Fatalf("request id = %q, want js-request-id", result.requestID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for jetstream event")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := events.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("shutdown JetStream events: %v", err)
	}
}

// --- envelope sanity: generated types are valid proto.Message through the proto marshaller ---

func TestProtoMarshallerRoundTrip(t *testing.T) {
	m := marshaller.DefaultProtoMarshaller
	in := &pb.AuditEntry{Id: "x", Action: pb.Action_ACTION_DELETE, Meta: map[string]string{"a": "b"}, Seq: -5}
	data, err := m.Marshall(&marshaller.MarshalObject{Data: in})
	if err != nil {
		t.Fatalf("marshall: %v", err)
	}
	out := &pb.AuditEntry{}
	if err := m.Unmarshall(data, &marshaller.MarshalObject{Data: out}); err != nil {
		t.Fatalf("unmarshall: %v", err)
	}
	if out.GetId() != "x" || out.GetAction() != pb.Action_ACTION_DELETE || out.GetSeq() != -5 || out.GetMeta()["a"] != "b" {
		t.Fatalf("roundtrip mismatch: %+v", out)
	}
}
