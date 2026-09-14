package controllers

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
)

const (
	defaultServicePort          = "6565"
	serviceStatusRequestTimeout = time.Second
)

func isServiceReady(ctx context.Context, log logr.Logger, service *v1.Service) bool {
	return probeServiceStatus(ctx, log, serviceStatusURL(service), serviceStatusRequestTimeout)
}

func serviceStatusURL(service *v1.Service) string {
	return "http://" + net.JoinHostPort(service.Spec.ClusterIP, defaultServicePort) + "/v1/status"
}

// The caller owns the response body. Client.Timeout bounds body reads too,
// without canceling the request context when this helper returns.
func requestServiceStatus(ctx context.Context, endpoint string, timeout time.Duration) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	return (&http.Client{Timeout: timeout}).Do(req)
}

func probeServiceStatus(ctx context.Context, log logr.Logger, endpoint string, timeout time.Duration) bool {
	resp, err := requestServiceStatus(ctx, endpoint, timeout)
	if err != nil {
		log.Error(err, "Failed to get service status", "endpoint", endpoint)
		return false
	}
	defer resp.Body.Close() //nolint:errcheck

	return resp.StatusCode < http.StatusBadRequest
}
