package natsrpcgo

type RPCRouter interface {
	Use(middlewares ...RpcMiddlewareFunc)

	AddRPC(method string, handler RpcHandleFunc, optsFuncs ...HandlerOptionFunc)
	Group(group string) RPCRouter
}
