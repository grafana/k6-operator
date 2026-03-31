package plz

import (
	"fmt"
	rand "math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func newTestWorker() *PLZWorker {
	c, _ := client.New(nil, client.Options{})
	return NewPLZWorker(&v1alpha1.PrivateLoadZone{}, "token", c, logr.Logger{})
}

func Test_PLZWorkers_AddWorker(t *testing.T) {
	var workers PLZWorkers
	w := newTestWorker()

	err := workers.AddWorker("foo", w)
	assert.NoError(t, err)
}

func Test_PLZWorkers_AddWorker_duplicate(t *testing.T) {
	var workers PLZWorkers
	w := newTestWorker()

	require.NoError(t, workers.AddWorker("foo", w))
	err := workers.AddWorker("foo", w)
	assert.Error(t, err)
}

func Test_PLZWorkers_GetWorker(t *testing.T) {
	var workers PLZWorkers
	w := newTestWorker()
	require.NoError(t, workers.AddWorker("foo", w))

	got, err := workers.GetWorker("foo")
	assert.NoError(t, err)
	assert.Equal(t, w, got)
}

func Test_PLZWorkers_GetWorker_missing(t *testing.T) {
	var workers PLZWorkers
	_, err := workers.GetWorker("nonexistent")
	assert.Error(t, err)
}

func Test_PLZWorkers_DeleteWorker(t *testing.T) {
	var workers PLZWorkers
	w := newTestWorker()
	require.NoError(t, workers.AddWorker("foo", w))

	workers.DeleteWorker("foo")

	_, err := workers.GetWorker("foo")
	assert.Error(t, err)
}

func Test_PLZWorkers_DeleteWorker_nonexistent(t *testing.T) {
	var workers PLZWorkers
	// should not panic
	workers.DeleteWorker("nonexistent")
}

func Test_PLZWorkers_ConcurrentAdd_differentKeys(t *testing.T) {
	var workers PLZWorkers
	var wg sync.WaitGroup

	const n = 20
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := newTestWorker()
			err := workers.AddWorker(fmt.Sprintf("plz-%d", i), w)
			assert.NoError(t, err)
		}()
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		_, err := workers.GetWorker(fmt.Sprintf("plz-%d", i))
		assert.NoError(t, err)
	}
}

func Test_PLZWorkers_ConcurrentAdd_sameKey(t *testing.T) {
	var workers PLZWorkers
	var wg sync.WaitGroup
	var successes atomic.Int32

	const n = 20
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := newTestWorker()
			if err := workers.AddWorker("contested", w); err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()

	assert.Equal(t, int32(1), successes.Load(), "exactly one AddWorker should succeed")

	_, err := workers.GetWorker("contested")
	assert.NoError(t, err)
}

func Test_PLZWorkers_ConcurrentAllsOps(t *testing.T) {
	var workers PLZWorkers

	const n = 20
	keys := make([]string, n)
	for i := 0; i < n; i++ {
		keys[i] = fmt.Sprintf("plz-%d", i)
	}

	// add all workers first
	for _, key := range keys {
		require.NoError(t, workers.AddWorker(key, newTestWorker()))
	}

	// each goroutine picks a random key and random operation
	var wg sync.WaitGroup
	for i := 0; i < n*3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := keys[rand.IntN(n)]
			switch rand.IntN(3) {
			case 0:
				workers.AddWorker(key, newTestWorker()) //nolint:errcheck
			case 1:
				workers.GetWorker(key) //nolint:errcheck
			case 2:
				workers.DeleteWorker(key)
			}
		}()
	}

	wg.Wait()
}
