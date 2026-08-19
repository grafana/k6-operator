package controllers

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/logr"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestServiceStatusEndpoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		clusterIP string
		want      string
		wantErr   bool
	}{
		{
			name:      "IPv4",
			clusterIP: "10.96.0.42",
			want:      "http://10.96.0.42:6565/v1/status",
		},
		{
			name:      "IPv6",
			clusterIP: "2001:db8::42",
			want:      "http://[2001:db8::42]:6565/v1/status",
		},
		{
			name:      "headless service",
			clusterIP: "None",
			wantErr:   true,
		},
		{
			name:    "empty address",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := serviceStatusEndpoint(tt.clusterIP)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("serviceStatusEndpoint(%q) error = nil, want error", tt.clusterIP)
				}
				return
			}
			if err != nil {
				t.Fatalf("serviceStatusEndpoint(%q) error = %v", tt.clusterIP, err)
			}
			if got != tt.want {
				t.Errorf("serviceStatusEndpoint(%q) = %q, want %q", tt.clusterIP, got, tt.want)
			}
		})
	}
}

func TestProbeServiceStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantReady  bool
	}{
		{name: "success", statusCode: http.StatusNoContent, wantReady: true},
		{name: "redirect", statusCode: http.StatusTemporaryRedirect, wantReady: true},
		{name: "client error", statusCode: http.StatusBadRequest},
		{name: "server error", statusCode: http.StatusServiceUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
				if req.Method != http.MethodGet {
					t.Errorf("request method = %q, want GET", req.Method)
				}
				if req.URL.Path != "/v1/status" {
					t.Errorf("request path = %q, want /v1/status", req.URL.Path)
				}
				return &http.Response{
					StatusCode: tt.statusCode,
					Body:       io.NopCloser(strings.NewReader("")),
					Header:     make(http.Header),
					Request:    req,
				}, nil
			})}

			got := probeServiceStatus(
				context.Background(),
				logr.Discard(),
				client,
				"http://127.0.0.1:6565/v1/status",
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

	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}

	start := time.Now()
	ready := probeServiceStatus(
		context.Background(),
		logr.Discard(),
		client,
		"http://127.0.0.1:6565/v1/status",
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

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	client := &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		if !errors.Is(req.Context().Err(), context.Canceled) {
			t.Errorf("request context error = %v, want context.Canceled", req.Context().Err())
		}
		return nil, req.Context().Err()
	})}

	if probeServiceStatus(
		ctx,
		logr.Discard(),
		client,
		"http://127.0.0.1:6565/v1/status",
		time.Second,
	) {
		t.Error("probeServiceStatus() = true for a canceled reconcile context")
	}
}
