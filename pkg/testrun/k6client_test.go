package testrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/grafana/k6-operator/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "go.k6.io/k6/v2/api/v1"
	k6Client "go.k6.io/k6/v2/api/v1/client"
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

// fakeRunner is a k6 REST API stub of a runner started with --paused. Like
// k6, it refuses to resume a runner that is not paused.
type fakeRunner struct {
	server *httptest.Server
	paused atomic.Bool
	// failures is the number of PATCH requests to fail before succeeding; -1 fails forever.
	failures int32
	// fail writes the response of a failed request; k6APIError by default.
	fail func(w http.ResponseWriter)
	// getFailures is the number of GET requests to fail with proxyError.
	getFailures int32
	// loseResponse drops the connection after the first resume, as if its response were lost.
	loseResponse bool
	gets         atomic.Int32
	patches      atomic.Int32
	inFlight     *atomic.Int32
	maxSeen      *atomic.Int32
	got          atomic.Pointer[types.StatusAPIRequest]
}

func newFakeRunner(t *testing.T, failures int32) *fakeRunner {
	t.Helper()
	r := &fakeRunner{failures: failures, fail: k6APIError, inFlight: &atomic.Int32{}, maxSeen: &atomic.Int32{}}
	r.paused.Store(true)
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

		if req.URL.Path != "/v1/status" {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch req.Method {
		case http.MethodGet:
			if r.gets.Add(1) <= r.getFailures {
				proxyError(w)
				return
			}
			_, _ = fmt.Fprintf(w, `{"data":{"type":"status","id":"default","attributes":{"paused":%t}}}`, r.paused.Load())

		case http.MethodPatch:
			call := r.patches.Add(1)
			if r.failures < 0 || call <= r.failures {
				r.fail(w)
				return
			}

			var body types.StatusAPIRequest
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"errors":[{"detail":"bad body"}]}`))
				return
			}
			if !r.paused.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"errors":[{"title":"Pause error","detail":"test execution wasn't paused"}]}`))
				return
			}
			r.got.Store(&body)
			r.paused.Store(false)
			if r.loseResponse && call == 1 {
				dropConnection(w)
				return
			}
			_, _ = w.Write([]byte(`{}`))

		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	t.Cleanup(r.server.Close)
	return r
}

// k6APIError responds like the k6 REST API does when it refuses a request.
func k6APIError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(`{"errors":[{"detail":"injected failure"}]}`))
}

// proxyError responds like a proxy in front of the runner, e.g. a service mesh sidecar.
func proxyError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = w.Write([]byte("no healthy upstream"))
}

// dropConnection closes the connection without a response.
func dropConnection(w http.ResponseWriter) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		panic(err)
	}
	_ = conn.Close()
}

// slowResponse responds after the request timeout used in the tests.
func slowResponse(w http.ResponseWriter) {
	time.Sleep(200 * time.Millisecond)
	k6APIError(w)
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

func Test_resumeRunners_AllRunners(t *testing.T) {
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

	err := resumeRunners(context.Background(), addrs, testStatusOptions)
	require.NoError(t, err)

	for _, r := range runners {
		assert.Equal(t, int32(1), r.patches.Load())
		require.NotNil(t, r.got.Load())
		assert.Equal(t, resumeRequest, *r.got.Load())
	}
	assert.LessOrEqual(t, maxSeen.Load(), int32(testStatusOptions.concurrency))
}

func Test_resumeRunners_RetriesNetworkFailure(t *testing.T) {
	tests := []struct {
		name    string
		fail    func(http.ResponseWriter)
		timeout time.Duration
	}{
		{name: "dropped connection", fail: dropConnection},
		{name: "proxy error", fail: proxyError},
		{name: "timeout", fail: slowResponse, timeout: 50 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newFakeRunner(t, 2)
			r.fail = tt.fail
			opts := testStatusOptions
			if tt.timeout != 0 {
				opts.timeout = tt.timeout
			}

			err := resumeRunners(context.Background(), []string{r.addr()}, opts)
			require.NoError(t, err)
			assert.Equal(t, int32(3), r.patches.Load())
		})
	}
}

func Test_resumeRunners_SkipsRunnerThatIsNotPaused(t *testing.T) {
	// e.g. resumed by an earlier reconcile, or started without --paused
	r := newFakeRunner(t, 0)
	r.paused.Store(false)

	err := resumeRunners(context.Background(), []string{r.addr()}, testStatusOptions)
	require.NoError(t, err)
	assert.Equal(t, int32(1), r.gets.Load())
	assert.Equal(t, int32(0), r.patches.Load())
}

func Test_resumeRunners_LostResponse(t *testing.T) {
	r := newFakeRunner(t, 0)
	r.loseResponse = true

	err := resumeRunners(context.Background(), []string{r.addr()}, testStatusOptions)
	require.NoError(t, err)
	assert.False(t, r.paused.Load())
	assert.Equal(t, int32(1), r.patches.Load(), "the retry must not resume the runner again")
}

func Test_resumeRunners_RetriesStatusFailure(t *testing.T) {
	r := newFakeRunner(t, 0)
	r.getFailures = 2

	err := resumeRunners(context.Background(), []string{r.addr()}, testStatusOptions)
	require.NoError(t, err)
	assert.Equal(t, int32(3), r.gets.Load())
	assert.Equal(t, int32(1), r.patches.Load())
}

func Test_resumeRunners_DoesNotRetryAPIError(t *testing.T) {
	r := newFakeRunner(t, -1)

	err := resumeRunners(context.Background(), []string{r.addr()}, testStatusOptions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "injected failure")
	assert.Equal(t, int32(1), r.patches.Load())
}

func Test_resumeRunners_ReportsPermanentFailure(t *testing.T) {
	ok := newFakeRunner(t, 0)
	broken := newFakeRunner(t, -1)
	broken.fail = dropConnection

	err := resumeRunners(context.Background(), []string{ok.addr(), broken.addr()}, testStatusOptions)
	require.Error(t, err)
	assert.Contains(t, err.Error(), broken.addr())
	assert.NotContains(t, err.Error(), ok.addr())
	assert.Equal(t, int32(1), ok.patches.Load())
	assert.Equal(t, int32(1+testStatusOptions.retries), broken.patches.Load())
}

func Test_isRetryable(t *testing.T) {
	apiErr := v1.Error{Status: "500", Title: "Pause error"}

	// a runner that is not listening (yet)
	closed := httptest.NewServer(http.NotFoundHandler())
	closed.Close()
	c, err := k6Client.New(strings.TrimPrefix(closed.URL, "http://"))
	require.NoError(t, err)
	connRefused := c.CallAPI(context.Background(), "PATCH", &url.URL{Path: "/v1/status"}, resumeRequest, nil)
	require.ErrorIs(t, connRefused, syscall.ECONNREFUSED)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "k6 API error", err: apiErr, want: false},
		{name: "wrapped k6 API error", err: fmt.Errorf("runner: %w", apiErr), want: false},
		{name: "joined k6 API error", err: errors.Join(errors.New("other runner"), apiErr), want: false},
		{name: "connection refused", err: connRefused, want: true},
		{name: "deadline exceeded", err: context.DeadlineExceeded, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, isRetryable(tt.err))
		})
	}
}

func Test_resumeRunners_NoHost(t *testing.T) {
	err := resumeRunners(context.Background(), []string{}, testStatusOptions)
	assert.NoError(t, err)
}
