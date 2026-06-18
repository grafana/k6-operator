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

func (poller *Poller) Start() {
	poller.mu.Lock()
	defer poller.mu.Unlock()

	if poller.running {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	poller.cancel = cancel
	poller.running = true

	ticker := time.NewTicker(poller.interval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				poller.OnDone()
				return

			case <-ticker.C:
				poller.OnInterval()
			}
		}
	}()
}

// Stop signals the poller to stop. It is non-blocking and idempotent: cancelling
// the context returns immediately even if OnInterval is currently executing (or
// stuck), so callers such as the reconciler are never wedged by a slow handler.
func (poller *Poller) Stop() {
	poller.mu.Lock()
	defer poller.mu.Unlock()

	if !poller.running {
		return
	}

	poller.cancel()
	poller.running = false
}
