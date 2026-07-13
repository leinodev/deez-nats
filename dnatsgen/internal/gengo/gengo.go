// Package gengo emits the Go side of a deez-nats contract: protobuf message
// types (by driving the official protoc-gen-go plugin) plus the NATS RPC/event
// glue (client, server, publisher, subscriber) rendered from templates.
package gengo

import (
	"bytes"
	"embed"
	"fmt"
	"go/format"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/leinodev/deez-nats/dnatsgen/internal/model"
)

//go:embed templates/glue.go.tmpl
var templatesFS embed.FS

var glueTmpl = template.Must(template.New("glue.go.tmpl").ParseFS(templatesFS, "templates/glue.go.tmpl"))

// GenerateMessages drives protoc-gen-go to emit *.pb.go for fileToGenerate into
// outDir. moduleHintDir is any directory inside a Go module that requires
// google.golang.org/protobuf (used to resolve the plugin via `go run`).
func GenerateMessages(fdset *descriptorpb.FileDescriptorSet, fileToGenerate []string, outDir, moduleHintDir string) error {
	files, err := runProtocGenGo(fdset, fileToGenerate, "paths=source_relative", moduleHintDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, f := range files {
		dst := filepath.Join(outDir, filepath.Base(f.GetName()))
		if err := os.WriteFile(dst, []byte(f.GetContent()), 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
	}
	return nil
}

// runProtocGenGo builds a CodeGeneratorRequest and pipes it to the protoc-gen-go
// plugin on stdin, returning its generated files. No protoc binary is involved.
func runProtocGenGo(fdset *descriptorpb.FileDescriptorSet, fileToGenerate []string, parameter, moduleHintDir string) ([]*pluginpb.CodeGeneratorResponse_File, error) {
	req := &pluginpb.CodeGeneratorRequest{
		FileToGenerate:  fileToGenerate,
		Parameter:       proto.String(parameter),
		ProtoFile:       fdset.File,
		CompilerVersion: &pluginpb.Version{Major: proto.Int32(6), Minor: proto.Int32(30), Patch: proto.Int32(0)},
	}
	in, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal CodeGeneratorRequest: %w", err)
	}

	name, args := pluginCommand()
	cmd := exec.Command(name, args...)
	cmd.Dir = moduleHintDir
	cmd.Stdin = bytes.NewReader(in)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run %s: %w\n%s", name, err, stderr.String())
	}

	var resp pluginpb.CodeGeneratorResponse
	if err := proto.Unmarshal(stdout.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("unmarshal CodeGeneratorResponse: %w", err)
	}
	if resp.GetError() != "" {
		return nil, fmt.Errorf("protoc-gen-go: %s", resp.GetError())
	}
	return resp.File, nil
}

// pluginCommand resolves the protoc-gen-go invocation. Override with
// DNATSGEN_PROTOC_GEN_GO (space-separated); defaults to running it via `go run`
// from the target module so no global install is required.
func pluginCommand() (string, []string) {
	if v := strings.TrimSpace(os.Getenv("DNATSGEN_PROTOC_GEN_GO")); v != "" {
		parts := strings.Fields(v)
		return parts[0], parts[1:]
	}
	return "go", []string{"run", "google.golang.org/protobuf/cmd/protoc-gen-go"}
}

// GenerateGlue renders the NATS RPC/event glue for one contract into
// outDir/<stem>_dnats.gen.go.
func GenerateGlue(c *model.Contract, stem, outDir string) error {
	data := struct {
		Contract *model.Contract
		Imports  []string
	}{Contract: c, Imports: imports(c)}

	var buf bytes.Buffer
	if err := glueTmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("render glue: %w", err)
	}
	src, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("gofmt glue (%s): %w\n%s", c.ProtoPackage, err, buf.String())
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	dst := filepath.Join(outDir, stem+"_dnats.gen.go")
	return os.WriteFile(dst, src, 0o644)
}

// imports computes the exact import set the glue file needs, so the rendered
// code never carries an unused import.
func imports(c *model.Contract) []string {
	needNats := c.HasRPC || c.HasCoreEvents
	needJetStream := c.HasJetStreamEvents
	needNatsRPC := c.HasRPC
	needEvents := c.HasCoreEvents || c.HasJetStreamEvents
	// context + marshaller are used by every client / publisher.
	needContext := c.HasRPC || needEvents
	needMarshaller := needContext

	var out []string
	if needContext {
		out = append(out, `"context"`)
	}
	if needNats {
		out = append(out, `"github.com/nats-io/nats.go"`)
	}
	if needJetStream {
		out = append(out, `"github.com/nats-io/nats.go/jetstream"`)
	}
	if needMarshaller {
		out = append(out, `"github.com/leinodev/deez-nats/marshaller"`)
	}
	if needEvents {
		out = append(out, `"github.com/leinodev/deez-nats/natsevents"`)
	}
	if needNatsRPC {
		out = append(out, `"github.com/leinodev/deez-nats/natsrpc"`)
	}
	return out
}
