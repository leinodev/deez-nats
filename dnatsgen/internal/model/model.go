// Package model is the language-neutral view of a deez-nats contract that the
// Go and Kotlin emitters render from. It captures the service/method graph and
// the NATS intent (subject, RPC vs event, transport) extracted from the
// deeznats.* custom options — it does NOT capture message field layout, which
// the emitters read directly from the protobuf descriptors.
package model

// Kind distinguishes request/reply RPC methods from fire-and-forget events.
type Kind int

const (
	KindRPC Kind = iota
	KindEvent
)

// Transport selects the NATS event delivery mechanism.
type Transport int

const (
	Core Transport = iota
	JetStream
)

// Contract is one generated .proto file's worth of services.
type Contract struct {
	ProtoPackage string // e.g. "spacemc.economy"
	GoImportPath string // import path of the generated messages package (from go_package)
	GoPkgName    string // short Go package name
	KtPackage    string // base Kotlin package (from -kt-package flag)
	Services     []*Service
	// HasRPC/HasCoreEvents/HasJetStreamEvents drive conditional imports in templates.
	HasRPC             bool
	HasCoreEvents      bool
	HasJetStreamEvents bool
}

// Service is a proto service, classified as RPC or event.
type Service struct {
	Name           string // proto/Go service name, e.g. "Economy"
	Kind           Kind
	EventTransport Transport // for event services: the transport of its single router
	EventStream    string    // for JetStream event services: the stream name
	Methods        []*Method
	// Convenience splits for templates.
	RPCMethods   []*Method
	EventMethods []*Method
}

// IsJetStream reports whether an event service uses the JetStream transport.
func (s *Service) IsJetStream() bool { return s.EventTransport == JetStream }

// Method is a single rpc entry, resolved to a full NATS subject + intent.
type Method struct {
	Name      string // Go method name, e.g. "GetBalance"
	KtFunc    string // Kotlin function name (lower camel), e.g. "getBalance"
	Service   string // owning service name (for unique subject const names)
	Subject   string // full subject, e.g. "economy.wallet.get"
	GoConst   string // Go subject const identifier, e.g. "EconomyGetBalanceSubject"
	KtConst   string // Kotlin subject const identifier, e.g. "ECONOMY_GET_BALANCE"
	Kind      Kind
	Transport Transport // events only
	Stream    string    // events only (JetStream)

	// Message type names (simple, as emitted by protoc-gen-go / genkt).
	InputName     string // e.g. "GetBalanceRequest"
	OutputName    string // e.g. "GetBalanceResponse"; empty for events / Empty
	InputTypeURL  string // "type.googleapis.com/<protoPkg>.<InputName>"
	OutputTypeURL string // for RPC responses; empty for events
}
