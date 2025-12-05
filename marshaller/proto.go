package marshaller

import (
	"errors"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

var (
	ErrProtoInvalidMessage = errors.New("content is not proto.Message")
)

type protoPayloadMarshaller struct {
}

func NewProtoMarshaller() PayloadMarshaller {
	return &protoPayloadMarshaller{}
}
func (*protoPayloadMarshaller) Marshall(v *MarshalObject) ([]byte, error) {
	var dataAny *anypb.Any
	var err error

	if v.Data == nil {
		dataAny = nil
	} else if protoContent, ok := v.Data.(*anypb.Any); ok {
		dataAny = protoContent
	} else if protoData, ok := v.Data.(proto.Message); ok {
		dataAny, err = anypb.New(protoData)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, ErrProtoInvalidMessage
	}

	wrap := &ProtobufWrap{
		Data: dataAny,
	}
	if v.Err != nil {
		wrap.Err = &ProtobufErrorWrap{
			Text: v.Err.Text,
			Code: int32(v.Err.Code),
		}
	}

	data, err := proto.Marshal(wrap)
	if err != nil {
		return nil, err
	}
	return data, nil
}
func (*protoPayloadMarshaller) Unmarshall(data []byte, v *MarshalObject) error {
	var wrap ProtobufWrap
	if err := proto.Unmarshal(data, &wrap); err != nil {
		return err
	}

	if anyTarget, ok := v.Data.(*anypb.Any); ok {
		if anyTarget == nil {
			return ErrProtoInvalidMessage
		}
		anyTarget.Reset()
		if wrap.Data != nil {
			anyTarget.TypeUrl = wrap.Data.TypeUrl
			anyTarget.Value = append(anyTarget.Value[:0], wrap.Data.Value...)
		}
	} else if protoTarget, ok := v.Data.(proto.Message); ok {
		if wrap.Data != nil {
			if err := wrap.Data.UnmarshalTo(protoTarget); err != nil {
				return err
			}
		}
	} else {
		return ErrProtoInvalidMessage
	}

	if wrap.Err != nil {
		v.Err = &Error{
			Text: wrap.Err.Text,
			Code: int(wrap.Err.Code),
		}
	}

	return nil
}
