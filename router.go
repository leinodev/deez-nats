package natsrpcgo

type RPCRouter interface {
	Use(middlewares ...RpcMiddlewareFunc)

	AddRPC(method string, handler RpcHandleFunc, optsFuncs HandlerOptions, middlewares ...RpcMiddlewareFunc)

	Group(group string) RPCRouter
}
