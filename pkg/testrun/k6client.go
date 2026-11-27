package testrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/grafana/k6-operator/pkg/types"
	v1 "go.k6.io/k6/v2/api/v1"
	k6Client "go.k6.io/k6/v2/api/v1/client"
	"golang.org/x/sync/errgroup"
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

// ResumeRunners un-pauses all runners, i.e. it starts the test. Runners that
// are not paused are left as they are, see resumeRunner.
func ResumeRunners(ctx context.Context, hostnames []string) error {
	addrs := make([]string, len(hostnames))
	for i, hostname := range hostnames {
		addrs[i] = net.JoinHostPort(hostname, "6565")
	}

	return resumeRunners(ctx, addrs, defaultStatusOptions)
}

// resumeRunners resumes all addrs concurrently. It stops at the first runner
// that can't be resumed and returns its error: the test run fails anyway, so
// there is no point in waiting for the other runners.
func resumeRunners(ctx context.Context, addrs []string, opts statusOptions) error {
	httpClient := &http.Client{Timeout: opts.timeout}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(opts.concurrency)
	// Once gctx is cancelled, the remaining runners fail without sending a
	// request. Their errors make sure that Wait doesn't report success.
	for _, addr := range addrs {
		g.Go(func() error {
			if err := resumeRunnerWithRetry(gctx, httpClient, addr, opts); err != nil {
				return fmt.Errorf("%s: %w", addr, err)
			}
			return nil
		})
	}

	return g.Wait()
}

func resumeRunnerWithRetry(ctx context.Context, httpClient *http.Client, addr string, opts statusOptions) error {
	c, err := k6Client.New(addr, k6Client.WithHTTPClient(httpClient))
	if err != nil {
		return err
	}

	delay := opts.retryDelay
	for attempt := 0; ; attempt++ {
		err = resumeRunner(ctx, c)
		if err == nil || attempt >= opts.retries || !isRetryable(err) {
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

// resumeRunner resumes the runner if it is paused. k6 responds with an error
// to a resume request once the test has started, so the status is checked
// first: the runner may have been resumed already, e.g. by an earlier attempt
// whose response was lost, or it may have been started without --paused.
func resumeRunner(ctx context.Context, c *k6Client.Client) error {
	var status v1.StatusJSONAPI
	if err := c.CallAPI(ctx, "GET", &url.URL{Path: "/v1/status"}, nil, &status); err != nil {
		return err
	}
	// a runner that doesn't report its state is resumed to be sure that it starts
	if paused := status.Status().Paused; paused.Valid && !paused.Bool {
		return nil
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
	return c.CallAPI(ctx, "PATCH", &url.URL{Path: "/v1/status"}, req, nil)
}

// isRetryable reports whether a failed request may succeed when it is sent
// again. An error response from the k6 REST API means that the runner received
// the request and refused it, so sending it again won't help. Anything else is
// treated as a network issue: no connection, a timeout, or a response that
// doesn't come from k6 (e.g. a plain-text 503 from a service mesh proxy).
func isRetryable(err error) bool {
	_, isAPIErr := errors.AsType[v1.Error](err)
	return !isAPIErr
}
