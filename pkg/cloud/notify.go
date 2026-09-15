package cloud

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	openapi "github.com/grafana/k6-cloud-openapi-client-go/k6"
	"go.k6.io/k6/v2/cloudapi"
)

const scriptExecutionCompletedEvent = "script_execution_completed"

type TestRunNotification struct {
	Code   ErrorCode
	Reason string
	Detail string
}

func NotifyTestRun(
	ctx context.Context,
	client *cloudapi.Client,
	token string,
	refID string,
	testRunNotification *TestRunNotification,
) error {
	if client == nil {
		return errors.New("cloud client is nil")
	}
	if token == "" {
		return errors.New("test run token is empty")
	}

	id, err := strconv.ParseInt(refID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid test run ID %q: %w", refID, err)
	}

	cfg := openapi.NewConfiguration()
	cfg.UserAgent = "k6-operator"
	cfg.HTTPClient = &http.Client{Timeout: time.Minute}
	cfg.Servers = []openapi.ServerConfiguration{{URL: notifyAPIHost(client)}}

	notification := openapi.NewScriptExecutionCompletedNotificationApiModel(scriptExecutionCompletedEvent)
	if testRunNotification == nil {
		notification.SetErrorNil()
	} else {
		errModel := openapi.NewScriptExecutionError(int32(testRunNotification.Code), testRunNotification.Reason)
		if testRunNotification.Detail != "" {
			errModel.SetDetail(testRunNotification.Detail)
		}
		notification.SetError(*errModel)
	}

	ctx = context.WithValue(ctx, openapi.ContextAccessToken, token)
	_, err = openapi.NewAPIClient(cfg).
		ProvisioningAPI.
		TestRunsNotify(ctx, id).
		ScriptExecutionCompletedNotificationApiModel(notification).
		Execute()
	return err
}

func notifyAPIHost(client *cloudapi.Client) string {
	host := strings.TrimSuffix(client.BaseURL(), "/v1")
	if strings.Contains(host, "ingest") {
		return ApiURL(host)
	}
	return host
}
