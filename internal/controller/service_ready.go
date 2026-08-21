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
	serviceStatusRequestTimeout = 2 * time.Second
)

func isServiceReady(ctx context.Context, log logr.Logger, service *v1.Service) bool {
	endpoint := "http://" + net.JoinHostPort(service.Spec.ClusterIP, defaultServicePort) + "/v1/status"

	return probeServiceStatus(ctx, log, endpoint, serviceStatusRequestTimeout)
}

func probeServiceStatus(
	ctx context.Context,
	log logr.Logger,
	endpoint string,
	timeout time.Duration,
) bool {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, endpoint, nil)
	if err != nil {
		log.Error(err, "Failed to build service status request", "endpoint", endpoint)
		return false
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err, "Failed to get service status", "endpoint", endpoint)
		return false
	}
	defer resp.Body.Close() //nolint:errcheck

	return resp.StatusCode < http.StatusBadRequest
}
