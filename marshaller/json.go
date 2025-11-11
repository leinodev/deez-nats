package marshaller

import "encoding/json"

type JsonTestMessage struct {
	Name string            `json:"name"`
	Data map[string]string `json:"data"`
}

type jsonWrap struct {
	Data any    `json:"d"`
	Err  string `json:"e"`
}

type jsonPayloadMarshaller struct {
}

func NewJsonMarshaller() PayloadMarshaller {
	return &jsonPayloadMarshaller{}
}
func (*jsonPayloadMarshaller) Marshall(v *MarshalObject) ([]byte, error) {
	data, err := json.Marshal(&jsonWrap{
		Data: v.Data,
		Err:  v.Error,
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (*jsonPayloadMarshaller) Unmarshall(data []byte, v *MarshalObject) error {
	var wrap jsonWrap
	wrap.Data = v.Data

	if err := json.Unmarshal(data, &wrap); err != nil {
		return err
	}
	v.Error = wrap.Err
	return nil
}
