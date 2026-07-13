// Package parse turns compiled protobuf descriptors into the dnatsgen model.
// It reads the deeznats.* custom options dynamically (by extension full-name)
// straight off the descriptors, so it needs no generated annotations package.
package parse

import (
	"fmt"
	"strings"

	"github.com/bufbuild/protocompile/protoutil"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	"github.com/leinodev/deez-nats/dnatsgen/internal/model"
)

// Option full-names, as resolved by protocompile from deeznats/annotations.proto.
const (
	optSubjectPrefix = "deeznats.subject_prefix"
	optServiceEvents = "deeznats.events"
	optServiceJetStr = "deeznats.jetstream"
	optServiceStream = "deeznats.stream"
	optMethodSubject = "deeznats.subject"
	optMethodEvent   = "deeznats.event"
	optMethodJetStr  = "deeznats.event_jetstream"
	optMethodStream  = "deeznats.event_stream"
	typeURLPrefix    = "type.googleapis.com/"
)

// BuildContracts builds one model.Contract per target file descriptor. It also
// returns non-fatal warnings (e.g. skipped streaming methods) for the caller to
// surface; unsupported constructs that cannot be represented (real oneof) are
// returned as a hard error instead.
func BuildContracts(files []protoreflect.FileDescriptor, ktPackage string) ([]*model.Contract, []string, error) {
	var out []*model.Contract
	var warnings []string
	for _, fd := range files {
		c, w, err := buildContract(fd, ktPackage)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", fd.Path(), err)
		}
		out = append(out, c)
		warnings = append(warnings, w...)
	}
	return out, warnings, nil
}

func buildContract(fd protoreflect.FileDescriptor, ktPackage string) (*model.Contract, []string, error) {
	// oneof is not representable in the generated Kotlin (no sealed/@ProtoOneOf),
	// so reject it outright. proto3 `optional` is a *synthetic* oneof and is allowed.
	if name, ok := findRealOneof(fd.Messages()); ok {
		return nil, nil, fmt.Errorf("oneof is not supported by dnatsgen (found %s); split it into separate fields", name)
	}

	var warnings []string
	goImport, goPkg := goPackage(fd)
	c := &model.Contract{
		ProtoPackage: string(fd.Package()),
		GoImportPath: goImport,
		GoPkgName:    goPkg,
		KtPackage:    ktPackage,
	}

	svcs := fd.Services()
	for i := 0; i < svcs.Len(); i++ {
		sd := svcs.Get(i)

		var prefix, svcStream string
		var svcEvents, svcJet bool
		rangeOpts(sd.Options(), func(name string, v protoreflect.Value) {
			switch name {
			case optSubjectPrefix:
				prefix = v.String()
			case optServiceEvents:
				svcEvents = v.Bool()
			case optServiceJetStr:
				svcJet = v.Bool()
			case optServiceStream:
				svcStream = v.String()
			}
		})

		svc := &model.Service{Name: string(sd.Name())}
		if svcEvents {
			svc.Kind = model.KindEvent
			if svcJet {
				svc.EventTransport = model.JetStream
			} else {
				svc.EventTransport = model.Core
			}
			svc.EventStream = svcStream
		}

		ms := sd.Methods()
		for j := 0; j < ms.Len(); j++ {
			md := ms.Get(j)

			// Streaming is not supported by deez-nats (unary request/reply only);
			// warn and skip the method rather than silently mis-generating it.
			if md.IsStreamingClient() || md.IsStreamingServer() {
				warnings = append(warnings, fmt.Sprintf("streaming method %s.%s is not supported and was skipped", sd.Name(), md.Name()))
				continue
			}

			var leaf, mStream string
			var mEvent, mJet bool
			rangeOpts(md.Options(), func(name string, v protoreflect.Value) {
				switch name {
				case optMethodSubject:
					leaf = v.String()
				case optMethodEvent:
					mEvent = v.Bool()
				case optMethodJetStr:
					mJet = v.Bool()
				case optMethodStream:
					mStream = v.String()
				}
			})
			_ = mJet
			_ = mStream

			if leaf == "" {
				leaf = deriveLeaf(string(md.Name()))
			}
			subject := leaf
			if prefix != "" {
				subject = prefix + "." + leaf
			}

			m := &model.Method{
				Name:         string(md.Name()),
				KtFunc:       lowerFirst(string(md.Name())),
				Service:      string(sd.Name()),
				Subject:      subject,
				GoConst:      goConst(string(sd.Name()), string(md.Name())),
				KtConst:      ktConst(string(sd.Name()), string(md.Name())),
				InputName:    string(md.Input().Name()),
				InputTypeURL: typeURLPrefix + string(md.Input().FullName()),
			}

			if svcEvents || mEvent {
				m.Kind = model.KindEvent
				m.Transport = svc.EventTransport
				m.Stream = svc.EventStream
				if m.Transport == model.JetStream {
					c.HasJetStreamEvents = true
				} else {
					c.HasCoreEvents = true
				}
				if md.Output().FullName() != "google.protobuf.Empty" {
					return nil, nil, fmt.Errorf("event method %s.%s must return google.protobuf.Empty", sd.Name(), md.Name())
				}
			} else {
				m.Kind = model.KindRPC
				m.OutputName = string(md.Output().Name())
				m.OutputTypeURL = typeURLPrefix + string(md.Output().FullName())
				c.HasRPC = true
			}
			svc.Methods = append(svc.Methods, m)
			if m.Kind == model.KindEvent {
				svc.EventMethods = append(svc.EventMethods, m)
			} else {
				svc.RPCMethods = append(svc.RPCMethods, m)
			}
		}

		if err := validateService(svc); err != nil {
			return nil, nil, err
		}
		c.Services = append(c.Services, svc)
	}
	return c, warnings, nil
}

