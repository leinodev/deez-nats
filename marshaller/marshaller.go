package marshaller

var (
	DefaultProtoMarshaller = NewProtoMarshaller()
	DefaultJsonMarshaller  = NewJsonMarshaller()
)

type PayloadMarshaller interface {
	Marshall(v *MarshalObject) ([]byte, error)
	Unmarshall(data []byte, v *MarshalObject) error
}

type Error struct {
	Text string
	Code int
}
type MarshalObject struct {
	Data any
	Err  *Error
}
