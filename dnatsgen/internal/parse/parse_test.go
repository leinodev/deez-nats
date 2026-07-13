package parse

import (
	"bytes"
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/reflect/protoreflect"

	protoassets "github.com/leinodev/deez-nats/dnatsgen/proto"
)

// compileInline compiles one in-memory .proto, serving deeznats/annotations.proto
// and the well-known types like the real CLI does.
func compileInline(t *testing.T, name, src string) []protoreflect.FileDescriptor {
	t.Helper()
	resolver := protocompile.WithStandardImports(protocompile.CompositeResolver{
		resolverFn(func(p string) (protocompile.SearchResult, error) {
			switch p {
			case name:
				return protocompile.SearchResult{Source: strings.NewReader(src)}, nil
			case protoassets.AnnotationsPath:
				return protocompile.SearchResult{Source: bytes.NewReader(protoassets.Annotations)}, nil
			}
			return protocompile.SearchResult{}, fs.ErrNotExist
		}),
	})
	c := protocompile.Compiler{Resolver: resolver}
	fds, err := c.Compile(context.Background(), name)
	if err != nil {
		t.Fatalf("compile %s: %v", name, err)
	}
	out := make([]protoreflect.FileDescriptor, len(fds))
	for i := range fds {
		out[i] = fds[i]
	}
	return out
}

type resolverFn func(string) (protocompile.SearchResult, error)

func (f resolverFn) FindFileByPath(p string) (protocompile.SearchResult, error) { return f(p) }

func TestOneofRejected(t *testing.T) {
	src := `syntax = "proto3";
package t;
message M {
  oneof kind {
    string a = 1;
    int32 b = 2;
  }
}`
	_, _, err := BuildContracts(compileInline(t, "oneof.proto", src), "")
	if err == nil {
		t.Fatal("expected error for oneof, got nil")
	}
	if !strings.Contains(err.Error(), "oneof") {
		t.Fatalf("expected oneof error, got: %v", err)
	}
}

func TestProto3OptionalAllowed(t *testing.T) {
	// proto3 `optional` is a synthetic oneof and must NOT be rejected.
	src := `syntax = "proto3";
package t;
message M {
  optional int32 a = 1;
  string b = 2;
}`
	if _, _, err := BuildContracts(compileInline(t, "opt.proto", src), ""); err != nil {
		t.Fatalf("proto3 optional should be allowed, got: %v", err)
	}
}

func TestStreamingWarnsAndSkips(t *testing.T) {
	src := `syntax = "proto3";
package t;
import "deeznats/annotations.proto";
message Q { string x = 1; }
message R { string y = 1; }
service S {
  option (deeznats.subject_prefix) = "s";
  rpc Unary(Q) returns (R) { option (deeznats.subject) = "u"; }
  rpc Tail(Q) returns (stream R) { option (deeznats.subject) = "t"; }
}`
	contracts, warnings, err := BuildContracts(compileInline(t, "stream.proto", src), "")
	if err != nil {
		t.Fatalf("streaming should warn, not error: %v", err)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "S.Tail") {
		t.Fatalf("expected one warning mentioning S.Tail, got: %v", warnings)
	}
	svc := contracts[0].Services[0]
	if len(svc.Methods) != 1 || svc.Methods[0].Name != "Unary" {
		t.Fatalf("expected only the unary method to survive, got: %+v", svc.Methods)
	}
}