// findRealOneof reports the first non-synthetic oneof in msgs (recursively).
// proto3 `optional` produces a *synthetic* oneof, which is not flagged.
func findRealOneof(msgs protoreflect.MessageDescriptors) (string, bool) {
	for i := 0; i < msgs.Len(); i++ {
		md := msgs.Get(i)
		if md.IsMapEntry() {
			continue
		}
		oneofs := md.Oneofs()
		for j := 0; j < oneofs.Len(); j++ {
			if !oneofs.Get(j).IsSynthetic() {
				return string(md.FullName()) + "." + string(oneofs.Get(j).Name()), true
			}
		}
		if name, ok := findRealOneof(md.Messages()); ok {
			return name, true
		}
	}
	return "", false
}

// validateService enforces v1 constraints: an events service is uniform (one
// transport / one stream for the single router it produces).
func validateService(svc *model.Service) error {
	if svc.Kind != model.KindEvent {
		return nil
	}
	if svc.EventTransport == model.JetStream && strings.TrimSpace(svc.EventStream) == "" {
		return fmt.Errorf("jetstream events service %q requires (deeznats.stream)", svc.Name)
	}
	return nil
}

// rangeOpts iterates the set (extension) fields of an options message by full-name.
func rangeOpts(opts protoreflect.ProtoMessage, fn func(name string, v protoreflect.Value)) {
	if opts == nil {
		return
	}
	m := opts.ProtoReflect()
	if !m.IsValid() {
		return
	}
	m.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		fn(string(fd.FullName()), v)
		return true
	})
}

// FileDescriptorSet collects the target files plus their transitive imports,
// topologically ordered (deps first), for feeding to protoc-gen-go.
func FileDescriptorSet(targets []protoreflect.FileDescriptor) *descriptorpb.FileDescriptorSet {
	seen := map[string]bool{}
	var out []*descriptorpb.FileDescriptorProto
	var visit func(fd protoreflect.FileDescriptor)
	visit = func(fd protoreflect.FileDescriptor) {
		if seen[fd.Path()] {
			return
		}
		seen[fd.Path()] = true
		imps := fd.Imports()
		for i := 0; i < imps.Len(); i++ {
			visit(imps.Get(i).FileDescriptor)
		}
		out = append(out, protoutil.ProtoFromFileDescriptor(fd))
	}
	for _, fd := range targets {
		visit(fd)
	}
	return &descriptorpb.FileDescriptorSet{File: out}
}

func goPackage(fd protoreflect.FileDescriptor) (importPath, pkgName string) {
	opts, _ := fd.Options().(*descriptorpb.FileOptions)
	gp := ""
	if opts != nil {
		gp = opts.GetGoPackage()
	}
	if gp == "" {
		// Fall back to proto package; last segment as name.
		importPath = string(fd.Package())
		pkgName = lastSegment(importPath, ".")
		return importPath, pkgName
	}
	if i := strings.LastIndex(gp, ";"); i >= 0 {
		return gp[:i], gp[i+1:]
	}
	return gp, lastSegment(gp, "/")
}

func lastSegment(s, sep string) string {
	if i := strings.LastIndex(s, sep); i >= 0 {
		return s[i+1:]
	}
	return s
}

// deriveLeaf turns a method name into a dotted subject leaf: GetBalance -> get.balance.
func deriveLeaf(method string) string {
	words := splitWords(method)
	for i := range words {
		words[i] = strings.ToLower(words[i])
	}
	return strings.Join(words, ".")
}

// goConst: Economy + GetBalance -> EconomyGetBalanceSubject (collision-free with
// message type names, which end in Request/Response/Event).
func goConst(svc, method string) string {
	return svc + method + "Subject"
}

// ktConst: Economy + GetBalance -> ECONOMY_GET_BALANCE.
func ktConst(svc, method string) string {
	parts := append(splitWords(svc), splitWords(method)...)
	for i := range parts {
		parts[i] = strings.ToUpper(parts[i])
	}
	return strings.Join(parts, "_")
}

// splitWords splits a CamelCase / PascalCase identifier into its words.
// "GetBalance" -> ["Get","Balance"]; "HTTPServer" -> ["HTTP","Server"].
func splitWords(s string) []string {
	var words []string
	runes := []rune(s)
	start := 0
	for i := 1; i < len(runes); i++ {
		prev, cur := runes[i-1], runes[i]
		var next rune
		if i+1 < len(runes) {
			next = runes[i+1]
		}
		boundary := false
		switch {
		case isLower(prev) && isUpper(cur):
			boundary = true
		case isUpper(prev) && isUpper(cur) && isLower(next):
			boundary = true
		case (isLetter(prev) && isDigit(cur)) || (isDigit(prev) && isLetter(cur)):
			boundary = true
		case cur == '_':
			words = append(words, string(runes[start:i]))
			start = i + 1
			continue
		}
		if boundary {
			words = append(words, string(runes[start:i]))
			start = i
		}
	}
	if start < len(runes) {
		words = append(words, string(runes[start:]))
	}
	// drop empties from underscores
	out := words[:0]
	for _, w := range words {
		if w != "" {
			out = append(out, w)
		}
	}
	return out
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if isUpper(r[0]) {
		r[0] = r[0] - 'A' + 'a'
	}
	return string(r)
}

func isUpper(r rune) bool  { return r >= 'A' && r <= 'Z' }
func isLower(r rune) bool  { return r >= 'a' && r <= 'z' }
func isDigit(r rune) bool  { return r >= '0' && r <= '9' }
func isLetter(r rune) bool { return isUpper(r) || isLower(r) }
