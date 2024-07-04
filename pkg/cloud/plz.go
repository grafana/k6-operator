package cloud

import (
	"fmt"
	"strings"

	"go.k6.io/k6/cloudapi"
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
		return fmt.Errorf("Received error `%s`. Message from server `%s`", err.Error(), resp.Error.Message)
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

// temporary hack!
func ApiURL(k6CloudHostEnvVar string) string {
	url := defaultApiUrl
	if strings.Contains(k6CloudHostEnvVar, "staging") {
		url = "https://api.staging.k6.io"
	}
	return url
}
