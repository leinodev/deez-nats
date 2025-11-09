package marshaller

var (
	DefaultProtoMarshaller = NewProtoMarshaller()
	DefaultJsonMarshaller  = NewJsonMarshaller()
)

type PayloadMarshaller interface {
	Marshall(v *MarshalObject) ([]byte, error)
	Unmarshall(data []byte, v *MarshalObject) error
}

type MarshalObject struct {
	Data    any
	Error   string
	Headers map[string]string
}
