package {{.PackageName}}
import (
	"context"
	natsrpc "github.com/leinodev/deez-nats"
)

{{}}

{{- range .Services}}
{{- $service := . -}}

type {{.ServiceNamePascal}}Server interface {
	{{- range .UnaryHandlers}}
	{{.MethodHandlerPascal}}(ctx natsrpc.UnaryContext, r *{{.RequestType}}) (*{{.ResponseType}}, error)
	{{- end}}
}

func Register{{.ServiceNamePascal}}Server(nrpc natsrpc.NatsRPC, impl {{.ServiceNamePascal}}Server, opts ...natsrpc.HandlerOption) {
	proxy := {{.ServiceNameCamel}}ServerProxyImpl{
		impl: impl,
	}

	group := nrpc.Group("{{.ServiceNameCamel}}")
	
	{{- range .UnaryHandlers}}
	group.AddUnaryHandler("{{.MethodRoute}}", proxy.{{.MethodHandlerPascal}}, opts...)
	{{- end}}
}

type {{.ServiceNameCamel}}ServerProxyImpl struct {
	impl {{.ServiceNamePascal}}Server
}

// Unary
{{- range .UnaryHandlers}}
func (s *{{$service.ServiceNameCamel}}ServerProxyImpl) {{.MethodHandlerPascal}}(ctx natsrpc.UnaryContext) error {
	var rq {{.RequestType}}
	if err := ctx.Request(&rq); err != nil {
		return err
	}
	resp, err := s.impl.{{.MethodHandlerPascal}}(ctx, &rq)
	if err != nil {
		return err
	}
	return ctx.Ok(resp)
}
{{- end}}
{{- end}}
