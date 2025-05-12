package natsrpc

type NatsError struct {
	Message string `json:"m"`
}

type natsRPCResponse[T any] struct {
	Data  T `json:"d"`
	Error *NatsError
}

type natsRPCRequest[T any] struct {
	Data T `json:"d"`
}
