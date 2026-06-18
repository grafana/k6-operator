package conn

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// The happy path test that checks that poller executes and
// finishes cleanly when OnInterval callback behaves well.
func Test_PollerStop(t *testing.T) {
	var x atomic.Int32

	ranOnce := make(chan struct{}) // a signal that poller can be stopped

	poller := NewPoller(time.Millisecond * 100)
	poller.OnInterval = func() {
		// run the handler a couple of times before stopping the poller
		if x.Add(1) == 2 {
			close(ranOnce)
		}
	}

	pollerStopped := make(chan struct{})

	poller.Start()

	go func() {
		// blocks until OnInterval signals it's run for a bit
		<-ranOnce
		poller.Stop()
		pollerStopped <- struct{}{}
	}()

	select {
	case <-pollerStopped:
		if poller.IsPolling() {
			t.Error("Poller is running even though pollerStopped was marked: something may be off with the test.")
		}

		if x.Load() == 0 {
			t.Error("Poller stopped correctly, but var's value hasn't been increased")
		}

	case <-time.After(time.Second * 10):
		t.Errorf("Poller failed to stop after 10 seconds. Poller status: %v", poller.IsPolling())
	}
}

// The un-happy path test which checks that the poller can successfully
// stop even on a faulty handler in OnInterval callback. This prevents
// the leaking of goroutines and stuck reconciler.
func Test_PollerStop_OnDeadlock(t *testing.T) {
	poller := NewPoller(time.Millisecond * 10)

	var once sync.Once
	startedHandler := make(chan struct{}) // a signal that the stuck handler has started executing
	blockHandler := make(chan struct{})   // simulates a stuck handler
	defer close(blockHandler)             // cleanup

	poller.OnInterval = func() {
		// Mimic `testRunsCh <- testRunId` blocking on a consumer that never reads.
		once.Do(func() { close(startedHandler) })
		<-blockHandler
	}

	poller.Start()

	// We need to call poller.Stop() only after OnInterval() got into stuck state.
	// To ensure that, blocking here until OnInterval() closes the channel and therefore,
	// signals that it started executing.
	<-startedHandler

	stopReturned := make(chan struct{})
	go func() {
		// try to stop the poller
		poller.Stop()
		close(stopReturned)
	}()

	select {
	case <-stopReturned:
		// All OK: Stop() managed to complete despite OnInterval() call hanging.
	case <-time.After(time.Second * 2):
		t.Fatal("Poller failed to stop because OnInterval callback is blocked. " +
			"Both the poller and the handler goroutines leak and the reconciler stalls.")
	}
}
