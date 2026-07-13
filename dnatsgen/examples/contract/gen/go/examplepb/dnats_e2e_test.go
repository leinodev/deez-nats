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

type walletImpl struct{}

func (walletImpl) GetBalance(_ natsrpc.RPCContext, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	return &pb.GetBalanceResponse{Balance: "100.50", CurrencyCode: "SMC", UpdatedAt: 42}, nil
}

func (walletImpl) Transfer(_ natsrpc.RPCContext, req *pb.TransferRequest) (*pb.TransferResponse, error) {
	return &pb.TransferResponse{TransactionId: "tx-" + req.GetIdempotencyKey(), Ok: true}, nil
}

func TestRPCRoundTrip(t *testing.T) {
	_, nc := startServer(t)
	// NOTE: we intentionally do NOT cancel the context to trigger router
	// Shutdown here — deez-nats v1.2.0 has a slice-aliasing bug in
	// subscriptions.Tracker.Unsubscribe that panics with >=2 subscriptions.
	// That is orthogonal to the generated code under test; the test process
	// exits cleanly via t.Cleanup (nc.Close + server.Shutdown).
	ctx := context.Background()

	router := natsrpc.New(nc)
	pb.RegisterWalletServer(router, walletImpl{})
	if err := router.StartWithContext(ctx); err != nil {
		t.Fatalf("start rpc: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}

	client := pb.NewWalletClient(nc)

	bal, err := client.GetBalance(ctx, &pb.GetBalanceRequest{ServerId: "s1", OwnerId: "o1"})
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
}

// --- Core events ---

type presenceImpl struct{ ch chan *pb.PresenceChanged }

func (p presenceImpl) Online(_ natsevents.EventContext[*nats.Msg, nats.AckOpt], ev *pb.PresenceChanged) error {
	p.ch <- ev
	return nil
}
func (p presenceImpl) Offline(_ natsevents.EventContext[*nats.Msg, nats.AckOpt], _ *pb.PresenceChanged) error {
	return nil
}

func TestCoreEventRoundTrip(t *testing.T) {
	_, nc := startServer(t)
	// NOTE: we intentionally do NOT cancel the context to trigger router
	// Shutdown here — deez-nats v1.2.0 has a slice-aliasing bug in
	// subscriptions.Tracker.Unsubscribe that panics with >=2 subscriptions.
	// That is orthogonal to the generated code under test; the test process
	// exits cleanly via t.Cleanup (nc.Close + server.Shutdown).
	ctx := context.Background()

	ch := make(chan *pb.PresenceChanged, 1)
	events := natsevents.New(nc)
	pb.RegisterPresence(events, presenceImpl{ch})
	if err := events.StartWithContext(ctx); err != nil {
		t.Fatalf("start core events: %v", err)
	}
	if err := nc.Flush(); err != nil {
		t.Fatal(err)
	}

	pub := pb.NewPresencePublisher(nc)
	want := &pb.PresenceChanged{PlayerId: "p1", ServerId: "s1", PingMs: 20, Roles: []string{"vip", "beta"}, PingDelta: -3}
	if err := pub.Online(ctx, want); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case got := <-ch:
		if got.GetPlayerId() != "p1" || got.GetPingMs() != 20 || got.GetPingDelta() != -3 || len(got.GetRoles()) != 2 {
			t.Fatalf("unexpected event: %+v", got)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for core event")
	}
}

// --- JetStream events ---

type auditImpl struct{ ch chan *pb.AuditEntry }

func (a auditImpl) Logged(_ natsevents.EventContext[jetstream.Msg, any], ev *pb.AuditEntry) error {
	a.ch <- ev
	return nil
}

func TestJetStreamEventRoundTrip(t *testing.T) {
	_, nc := startServer(t)
	// NOTE: we intentionally do NOT cancel the context to trigger router
	// Shutdown here — deez-nats v1.2.0 has a slice-aliasing bug in
	// subscriptions.Tracker.Unsubscribe that panics with >=2 subscriptions.
	// That is orthogonal to the generated code under test; the test process
	// exits cleanly via t.Cleanup (nc.Close + server.Shutdown).
	ctx := context.Background()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("jetstream: %v", err)
	}
	if _, err := js.CreateStream(ctx, jetstream.StreamConfig{Name: "AUDIT", Subjects: []string{"audit.>"}}); err != nil {
		t.Fatalf("create stream: %v", err)
	}

	ch := make(chan *pb.AuditEntry, 1)
	events := natsevents.NewJetStream(js, natsevents.WithJetStreamStream("AUDIT"))
	pb.RegisterAudit(events, auditImpl{ch})
	if err := events.StartWithContext(ctx); err != nil {
		t.Fatalf("start jetstream events: %v", err)
	}

	pub := pb.NewAuditPublisher(js)
	want := &pb.AuditEntry{Id: "a1", ActorId: "o1", Action: pb.Action_ACTION_CREATE, Meta: map[string]string{"k": "v"}, At: 7, Seq: 99}
	if err := pub.Logged(ctx, want); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case got := <-ch:
		if got.GetId() != "a1" || got.GetAction() != pb.Action_ACTION_CREATE || got.GetMeta()["k"] != "v" || got.GetSeq() != 99 {
			t.Fatalf("unexpected audit: %+v", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for jetstream event")
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
