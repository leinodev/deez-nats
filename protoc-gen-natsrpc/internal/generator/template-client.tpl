package {{.PackageName}}

import (
	"context"
	natsrpc "github.com/leinodev/deez-nats"
)

{{}}

{{- range .Services}}
{{- $service := . -}}

type {{.ServiceNamePascal}}Client interface {
	{{- range .UnaryHandlers}}
	{{.MethodHandlerPascal}}(ctx natsrpc.UnaryContext, r *{{.RequestType}}) (*{{.ResponseType}}, error)
	{{- end}}
}

func New{{.ServiceNamePascal}}Client(nrpc natsrpc.NatsRPC) *{{.ServiceNamePascal}}Client {
	return &{{.ServiceNamePascal}}ClientImpl{
		nrpc: nrpc,
	}
}

type {{.ServiceNamePascal}}ClientImpl struct {
	nrpc natsrpc.NatsRPC
}

// Unary
{{- range .UnaryHandlers}}
func (c *{{$service.ServiceNamePascal}}ClientImpl) {{.MethodHandlerPascal}}(ctx context.Context, r *{{.RequestType}}, opts ...natsrpc.CallOption) (*{{.ResponseType}}, error) {
	var resp HelloResponse
	if err := c.nrpc.UnaryCall(ctx, "{{$service.ServiceNameCamel}}.{{.MethodRoute}}", r, &resp, opts...); err != nil {
		return nil, err
	}
	return &resp, nil
}
{{- end}}

{{- end}}
