package graceful

import (
	"context"
	"sync"
	"time"
)

type ShutdownManager struct {
	mu             sync.Mutex
	shutdown       bool
	parentCtx      context.Context
	shutdownCtx    context.Context
	shutdownCancel context.CancelFunc
	activeHandlers sync.WaitGroup
}

func NewShutdownManager() *ShutdownManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &ShutdownManager{
		shutdownCtx:    ctx,
		shutdownCancel: cancel,
	}
}

func (m *ShutdownManager) SetParentContext(parentCtx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.parentCtx != nil {
		// Already set, don't override
		return
	}

	m.parentCtx = parentCtx
	// Recreate shutdown context with parent
	m.shutdownCancel()
	ctx, cancel := context.WithCancel(parentCtx)
	m.shutdownCtx = ctx
	m.shutdownCancel = cancel
}

func (m *ShutdownManager) StartHandler() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.shutdown {
		return false
	}

	m.activeHandlers.Add(1)
	return true
}

func (m *ShutdownManager) FinishHandler() {
	m.activeHandlers.Done()
}

func (m *ShutdownManager) Shutdown(ctx context.Context) error {
	m.mu.Lock()
	if m.shutdown {
		m.mu.Unlock()
		return nil
	}
	m.shutdown = true
	m.shutdownCancel()
	m.mu.Unlock()

	done := make(chan struct{})
	go func() {
		m.activeHandlers.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (m *ShutdownManager) ShutdownWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return m.Shutdown(ctx)
}

func (m *ShutdownManager) IsShuttingDown() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.shutdown
}

func (m *ShutdownManager) ShutdownContext() context.Context {
	return m.shutdownCtx
}
