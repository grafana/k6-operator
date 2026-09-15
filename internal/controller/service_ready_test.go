package controllers

import (
	"context"
	"io"
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

// if the goal is to read the body of status response, it must not close prematurely
func TestRequestServiceStatusBodyRemainsReadable(t *testing.T) {
	t.Parallel()
	releaseBody := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		select {
		case <-releaseBody:
			_, _ = io.WriteString(w, "status body")
		case <-req.Context().Done():
		}
	}))
	defer server.Close()

	resp, err := requestServiceStatus(context.Background(), server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	close(releaseBody)
	body, err := io.ReadAll(resp.Body)
	if err != nil || string(body) != "status body" {
		t.Fatalf("body=%q error=%v", body, err)
	}
}

// headers arrive, but the body stalls, so timeout
func TestRequestServiceStatusBodyTimeout(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		<-req.Context().Done()
	}))
	defer server.Close()

	start := time.Now()
	resp, err := requestServiceStatus(context.Background(), server.URL, 100*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if _, err := io.ReadAll(resp.Body); err == nil {
		t.Fatal("expected response body timeout")
	}
	if time.Since(start) > time.Second {
		t.Fatal("response body read exceeded its timeout")
	}
}

// server unresponsive, so timeout
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

// The caller cancels context: don't wait for timeout
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
