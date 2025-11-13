package middleware

func Apply[T any, M ~func(T) T](handler T, middlewares []M, reverse bool) T {
	if len(middlewares) == 0 {
		return handler
	}

	if reverse {
		for i := len(middlewares) - 1; i >= 0; i-- {
			handler = middlewares[i](handler)
		}
	} else {
		for _, mw := range middlewares {
			handler = mw(handler)
		}
	}

	return handler
}
