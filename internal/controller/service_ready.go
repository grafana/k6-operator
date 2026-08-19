package controllers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-logr/logr"
	v1 "k8s.io/api/core/v1"
)

const (
	serviceStatusPort           = "6565"
	serviceStatusRequestTimeout = 2 * time.Second
)

type httpDoer interface {
	Do(*http.Request) (*http.Response, error)
}

func serviceStatusEndpoint(clusterIP string) (string, error) {
	if net.ParseIP(clusterIP) == nil {
		return "", fmt.Errorf("invalid service ClusterIP %q", clusterIP)
	}

	return fmt.Sprintf("http://%s/v1/status", net.JoinHostPort(clusterIP, serviceStatusPort)), nil
}

func isServiceReady(ctx context.Context, log logr.Logger, service *v1.Service) bool {
	endpoint, err := serviceStatusEndpoint(service.Spec.ClusterIP)
	if err != nil {
		log.Error(err, "Failed to build service status endpoint", "service", service.Name)
		return false
	}

	return probeServiceStatus(ctx, log, http.DefaultClient, endpoint, serviceStatusRequestTimeout)
}

func probeServiceStatus(
	ctx context.Context,
	log logr.Logger,
	client httpDoer,
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

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err, "Failed to get service status", "endpoint", endpoint)
		return false
	}
	defer resp.Body.Close() //nolint:errcheck

	return resp.StatusCode < http.StatusBadRequest
}
