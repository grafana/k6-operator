package testrun

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/grafana/k6-operator/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RunTeardownNoHost(t *testing.T) {
	err := RunTeardown(context.Background(), []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no k6 Service is available to run teardown")
}

func Test_SetSetupDataNoHost(t *testing.T) {
	data := json.RawMessage(`{"foo":"bar"}`)
	err := SetSetupData(context.Background(), []string{}, data)
	assert.NoError(t, err)
}

// fakeRunner is a k6 REST API stub that records PATCH /v1/status requests.
type fakeRunner struct {
	server *httptest.Server
	// failures is the number of requests to fail before succeeding; -1 fails forever.
	failures int32
	calls    atomic.Int32
	inFlight *atomic.Int32
	maxSeen  *atomic.Int32
	got      atomic.Pointer[types.StatusAPIRequest]
}

func newFakeRunner(t *testing.T, failures int32) *fakeRunner {
	t.Helper()
	r := &fakeRunner{failures: failures, inFlight: &atomic.Int32{}, maxSeen: &atomic.Int32{}}
	r.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		n := r.inFlight.Add(1)
		defer r.inFlight.Add(-1)
		for {
			m := r.maxSeen.Load()
			if n <= m || r.maxSeen.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)

		if req.Method != http.MethodPatch || req.URL.Path != "/v1/status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		call := r.calls.Add(1)
		if r.failures < 0 || call <= r.failures {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"errors":[{"detail":"injected failure"}]}`))
			return
		}

		var body types.StatusAPIRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors":[{"detail":"bad body"}]}`))
			return
		}
		r.got.Store(&body)
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(r.server.Close)
	return r
}

func (r *fakeRunner) addr() string {
	return strings.TrimPrefix(r.server.URL, "http://")
}

var testStatusOptions = statusOptions{
	concurrency: 2,
	retries:     2,
	retryDelay:  time.Millisecond,
	timeout:     time.Second,
}

var resumeRequest = types.StatusAPIRequest{
	Data: types.StatusAPIRequestData{
		Attributes: types.StatusAPIRequestDataAttributes{Paused: false},
		ID:         "default",
		Type:       "status",
	},
}

func Test_patchStatus_AllRunners(t *testing.T) {
	var (
		runners []*fakeRunner
		addrs   []string
	)
	// share the counters so that the concurrency across all runners is measured
	inFlight, maxSeen := &atomic.Int32{}, &atomic.Int32{}
	for range 5 {
		r := newFakeRunner(t, 0)
		r.inFlight, r.maxSeen = inFlight, maxSeen
		runners = append(runners, r)
		addrs = append(addrs, r.addr())
	}

	err := patchStatus(context.Background(), addrs, resumeRequest, testStatusOptions)
	require.NoError(t, err)

	for _, r := range runners {
		assert.Equal(t, int32(1), r.calls.Load())
		require.NotNil(t, r.got.Load())
		assert.Equal(t, resumeRequest, *r.got.Load())
	}
	assert.LessOrEqual(t, maxSeen.Load(), int32(testStatusOptions.concurrency))
}

func Test_patchStatus_RetriesTransientFailure(t *testing.T) {
	r := newFakeRunner(t, 2)

	err := patchStatus(context.Background(), []string{r.addr()}, resumeRequest, testStatusOptions)
	require.NoError(t, err)
	assert.Equal(t, int32(3), r.calls.Load())
}

func Test_patchStatus_ReportsPermanentFailure(t *testing.T) {
	ok := newFakeRunner(t, 0)
	broken := newFakeRunner(t, -1)

	err := patchStatus(context.Background(), []string{ok.addr(), broken.addr()}, resumeRequest, testStatusOptions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), broken.addr())
	assert.NotContains(t, err.Error(), ok.addr())
	assert.Equal(t, int32(1), ok.calls.Load())
	assert.Equal(t, int32(1+testStatusOptions.retries), broken.calls.Load())
}

func Test_patchStatus_NoHost(t *testing.T) {
	err := patchStatus(context.Background(), []string{}, resumeRequest, testStatusOptions)
	assert.NoError(t, err)
}
