package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"go.k6.io/k6/v2/cloudapi"
)

const (
	defaultApiUrl    = "https://api.k6.io"
	defaultIngestUrl = "https://ingest.k6.io"
)

func RegisterPLZ(client *cloudapi.Client, data PLZRegistrationData) error {
	url := fmt.Sprintf("%s/cloud-resources/v1/load-zones", strings.TrimSuffix(client.BaseURL(), "/v1"))

	data.LZConfig = LZConfig{
		RunnerImage: data.RunnerImage,
	}

	req, err := client.NewRequest("POST", url, data)
	if err != nil {
		return err
	}

	var resp struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err = client.Do(req, &resp); err != nil {
		return fmt.Errorf("received error `%s`. Message from server `%s`", err.Error(), resp.Error.Message)
	}

	return nil
}

func DeRegisterPLZ(client *cloudapi.Client, name string) error {
	url := fmt.Sprintf("%s/cloud-resources/v1/load-zones/%s", strings.TrimSuffix(client.BaseURL(), "/v1"), name)

	req, err := client.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	return client.Do(req, nil)
}

const (
	apiURLEnvVar             = "K6_CLOUD_API_URL"
	tokenExchangeURLEnvVar   = "K6_CLOUD_TOKEN_EXCHANGE_URL"
	tokenExchangeTokenEnvVar = "K6_CLOUD_TOKEN_EXCHANGE_TOKEN"
)

func init() {
	if url := os.Getenv(tokenExchangeURLEnvVar); url != "" {
		http.DefaultTransport = exchangeTransport{http.DefaultTransport, url, os.Getenv(tokenExchangeTokenEnvVar)}
	}
}

type exchangeTransport struct {
	http.RoundTripper
	url, token string
}

func (t exchangeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") == "" {
		return t.RoundTripper.RoundTrip(req)
	}
	accessToken, err := t.exchange(req.Context())
	if err != nil {
		return nil, err
	}
	req = req.Clone(req.Context())
	req.Header.Del("Authorization")
	req.Header.Set("X-Access-Token", accessToken)
	return t.RoundTripper.RoundTrip(req)
}

func (t exchangeTransport) exchange(ctx context.Context) (string, error) {
	body := strings.NewReader(`{"namespace":"*","audiences":["apiextensions.k8s.io"]}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+t.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := t.RoundTripper.RoundTrip(req)
	if err != nil {
		return "", fmt.Errorf("token exchange: %w", err)
	}
	defer resp.Body.Close()

	var out struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.Data.Token == "" {
		return "", fmt.Errorf("token exchange failed: %s %s", resp.Status, out.Error)
	}
	return out.Data.Token, nil
}

// temporary hack!
func ApiURL(k6CloudHostEnvVar string) string {
	if url := os.Getenv(apiURLEnvVar); url != "" {
		return strings.TrimRight(url, "/")
	}
	url := defaultApiUrl
	if strings.Contains(k6CloudHostEnvVar, "staging") {
		url = "https://api.staging.k6.io"
	}
	return url
}

func K6CloudHost() string {
	host, ok := os.LookupEnv("K6_CLOUD_HOST")
	if !ok {
		return "https://ingest.k6.io"
	}

	return host
}
