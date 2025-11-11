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

	data, err := proto.Marshal(&ProtobufWrap{
		Data:  dataAny,
		Error: v.Error,
	})
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
		*anyTarget = *wrap.Data
	} else if protoTarget, ok := v.Data.(proto.Message); ok {
		if wrap.Data != nil {
			if err := wrap.Data.UnmarshalTo(protoTarget); err != nil {
				return err
			}
		}
	} else {
		return ErrProtoInvalidMessage
	}

	v.Error = wrap.Error
	return nil
}
