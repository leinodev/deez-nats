package provider

import (
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type msgImpl struct {
	msg   *nats.Msg
	jsMsg jetstream.Msg
}

func (m *msgImpl) CoreMsg() *nats.Msg {
	return m.msg
}

func (m *msgImpl) JsMsg() jetstream.Msg {
	return m.jsMsg
}

func NewTransportMessage(msg *nats.Msg, jsMsg jetstream.Msg) TransportMessage {
	return &msgImpl{msg: msg, jsMsg: jsMsg}
}
