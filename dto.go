package natsrpc

type NatsError struct {
	Message string `json:"m"`
}
type NatsRPCResponse[T any] struct {
	Data  T `json:"d"`
	Error *NatsError
}

type NatsRPCRequest[T any] struct {
	Data T `json:"d"`
}
