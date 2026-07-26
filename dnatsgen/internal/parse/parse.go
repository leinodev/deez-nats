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
		service, serviceWarnings, err := buildService(svcs.Get(i), c)
		if err != nil {
			return nil, nil, err
		}
		warnings = append(warnings, serviceWarnings...)
		c.Services = append(c.Services, service)
	}
	return c, warnings, nil
}

type serviceOptions struct {
	prefix, stream string
	events, jet    bool
}

func buildService(descriptor protoreflect.ServiceDescriptor, contract *model.Contract) (*model.Service, []string, error) {
	options := readServiceOptions(descriptor)
	service := &model.Service{Name: string(descriptor.Name())}
	if options.events {
		service.Kind = model.KindEvent
		service.EventTransport = model.Core
		if options.jet {
			service.EventTransport = model.JetStream
		}
		service.EventStream = options.stream
	}
	var warnings []string
	methods := descriptor.Methods()
	for i := 0; i < methods.Len(); i++ {
		method, warning, err := buildMethod(descriptor, methods.Get(i), service, options.prefix, contract)
		if err != nil {
			return nil, nil, err
		}
		if warning != "" {
			warnings = append(warnings, warning)
			continue
		}
		addMethod(service, method)
	}
	if err := validateService(service); err != nil {
		return nil, nil, err
	}
	return service, warnings, nil
}

func readServiceOptions(descriptor protoreflect.ServiceDescriptor) serviceOptions {
	var options serviceOptions
	rangeOpts(descriptor.Options(), func(name string, value protoreflect.Value) {
		switch name {
		case optSubjectPrefix:
			options.prefix = value.String()
		case optServiceEvents:
			options.events = value.Bool()
		case optServiceJetStr:
			options.jet = value.Bool()
		case optServiceStream:
			options.stream = value.String()
		}
	})
	return options
}

func buildMethod(
	serviceDescriptor protoreflect.ServiceDescriptor,
	descriptor protoreflect.MethodDescriptor,
	service *model.Service,
	prefix string,
	contract *model.Contract,
) (*model.Method, string, error) {
	if descriptor.IsStreamingClient() || descriptor.IsStreamingServer() {
		return nil, fmt.Sprintf(
			"streaming method %s.%s is not supported and was skipped", serviceDescriptor.Name(), descriptor.Name(),
		), nil
	}
	leaf, event := readMethodOptions(descriptor)
	if leaf == "" {
		leaf = deriveLeaf(string(descriptor.Name()))
	}
	subject := leaf
	if prefix != "" {
		subject = prefix + "." + leaf
	}
	method := &model.Method{
		Name: string(descriptor.Name()), KtFunc: lowerFirst(string(descriptor.Name())),
		Service: string(serviceDescriptor.Name()), Subject: subject,
		GoConst:   goConst(string(serviceDescriptor.Name()), string(descriptor.Name())),
		KtConst:   ktConst(string(serviceDescriptor.Name()), string(descriptor.Name())),
		InputName: string(descriptor.Input().Name()), InputTypeURL: typeURLPrefix + string(descriptor.Input().FullName()),
	}
	if service.Kind == model.KindEvent || event {
		if descriptor.Output().FullName() != "google.protobuf.Empty" {
			return nil, "", fmt.Errorf(
				"event method %s.%s must return google.protobuf.Empty", serviceDescriptor.Name(), descriptor.Name(),
			)
		}
		method.Kind, method.Transport, method.Stream = model.KindEvent, service.EventTransport, service.EventStream
		contract.HasJetStreamEvents = contract.HasJetStreamEvents || method.Transport == model.JetStream
		contract.HasCoreEvents = contract.HasCoreEvents || method.Transport != model.JetStream
		return method, "", nil
	}
	method.Kind = model.KindRPC
	method.OutputName = string(descriptor.Output().Name())
	method.OutputTypeURL = typeURLPrefix + string(descriptor.Output().FullName())
	contract.HasRPC = true
	return method, "", nil
}

func readMethodOptions(descriptor protoreflect.MethodDescriptor) (leaf string, event bool) {
	rangeOpts(descriptor.Options(), func(name string, value protoreflect.Value) {
		switch name {
		case optMethodSubject:
			leaf = value.String()
		case optMethodEvent:
			event = value.Bool()
		}
	})
	return leaf, event
}

func addMethod(service *model.Service, method *model.Method) {
	service.Methods = append(service.Methods, method)
	if method.Kind == model.KindEvent {
		service.EventMethods = append(service.EventMethods, method)
		return
	}
	service.RPCMethods = append(service.RPCMethods, method)
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
		if cur == '_' {
			words = append(words, string(runes[start:i]))
			start = i + 1
			continue
		}
		if wordBoundary(prev, cur, next) {
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

func wordBoundary(previous, current, next rune) bool {
	return isLower(previous) && isUpper(current) ||
		isUpper(previous) && isUpper(current) && isLower(next) ||
		isLetter(previous) && isDigit(current) ||
		isDigit(previous) && isLetter(current)
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
