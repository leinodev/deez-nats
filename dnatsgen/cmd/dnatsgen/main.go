// Command dnatsgen generates deez-nats client/server code (Go + Kotlin) for RPC
// and events from protobuf service definitions annotated with deeznats.* options.
//
// Example:
//
//	dnatsgen \
//	  -I examples/contract \
//	  -proto contract.proto \
//	  -go-out examples/contract/gen/go/contractpb \
//	  -kt-out examples/contract/kotlin/src/main/kotlin/.../gen \
//	  -kt-package ru.lnik801l.example.contract
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bufbuild/protocompile"
	"google.golang.org/protobuf/reflect/protoreflect"

	protoassets "github.com/leinodev/deez-nats/dnatsgen/proto"

	"github.com/leinodev/deez-nats/dnatsgen/internal/gengo"
	"github.com/leinodev/deez-nats/dnatsgen/internal/genkt"
	"github.com/leinodev/deez-nats/dnatsgen/internal/genrust"
	"github.com/leinodev/deez-nats/dnatsgen/internal/parse"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "dnatsgen:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		importPaths   = flag.String("I", ".", "comma/colon-separated proto import paths (roots)")
		protos        = flag.String("proto", "", "comma-separated contract .proto paths (relative to an import path)")
		goOut         = flag.String("go-out", "", "output dir for generated Go (messages + glue); empty disables Go")
		ktOut         = flag.String("kt-out", "", "output dir for generated Kotlin; empty disables Kotlin")
		ktPackage     = flag.String("kt-package", "", "base Kotlin package for generated code")
		ktRuntimePkg  = flag.String("kt-runtime-package", "", "Kotlin package for the DnatsProto runtime (default <kt-package>.runtime)")
		ktRuntimeOut  = flag.String("kt-runtime-out", "", "output dir for DnatsProto.kt (default <kt-out>/runtime); empty + kt-out set => <kt-out>/runtime")
		emitKtRuntime = flag.Bool("kt-runtime", true, "emit the DnatsProto.kt runtime alongside Kotlin output")

		rustOut         = flag.String("rust-out", "", "output dir for generated Rust (<stem>.rs); empty disables Rust")
		rustRuntimeOut  = flag.String("rust-runtime-out", "", "output dir for dnats.rs (default <rust-out>)")
		emitRustRuntime = flag.Bool("rust-runtime", true, "emit the dnats.rs runtime alongside Rust output")
	)
	flag.Parse()

	if strings.TrimSpace(*protos) == "" {
		return errors.New("missing -proto")
	}
	roots := splitList(*importPaths)
	files := splitList(*protos)

	if *ktOut != "" && *ktPackage == "" {
		return errors.New("-kt-out requires -kt-package")
	}
	runtimePackage := *ktRuntimePkg
	if runtimePackage == "" && *ktPackage != "" {
		runtimePackage = *ktPackage + ".runtime"
	}

	compiled, err := compile(roots, files)
	if err != nil {
		return err
	}

	contracts, warnings, err := parse.BuildContracts(compiled, *ktPackage)
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "dnatsgen: warning: "+w)
	}
	if err != nil {
		return err
	}

	// ---- Go ----
	if *goOut != "" {
		absGoOut, err := filepath.Abs(*goOut)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(absGoOut, 0o755); err != nil {
			return err
		}
		fdset := parse.FileDescriptorSet(compiled)
		if err := gengo.GenerateMessages(fdset, files, absGoOut, absGoOut); err != nil {
			return fmt.Errorf("generate Go messages: %w", err)
		}
		for i, fd := range compiled {
			if len(contracts[i].Services) == 0 {
				continue // messages-only proto (e.g. annotations); no NATS glue
			}
			if err := gengo.GenerateGlue(contracts[i], stem(fd.Path()), absGoOut); err != nil {
				return fmt.Errorf("generate Go glue for %s: %w", fd.Path(), err)
			}
		}
		fmt.Printf("dnatsgen: wrote Go -> %s\n", *goOut)
	}

	// ---- Kotlin ----
	if *ktOut != "" {
		for i, fd := range compiled {
			if len(contracts[i].Services) == 0 {
				continue // messages-only proto; no Kotlin client/server to emit
			}
			if err := genkt.GenerateContract(fd, contracts[i], runtimePackage, *ktOut); err != nil {
				return fmt.Errorf("generate Kotlin for %s: %w", fd.Path(), err)
			}
		}
		if *emitKtRuntime {
			rtOut := *ktRuntimeOut
			if rtOut == "" {
				rtOut = filepath.Join(*ktOut, "runtime")
			}
			if err := genkt.GenerateRuntime(runtimePackage, rtOut); err != nil {
				return fmt.Errorf("generate Kotlin runtime: %w", err)
			}
		}
		fmt.Printf("dnatsgen: wrote Kotlin -> %s\n", *ktOut)
	}

	// ---- Rust ----
	if *rustOut != "" {
		for i, fd := range compiled {
			if len(contracts[i].Services) == 0 {
				continue // messages-only proto; no Rust client/server to emit
			}
			if err := genrust.GenerateContract(fd, contracts[i], stem(fd.Path()), *rustOut); err != nil {
				return fmt.Errorf("generate Rust for %s: %w", fd.Path(), err)
			}
		}
		if *emitRustRuntime {
			rtOut := *rustRuntimeOut
			if rtOut == "" {
				rtOut = *rustOut
			}
			if err := genrust.GenerateRuntime(rtOut); err != nil {
				return fmt.Errorf("generate Rust runtime: %w", err)
			}
		}
		fmt.Printf("dnatsgen: wrote Rust -> %s\n", *rustOut)
	}

	return nil
}

// compile parses the contract files with protocompile, serving the embedded
// deeznats/annotations.proto and the standard well-known types in-process.
func compile(roots, files []string) ([]protoreflect.FileDescriptor, error) {
	resolver := protocompile.WithStandardImports(protocompile.CompositeResolver{
		resolverFunc(func(path string) (protocompile.SearchResult, error) {
			if path == protoassets.AnnotationsPath {
				return protocompile.SearchResult{Source: bytes.NewReader(protoassets.Annotations)}, nil
			}
			return protocompile.SearchResult{}, fs.ErrNotExist
		}),
		&protocompile.SourceResolver{ImportPaths: roots},
	})

	c := protocompile.Compiler{
		Resolver:       resolver,
		SourceInfoMode: protocompile.SourceInfoStandard,
	}
	fds, err := c.Compile(context.Background(), files...)
	if err != nil {
		return nil, fmt.Errorf("compile protos: %w", err)
	}
	out := make([]protoreflect.FileDescriptor, len(fds))
	for i := range fds {
		out[i] = fds[i]
	}
	return out, nil
}

type resolverFunc func(string) (protocompile.SearchResult, error)

func (f resolverFunc) FindFileByPath(p string) (protocompile.SearchResult, error) { return f(p) }

func splitList(s string) []string {
	fields := strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ':' })
	out := fields[:0]
	for _, f := range fields {
		if t := strings.TrimSpace(f); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func stem(protoPath string) string {
	base := filepath.Base(protoPath)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
