package natsrpc

import (
	"context"
	"fmt"
	"sync"

	"github.com/leinodev/deez-nats/internal/utils"
	"github.com/nats-io/nats.go"
)

type request struct {
	handler RpcUnaryHandleFunc
	ctx     UnaryContext
}

type workerPool struct {
	root NatsRPC
	ctx  context.Context

	inChan chan *nats.Msg
	queue  chan *request

	workerGroup sync.WaitGroup

	handlerGroup sync.WaitGroup

	// Subject -> router
	routes map[string]*routeInfo
}

func newWorkerPool(root NatsRPC, ctx context.Context, workers, msgPoolSize int) *workerPool {
	pool := &workerPool{
		root:   root,
		ctx:    ctx,
		inChan: make(chan *nats.Msg, msgPoolSize),
		queue:  make(chan *request),
		routes: map[string]*routeInfo{},
	}
	go pool.scheduller()

	pool.workerGroup.Add(workers)
	for range workers {
		go pool.worker()
	}
	return pool
}

func (p *workerPool) scheduller() {
	for msg := range p.inChan {
		// Collect inbound messages and place then in queue
		route, ok := p.routes[msg.Subject]
		if !ok {
			continue
		}
		rq := &request{
			handler: utils.ApplyMiddlewares(route.handler, route.middlewares, true),
			ctx:     newRpcContext(p.root, p.ctx, msg, route.options),
		}
		select {
		case p.queue <- rq:
			continue
		default:
			rq.ctx.writeError(ErrPoolBusy)
		}

	}

}

func (p *workerPool) worker() {
	p.workerGroup.Done()
	for msg := range p.queue {
		// Process msg
		p.handlerGroup.Add(1)
		msg.ctx.run(msg.handler)
		p.handlerGroup.Done()
	}
}

func (p *workerPool) Stop(shutdownCtx context.Context) error {
	// Drain
	for _, route := range p.routes {
		route.sub.Drain()
	}

	// Wait for all handlers finished
	finished := make(chan struct{})
	go func() {
		p.handlerGroup.Wait()
		finished <- struct{}{}
		close(finished)
	}()

	select {
	case <-finished:
		break
	case <-shutdownCtx.Done():
		return fmt.Errorf("failed to wait for handlers finish: %w", context.DeadlineExceeded)
	}

	close(p.inChan)
	close(p.queue)
	return nil
}
