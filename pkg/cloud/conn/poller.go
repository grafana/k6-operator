package conn

import (
	"time"
)

type Poller struct {
	interval time.Duration
	running  bool
	done     chan bool

	OnInterval func()
	OnDone     func()
}

func NewPoller(interval time.Duration) *Poller {
	return &Poller{
		interval: interval,
		running:  false,
		done:     make(chan bool),

		// by default, poller does nothing
		OnInterval: func() {},
		OnDone:     func() {},
	}
}

func (poller Poller) IsPolling() bool {
	return poller.running
}

func (poller *Poller) Start() {
	ticker := time.NewTicker(poller.interval)

	go func() {
		for {
			select {
			case <-poller.done:
				ticker.Stop()
				poller.OnDone()
				return

			case <-ticker.C:
				poller.OnInterval()
			}
		}
	}()

	poller.running = true
}

func (poller *Poller) Stop() {
	if poller.IsPolling() {
		poller.done <- true

		poller.running = false
	}
}
