package generator

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"text/template"

	"google.golang.org/protobuf/types/descriptorpb"
	plugin "google.golang.org/protobuf/types/pluginpb"
)

type UnrayHandler struct {
	MethodRoute         string
	MethodHandlerPascal string
	RequestType         string
	ResponseType        string
}

type ServiceData struct {
	ServiceNamePascal string
	ServiceNameCamel  string
	UnaryHandlers     []UnrayHandler
}

type TemplateData struct {
	PackageName string
	Services    []ServiceData
}

func (r *GenerationRequest) Generate() (*plugin.CodeGeneratorResponse, error) {
	response := &plugin.CodeGeneratorResponse{}

	for _, name := range r.rq.GetFileToGenerate() {
		var fd *descriptorpb.FileDescriptorProto
		for _, fd = range r.rq.GetProtoFile() {
			if name == fd.GetName() {
				break
			}
		}
		if fd == nil {
			return nil, fmt.Errorf("could not find the .proto file for %s", name)
		}

		if fd.Options.GoPackage == nil || len(*fd.Options.GoPackage) == 0 {
			return nil, fmt.Errorf("go_package is not provided")
		}

		pkg := strings.TrimLeft(*fd.Options.GoPackage, ";./\\")

		data := &TemplateData{
			PackageName: pkg,
			Services:    []ServiceData{},
		}
		// For every serivce
		for _, svc := range fd.Service {
			sd := ServiceData{
				ServiceNamePascal: firstToUpper(*svc.Name),
				ServiceNameCamel:  firstToLower(*svc.Name),
				UnaryHandlers:     []UnrayHandler{},
			}
			for _, method := range svc.Method {
				// Skip non-unary methods
				if method.ClientStreaming != nil && *method.ClientStreaming {
					continue
				}
				if method.ServerStreaming != nil && *method.ServerStreaming {
					continue
				}
				sd.UnaryHandlers = append(sd.UnaryHandlers, UnrayHandler{
					RequestType:         strings.TrimLeft(*method.InputType, "."),
					ResponseType:        strings.TrimLeft(*method.OutputType, "."),
					MethodHandlerPascal: firstToUpper(*method.Name),
					MethodRoute:         toRoute(*method.Name),
				})
			}
			data.Services = append(data.Services, sd)
		}

		buf := bytes.Buffer{}
		if err := r.generateServerFile(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to generate server file: %v", err)
		}
		serverFileContent := buf.String()
		serverFileName := fmt.Sprintf("./%s.natsrpc_server.go", firstToLower(name))

		buf.Reset()
		if err := r.generateClientFile(&buf, data); err != nil {
			return nil, fmt.Errorf("failed to generate client file: %v", err)
		}
		clientFileContent := buf.String()
		clientFileName := fmt.Sprintf("./%s.natsrpc_client.go", firstToLower(name))

		response.File = append(response.File, &plugin.CodeGeneratorResponse_File{
			Content: &serverFileContent,
			Name:    &serverFileName,
		}, &plugin.CodeGeneratorResponse_File{
			Content: &clientFileContent,
			Name:    &clientFileName,
		})
	}

	return response, nil
}

func (r *GenerationRequest) generateServerFile(wr io.Writer, td *TemplateData) error {
	tmpl, err := template.New(".").Parse(serverTemplate)
	if err != nil {
		return err
	}
	return tmpl.Execute(wr, td)
}

func (r *GenerationRequest) generateClientFile(wr io.Writer, td *TemplateData) error {
	tmpl, err := template.New(".").Parse(clientTemplate)
	if err != nil {
		return err
	}
	return tmpl.Execute(wr, td)
}
