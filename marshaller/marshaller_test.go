package marshaller_test

import (
	"testing"

	"github.com/leinodev/deez-nats/marshaller"
)

func TestProto(t *testing.T) {
	content := &marshaller.ProtobufTestMessage{
		Name: "Hello World",
		Data: map[string]string{
			"test1": "test2",
			"test3": "test4",
		},
	}

	bytes, err := marshaller.DefaultProtoMarshaller.Marshall(&marshaller.MarshalObject{
		Data:  content,
		Error: "test",
	})
	if err != nil {
		t.Error(err)
		return
	}

	targetObject := &marshaller.MarshalObject{
		Data: &marshaller.ProtobufTestMessage{},
	}

	err = marshaller.DefaultProtoMarshaller.Unmarshall(bytes, targetObject)
	if err != nil {
		t.Error(err)
		return
	}
}
func BenchmarkProtoRoundTrip(b *testing.B) {
	content := &marshaller.ProtobufTestMessage{
		Name: "Benchmark Test Message",
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
			"key4": "value4",
			"key5": "value5",
		},
	}

	originalObj := &marshaller.MarshalObject{
		Data:  content,
		Error: "",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Marshall
		data, err := marshaller.DefaultProtoMarshaller.Marshall(originalObj)
		if err != nil {
			b.Fatalf("Marshall failed: %v", err)
		}

		// Unmarshall
		targetObj := &marshaller.MarshalObject{
			Data: &marshaller.ProtobufTestMessage{},
		}
		err = marshaller.DefaultProtoMarshaller.Unmarshall(data, targetObj)
		if err != nil {
			b.Fatalf("Unmarshall failed: %v", err)
		}
	}
}

func TestJson(t *testing.T) {
	content := &marshaller.JsonTestMessage{
		Name: "Hello World",
		Data: map[string]string{
			"test1": "test2",
			"test3": "test4",
		},
	}

	bytes, err := marshaller.DefaultJsonMarshaller.Marshall(&marshaller.MarshalObject{
		Data:  content,
		Error: "test",
	})
	if err != nil {
		t.Error(err)
		return
	}

	targetObject := &marshaller.MarshalObject{
		Data: &marshaller.JsonTestMessage{},
	}

	err = marshaller.DefaultJsonMarshaller.Unmarshall(bytes, targetObject)
	if err != nil {
		t.Error(err)
		return
	}
}
func BenchmarkJsonRoundTrip(b *testing.B) {
	content := &marshaller.JsonTestMessage{
		Name: "Benchmark Test Message",
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
			"key4": "value4",
			"key5": "value5",
		},
	}

	originalObj := &marshaller.MarshalObject{
		Data:  content,
		Error: "",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Marshall
		data, err := marshaller.DefaultJsonMarshaller.Marshall(originalObj)
		if err != nil {
			b.Fatalf("Marshall failed: %v", err)
		}

		// Unmarshall
		targetObj := &marshaller.MarshalObject{
			Data: &marshaller.JsonTestMessage{},
		}
		err = marshaller.DefaultJsonMarshaller.Unmarshall(data, targetObj)
		if err != nil {
			b.Fatalf("Unmarshall failed: %v", err)
		}
	}
}
