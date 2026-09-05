package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

func TestProbeServiceStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantReady  bool
	}{
		{name: "success", statusCode: http.StatusOK, wantReady: true},
		{name: "server error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != http.MethodGet {
					t.Errorf("request method = %q, want GET", req.Method)
				}
				if req.URL.Path != "/v1/status" {
					t.Errorf("request path = %q, want /v1/status", req.URL.Path)
				}
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			got := probeServiceStatus(
				context.Background(),
				logr.Discard(),
				server.URL+"/v1/status",
				time.Second,
			)
			if got != tt.wantReady {
				t.Errorf("probeServiceStatus() = %v, want %v", got, tt.wantReady)
			}
		})
	}
}

func TestProbeServiceStatusHonorsTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, req *http.Request) {
		<-req.Context().Done()
	}))
	defer server.Close()

	start := time.Now()
	ready := probeServiceStatus(
		context.Background(),
		logr.Discard(),
		server.URL+"/v1/status",
		20*time.Millisecond,
	)
	elapsed := time.Since(start)

	if ready {
		t.Error("probeServiceStatus() = true after request timeout")
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("probeServiceStatus() returned after %v, want a bounded timeout", elapsed)
	}
}

func TestProbeServiceStatusHonorsParentCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if probeServiceStatus(
		ctx,
		logr.Discard(),
		server.URL+"/v1/status",
		time.Second,
	) {
		t.Error("probeServiceStatus() = true for a canceled reconcile context")
	}
}
