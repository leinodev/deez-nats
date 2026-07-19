package genrust

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
)

func TestWriteExternalTypeImports(t *testing.T) {
	files, err := protodesc.NewFiles(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{
		{
			Name:    proto.String("owner/proto/rpc.proto"),
			Package: proto.String("example.rpc"),
			Syntax:  proto.String("proto3"),
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: proto.String("PersonAttributes"),
			}},
			EnumType: []*descriptorpb.EnumDescriptorProto{{
				Name: proto.String("PersonState"),
				Value: []*descriptorpb.EnumValueDescriptorProto{{
					Name:   proto.String("PERSON_STATE_UNSPECIFIED"),
					Number: proto.Int32(0),
				}},
			}},
		},
		{
			Name:       proto.String("owner/proto/events.proto"),
			Package:    proto.String("example.events"),
			Syntax:     proto.String("proto3"),
			Dependency: []string{"owner/proto/rpc.proto"},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: proto.String("PersonUpserted"),
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:     proto.String("attributes"),
						Number:   proto.Int32(1),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_MESSAGE.Enum(),
						TypeName: proto.String(".example.rpc.PersonAttributes"),
					},
					{
						Name:     proto.String("state"),
						Number:   proto.Int32(2),
						Label:    descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
						Type:     descriptorpb.FieldDescriptorProto_TYPE_ENUM.Enum(),
						TypeName: proto.String(".example.rpc.PersonState"),
					},
				},
			}},
		},
	}})
	if err != nil {
		t.Fatalf("build descriptors: %v", err)
	}
	events, err := files.FindFileByPath("owner/proto/events.proto")
	if err != nil {
		t.Fatalf("find events descriptor: %v", err)
	}

	var got strings.Builder
	writeExternalTypeImports(&got, events)
	const want = "use crate::rpc::{PersonAttributes, PersonState};\n\n"
	if got.String() != want {
		t.Fatalf("external imports mismatch\nwant: %q\n got: %q", want, got.String())
	}
}
