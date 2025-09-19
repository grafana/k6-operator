package cloud

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/pkg/cloud/conn"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/cloudapi"
)

type TestRunPoller struct {
	*conn.Poller

	token     string
	host      string
	logger    logr.Logger
	testRunCh chan string

	Client *cloudapi.Client
}

func NewTestRunPoller(host, token, plzName string, logger logr.Logger) *TestRunPoller {
	// We need two loggers here because of logrus dependency in cloudapi.
	// This will have a re-visit during or after https://github.com/grafana/k6-operator/issues/571
	l := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}
	logrusLogger := l.WithFields(logrus.Fields{"k6_cloud_host": host})
	logger = logger.WithValues("k6_cloud_host", host)

	testRunsCh := make(chan string)

	poller := conn.NewPoller(10 * time.Second)
	poller.OnDone = func() {
		close(testRunsCh)
	}

	testRunPoller := TestRunPoller{
		Poller: poller,

		token:     token,
		host:      host,
		logger:    logger,
		testRunCh: testRunsCh,

		Client: cloudapi.NewClient(logrusLogger, token, host, "1.2.3", time.Duration(time.Minute)),
	}

	testRunPoller.OnInterval = func() {
		list, err := testRunPoller.getTestRuns(plzName)
		if err != nil {
			logger.Error(err, "Failed to get test runs from k6 Cloud.")
		} else {
			logger.Info(fmt.Sprintf("Retrieved test runs: %+v", list))

			for _, testRunId := range list {
				testRunsCh <- testRunId
			}
		}
	}

	return &testRunPoller
}

func (poller *TestRunPoller) GetTestRuns() chan string {
	return poller.testRunCh
}

func (poller *TestRunPoller) getTestRuns(plzName string) ([]string, error) {
	url := fmt.Sprintf("%s/v4/plz-test-runs?plz_name=%s", poller.host, plzName)
	req, err := poller.Client.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var list testRunList
	if err = poller.Client.Do(req, &list); err != nil {
		return nil, err
	}

	simplifiedList := make([]string, len(list.List))
	for i, item := range list.List {
		simplifiedList[i] = fmt.Sprintf("%d", item.ID)
	}

	return simplifiedList, nil
}

func getTestRun(client *cloudapi.Client, url string) (*TestRunData, error) {
	req, err := client.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var trData TestRunData
	if err = client.Do(req, &trData); err != nil {
		return nil, err
	}

	return &trData, nil
}

// called by PLZworker
func GetTestRunData(client *cloudapi.Client, refID string) (*TestRunData, error) {
	url := fmt.Sprintf("%s/loadtests/v4/test_runs(%s)?$select=id,run_status,k8s_load_zones_config,k6_runtime_config", strings.TrimSuffix(client.BaseURL(), "/v1"), refID)
	return getTestRun(client, url)
}

// called by TestRun controller
// If there's an error, it'll be logged.
func GetTestRunState(client *cloudapi.Client, refID string, logger logr.Logger) (TestRunStatus, error) {
	host := ApiURL(client.BaseURL())
	logger = logger.WithValues("k6_cloud_host", host)

	url := fmt.Sprintf("%s/loadtests/v4/test_runs(%s)?$select=id,run_status", host, refID)
	trData, err := getTestRun(client, url)
	if err != nil {
		logger.Error(err, "Failed to get test run state.")
		return TestRunStatus(cloudapi.RunStatusRunning), err
	}

	status := TestRunStatus(trData.RunStatus)
	logger.Info(fmt.Sprintf("Received test run status %v", status))

	return status, nil
}

// called by TestRun controller
// If there's an error, it'll be logged.
func SendTestRunEvents(client *cloudapi.Client, refID string, logger logr.Logger, events *Events) {
	if len(*events) == 0 {
		return
	}

	host := strings.TrimSuffix(client.BaseURL(), "/v1")
	logger = logger.WithValues("k6_cloud_host", host)

	url := fmt.Sprintf("%s/orchestrator/v1/testruns/%s/events", host, refID)
	req, err := client.NewRequest("POST", url, events)

	if err != nil {
		logger.Error(err, fmt.Sprintf("Failed to create events HTTP request %+v", events))
		return
	}

	logger.Info(fmt.Sprintf("Sending events to k6 Cloud %+v", *events))

	// status code is checked in Do
	if err = client.Do(req, nil); err != nil {
		logger.Error(err, fmt.Sprintf("Failed to send events %+v", events))
	}
}
