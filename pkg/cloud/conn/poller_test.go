package conn

import (
	"testing"
	"time"
)

func Test_PollerStops(t *testing.T) {
	var x int

	poller := NewPoller(time.Millisecond * 100)
	poller.OnInterval = func() {
		// sleep for 200ms before changing the var so that each polling loop
		// takes twice longer than the ticker itself
		time.Sleep(time.Millisecond * 200)
		x++
	}

	pollerStopped := make(chan struct{})

	poller.Start()

	go func() {
		// If we stop after 300ms, then there should be ~ 2
		// polling loops done in total.
		time.Sleep(time.Millisecond * 300)
		poller.Stop()
		pollerStopped <- struct{}{}
	}()

	select {
	case <-pollerStopped:
		if poller.IsPolling() {
			t.Error("Poller is running even though pollerStopped was marked: something may be off with the test.")
		}

		if x == 0 {
			t.Error("Poller stopped correctly, but var's value hasn't been increased")
		}

	case <-time.After(time.Second * 10):
		t.Errorf("Poller failed to stop after 5 seconds. Poller status: %v", poller.IsPolling())
	}
}
