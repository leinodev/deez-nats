// Package protoassets embeds dnatsgen's bundled .proto files so the generator
// is self-contained: a contract can `import "deeznats/annotations.proto"` without
// that file existing on disk next to it.
package protoassets

import _ "embed"

//go:embed deeznats/annotations.proto
var Annotations []byte

// AnnotationsPath is the import path under which Annotations is served.
const AnnotationsPath = "deeznats/annotations.proto"
