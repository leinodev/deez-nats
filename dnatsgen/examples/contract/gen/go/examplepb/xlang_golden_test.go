// Hand-written (NOT generated): the Go half of the cross-language wire lock.
// goldenEnvelopeHex is the byte sequence both Go and Kotlin produce for a fixed
// AuditEntry (the Kotlin CrossLangCheck asserts it re-encodes to the same hex).
// This test proves Go decodes that exact sequence and re-encodes identically.
package examplepb_test

import (
	"encoding/hex"
	"testing"

	"github.com/leinodev/deez-nats/marshaller"

	pb "github.com/leinodev/deez-nats/dnatsgen/examples/contract/gen/go/examplepb"
)

const goldenEnvelopeHex = "0a470a26747970652e676f6f676c65617069732e636f6d2f6578616d706c652e4175646974456e747279121d0a02613112026f31180122060a016b1201762807316300000000000000"

func TestCrossLangGoldenEnvelope(t *testing.T) {
	data, err := hex.DecodeString(goldenEnvelopeHex)
	if err != nil {
		t.Fatal(err)
	}

	out := &pb.AuditEntry{}
	if err := marshaller.DefaultProtoMarshaller.Unmarshall(data, &marshaller.MarshalObject{Data: out}); err != nil {
		t.Fatalf("unmarshall golden: %v", err)
	}
	if out.GetId() != "a1" || out.GetActorId() != "o1" || out.GetAction() != pb.Action_ACTION_CREATE ||
		out.GetMeta()["k"] != "v" || out.GetAt() != 7 || out.GetSeq() != 99 {
		t.Fatalf("golden decoded to unexpected value: %+v", out)
	}

	re, err := marshaller.DefaultProtoMarshaller.Marshall(&marshaller.MarshalObject{Data: out})
	if err != nil {
		t.Fatalf("re-marshall: %v", err)
	}
	if got := hex.EncodeToString(re); got != goldenEnvelopeHex {
		t.Fatalf("Go re-encode not byte-identical to golden:\n got=%s\nwant=%s", got, goldenEnvelopeHex)
	}
}
