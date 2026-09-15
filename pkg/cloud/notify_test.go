package cloud

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"go.k6.io/k6/v2/cloudapi"
)

func TestNotifyTestRunSuccess(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/provisioning/v1/test_runs/42/notify" {
			t.Errorf("path = %s", r.URL.Path)
		}
		if r.UserAgent() != "k6-operator" {
			t.Errorf("user agent = %q", r.UserAgent())
		}
		if got := r.Header.Get("Authorization"); got != "Bearer run-token" {
			t.Errorf("authorization = %q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["event_type"] != scriptExecutionCompletedEvent {
			t.Errorf("event_type = %v", body["event_type"])
		}
		if errValue, ok := body["error"]; !ok || errValue != nil {
			t.Errorf("error = %v, present = %t; want explicit null", errValue, ok)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := newNotifyTestClient(server.URL)
	if err := NotifyTestRun(context.Background(), client, "run-token", "42", nil); err != nil {
		t.Fatal(err)
	}
}

func TestNotifyTestRunError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			EventType string `json:"event_type"`
			Error     struct {
				Code   int32  `json:"code"`
				Reason string `json:"reason"`
				Detail string `json:"detail"`
			} `json:"error"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.EventType != scriptExecutionCompletedEvent {
			t.Errorf("event_type = %s", body.EventType)
		}
		if body.Error.Code != int32(ScriptAborted) {
			t.Errorf("code = %d", body.Error.Code)
		}
		if body.Error.Reason != "script aborted" {
			t.Errorf("reason = %q", body.Error.Reason)
		}
		if body.Error.Detail != "runner pod exited" {
			t.Errorf("detail = %q", body.Error.Detail)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := NotifyTestRun(context.Background(), newNotifyTestClient(server.URL), "run-token", "42",
		&TestRunNotification{
			Code:   ScriptAborted,
			Reason: "script aborted",
			Detail: "runner pod exited",
		})
	if err != nil {
		t.Fatal(err)
	}
}

func TestNotifyAPIHost(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "api host", host: "https://api.k6.io", want: "https://api.k6.io"},
		{name: "ingest host", host: "https://ingest.k6.io", want: "https://api.k6.io"},
		{name: "staging ingest host", host: "https://ingest.staging.k6.io", want: "https://api.staging.k6.io"},
		{name: "test server", host: "http://127.0.0.1:12345", want: "http://127.0.0.1:12345"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := notifyAPIHost(newNotifyTestClient(tt.host)); got != tt.want {
				t.Errorf("notifyAPIHost() = %q, want %q", got, tt.want)
			}
		})
	}
}

func newNotifyTestClient(host string) *cloudapi.Client {
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	return cloudapi.NewClient(logger, "legacy-token", host, "test", time.Second)
}

func TestNotifyTestRunHTTPError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()
	if err := NotifyTestRun(context.Background(), newNotifyTestClient(server.URL), "run-token", "42", nil); err == nil {
		t.Fatal("expected HTTP error")
	}
}
