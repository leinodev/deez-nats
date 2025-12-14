package generator

import (
	"strings"

	plugin "google.golang.org/protobuf/types/pluginpb"

	_ "embed"
)

//go:embed template-client.tpl
var clientTemplate string

//go:embed template-server.tpl
var serverTemplate string

type GenerationRequest struct {
	rq             *plugin.CodeGeneratorRequest
	Params         map[string]string
	SourceRelative bool
}

func ParseGenerationRequest(rq *plugin.CodeGeneratorRequest) *GenerationRequest {
	r := &GenerationRequest{
		rq:     rq,
		Params: map[string]string{},
	}
	for _, param := range strings.Split(rq.GetParameter(), ",") {
		var value string
		if i := strings.Index(param, "="); i >= 0 {
			value := param[i+1:]
			param = param[0:i]
			r.Params[param] = value
			continue
		}
		switch param {
		case "paths":
			if value == "source_relative" {
				r.SourceRelative = true
			} else if value == "import" {
				r.SourceRelative = false
			}
		}
	}

	return r
}
