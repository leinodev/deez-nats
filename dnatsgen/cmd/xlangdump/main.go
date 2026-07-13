// Command xlangdump prints, as hex, the deez-nats protobuf-binary wire envelope
// for a fixed AuditEntry. It is used to seed the Go<->Kotlin cross-language
// interop tests (the Kotlin test decodes this exact byte sequence).
package main

import (
	"encoding/hex"
	"fmt"

	"github.com/leinodev/deez-nats/marshaller"

	pb "github.com/leinodev/deez-nats/dnatsgen/examples/contract/gen/go/examplepb"
)

func main() {
	entry := &pb.AuditEntry{
		Id:      "a1",
		ActorId: "o1",
		Action:  pb.Action_ACTION_CREATE,
		Meta:    map[string]string{"k": "v"},
		At:      7,
		Seq:     99,
	}
	data, err := marshaller.DefaultProtoMarshaller.Marshall(&marshaller.MarshalObject{Data: entry})
	if err != nil {
		panic(err)
	}
	fmt.Println(hex.EncodeToString(data))
}
