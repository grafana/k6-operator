package testrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/grafana/k6-operator/pkg/types"
	k6Client "go.k6.io/k6/v2/api/v1/client"
)

// This will probably be removed once distributed mode in k6 is implemented.

func RunSetup(ctx context.Context, hostname string) (_ json.RawMessage, err error) {
	c, err := k6Client.New(net.JoinHostPort(hostname, "6565"), k6Client.WithHTTPClient(&http.Client{
		Timeout: 0,
	}))
	if err != nil {
		return
	}

	var response types.SetupData
	if err = c.CallAPI(ctx, "POST", &url.URL{Path: "/v1/setup"}, nil, &response); err != nil {
		return nil, err
	}

	if response.Data.Attributes.Data != nil {
		var tmpSetupDataObj interface{}
		if err := json.Unmarshal(response.Data.Attributes.Data, &tmpSetupDataObj); err != nil {
			return nil, err
		}
	}

	return response.Data.Attributes.Data, nil
}

func SetSetupData(ctx context.Context, hostnames []string, data json.RawMessage) (err error) {
	for _, hostname := range hostnames {
		c, err := k6Client.New(net.JoinHostPort(hostname, "6565"), k6Client.WithHTTPClient(&http.Client{
			Timeout: 0,
		}))
		if err != nil {
			return err
		}

		if err = c.CallAPI(ctx, "PUT", &url.URL{Path: "/v1/setup"}, data, nil); err != nil {
			return err
		}
	}

	return nil
}

func RunTeardown(ctx context.Context, hostnames []string) (err error) {
	if len(hostnames) == 0 {
		return errors.New("no k6 Service is available to run teardown")
	}

	c, err := k6Client.New(net.JoinHostPort(hostnames[0], "6565"), k6Client.WithHTTPClient(&http.Client{
		Timeout: 0,
	}))
	if err != nil {
		return
	}

	return c.CallAPI(ctx, "POST", &url.URL{Path: "/v1/teardown"}, nil, nil)
}

// statusOptions control how a status change is sent to the runners.
type statusOptions struct {
	// concurrency is the maximum number of runners called at the same time.
	concurrency int
	// retries is the number of retries per runner after the first attempt.
	retries int
	// retryDelay is the delay before the first retry; it doubles on each retry.
	retryDelay time.Duration
	// timeout is the timeout of a single request.
	timeout time.Duration
}

var defaultStatusOptions = statusOptions{
	concurrency: 100,
	retries:     3,
	retryDelay:  time.Second,
	timeout:     10 * time.Second,
}

// ResumeRunners un-pauses all runners, i.e. it starts the test.
func ResumeRunners(ctx context.Context, hostnames []string) error {
	addrs := make([]string, len(hostnames))
	for i, hostname := range hostnames {
		addrs[i] = net.JoinHostPort(hostname, "6565")
	}

	req := types.StatusAPIRequest{
		Data: types.StatusAPIRequestData{
			Attributes: types.StatusAPIRequestDataAttributes{
				Paused: false,
			},
			ID:   "default",
			Type: "status",
		},
	}

	return patchStatus(ctx, addrs, req, defaultStatusOptions)
}

// patchStatus sends the status request to all addrs concurrently and
// returns the errors of all runners that could not be updated.
func patchStatus(ctx context.Context, addrs []string, req types.StatusAPIRequest, opts statusOptions) error {
	httpClient := &http.Client{Timeout: opts.timeout}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
		sem  = make(chan struct{}, opts.concurrency)
	)

	for _, addr := range addrs {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := patchStatusWithRetry(ctx, httpClient, addr, req, opts); err != nil {
				mu.Lock()
				errs = append(errs, fmt.Errorf("%s: %w", addr, err))
				mu.Unlock()
			}
		})
	}
	wg.Wait()

	return errors.Join(errs...)
}

func patchStatusWithRetry(ctx context.Context, httpClient *http.Client, addr string, req types.StatusAPIRequest, opts statusOptions) error {
	c, err := k6Client.New(addr, k6Client.WithHTTPClient(httpClient))
	if err != nil {
		return err
	}

	delay := opts.retryDelay
	for attempt := 0; ; attempt++ {
		err = c.CallAPI(ctx, "PATCH", &url.URL{Path: "/v1/status"}, req, nil)
		if err == nil || attempt >= opts.retries {
			return err
		}

		select {
		case <-ctx.Done():
			return errors.Join(err, ctx.Err())
		case <-time.After(delay):
		}
		delay *= 2
	}
}
