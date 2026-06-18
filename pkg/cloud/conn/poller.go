package conn

import (
	"context"
	"sync"
	"time"
)

type Poller struct {
	interval time.Duration

	mu      sync.Mutex
	running bool
	ctx     context.Context
	cancel  context.CancelFunc

	OnInterval func()
	OnDone     func()
}

func NewPoller(interval time.Duration) *Poller {
	return &Poller{
		interval: interval,

		// by default, poller does nothing
		OnInterval: func() {},
		OnDone:     func() {},
	}
}

func (poller *Poller) IsPolling() bool {
	poller.mu.Lock()
	defer poller.mu.Unlock()

	return poller.running
}

// Context returns the context owned by the poller. When poller stops,
// it cancels the context. Use it to abort / cancel processes downstream.
func (poller *Poller) Context() context.Context {
	poller.mu.Lock()
	defer poller.mu.Unlock()

	if poller.ctx == nil {
		return context.Background()
	}

	return poller.ctx
}

func (poller *Poller) Start() {
	poller.mu.Lock()
	defer poller.mu.Unlock()

	if poller.running {
		return
	}

	// At the moment of writing, context.Background() is intentional here.
	// The only upstream context we can use here instead is the manager's one,
	// but wiring it up will complicate impl. with little benefit.
	poller.ctx, poller.cancel = context.WithCancel(context.Background())
	poller.running = true

	ticker := time.NewTicker(poller.interval)

	go func() {
		for {
			select {
			case <-poller.ctx.Done():
				ticker.Stop()
				poller.OnDone()
				return

			case <-ticker.C:
				poller.OnInterval()
			}
		}
	}()
}

// Stop signals the poller to stop. Non-blocking and idempotent: cancelling
// the context returns immediately even if OnInterval is currently executing (or
// stuck), so callers (reconciler) don't get stuck stuck by a handler.
func (poller *Poller) Stop() {
	poller.mu.Lock()
	defer poller.mu.Unlock()

	if !poller.running {
		return
	}

	poller.cancel()
	poller.running = false
}
