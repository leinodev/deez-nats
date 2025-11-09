package natsrpcgo

import "sync"

type RHeaders interface {
	Get(k string) string
}
type RWHeaders interface {
	RHeaders
	Set(k string, v string)
	Del(k string)
}

type headersImpl struct {
	headers map[string]string
	m       sync.RWMutex
}

func (h *headersImpl) Get(k string) string {
	h.m.RLock()
	defer h.m.RUnlock()
	return h.headers[k]
}

func (h *headersImpl) Del(k string) {
	h.m.Lock()
	defer h.m.Unlock()
	delete(h.headers, k)
}

func (h *headersImpl) Set(k, v string) {
	h.m.Lock()
	defer h.m.Unlock()
	h.headers[k] = v
}

func newHeaders() *headersImpl {
	return &headersImpl{
		headers: map[string]string{},
	}
}
