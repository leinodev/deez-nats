package natsrpc

import "errors"

var (
	ErrInvalidSubject         = registerError("invalid subject")
	ErrResponseAlreadyWritten = registerError("rpc response already written")
	ErrEmptyError             = registerError("cannot respond with empty error")
	ErrPoolBusy               = registerError("server handlers pool is full")
)

var (
	errNum = 1

	codeErr = map[int]error{}
	errCode = map[error]int{}
)

func registerError(text string) error {
	code := errNum
	errNum++

	err := errors.New(text)
	codeErr[code] = err
	errCode[err] = code

	return err
}
