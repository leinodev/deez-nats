package marshaller

import "encoding/json"

type JsonTestMessage struct {
	Name string            `json:"name"`
	Data map[string]string `json:"data"`
}

type jsonErrWrap struct {
	Text string `json:"t"`
	Code int    `json:"c"`
}

type jsonWrap struct {
	Data any          `json:"d"`
	Err  *jsonErrWrap `json:"e"`
}

type jsonPayloadMarshaller struct {
}

func NewJsonMarshaller() PayloadMarshaller {
	return &jsonPayloadMarshaller{}
}
func (*jsonPayloadMarshaller) Marshall(v *MarshalObject) ([]byte, error) {
	wr := &jsonWrap{
		Data: v.Data,
	}
	if v.Err != nil {
		wr.Err = &jsonErrWrap{
			Text: v.Err.Text,
			Code: v.Err.Code,
		}
	}

	data, err := json.Marshal(wr)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (*jsonPayloadMarshaller) Unmarshall(data []byte, v *MarshalObject) error {
	wrap := jsonWrap{
		Data: v.Data,
	}

	if err := json.Unmarshal(data, &wrap); err != nil {
		return err
	}
	if wrap.Err != nil {
		v.Err = &Error{
			Text: wrap.Err.Text,
			Code: wrap.Err.Code,
		}
	}
	return nil
}
