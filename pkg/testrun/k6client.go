package testrun

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/grafana/k6-operator/pkg/types"
	k6Client "go.k6.io/k6/api/v1/client"
)

// This will probably be removed once distributed mode in k6 is implemented.

func RunSetup(ctx context.Context, hostname string) (_ json.RawMessage, err error) {
	c, err := k6Client.New(fmt.Sprintf("%v:6565", hostname), k6Client.WithHTTPClient(&http.Client{
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
		c, err := k6Client.New(fmt.Sprintf("%v:6565", hostname), k6Client.WithHTTPClient(&http.Client{
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

	c, err := k6Client.New(fmt.Sprintf("%v:6565", hostnames[0]), k6Client.WithHTTPClient(&http.Client{
		Timeout: 0,
	}))
	if err != nil {
		return
	}

	return c.CallAPI(ctx, "POST", &url.URL{Path: "/v1/teardown"}, nil, nil)
}
