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
	"go.k6.io/k6/lib/consts"
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
	logrusLogger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}

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

		Client: cloudapi.NewClient(logrusLogger, token, host, consts.Version, time.Duration(time.Minute)),
	}

	testRunPoller.Poller.OnInterval = func() {
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

// called by PLZ controller
func GetTestRunData(client *cloudapi.Client, refID string) (*TestRunData, error) {
	url := fmt.Sprintf("%s/loadtests/v4/test_runs(%s)?$select=id,run_status,k8s_load_zones_config,k6_runtime_config", strings.TrimSuffix(client.BaseURL(), "/v1"), refID)
	return getTestRun(client, url)
}

// called by TestRun controller
func GetTestRunState(client *cloudapi.Client, refID string, log logr.Logger) (TestRunStatus, error) {
	url := fmt.Sprintf("%s/loadtests/v4/test_runs(%s)?$select=id,run_status", ApiURL(client.BaseURL()), refID)
	trData, err := getTestRun(client, url)
	if err != nil {
		return TestRunStatus(cloudapi.RunStatusRunning), err
	}

	return TestRunStatus(trData.RunStatus), nil
}

// called by TestRun controller
// If there's an error, it'll be logged.
func SendTestRunEvents(client *cloudapi.Client, refID string, log logr.Logger, events *Events) {
	if len(*events) == 0 {
		return
	}

	url := fmt.Sprintf("%s/orchestrator/v1/testruns/%s/events", strings.TrimSuffix(client.BaseURL(), "/v1"), refID)
	req, err := client.NewRequest("POST", url, events)

	if err != nil {
		log.Error(err, fmt.Sprintf("Failed to create events HTTP request %+v", events))
		return
	}

	log.Info(fmt.Sprintf("Sending events to k6 Cloud %+v", *events))

	// status code is checked in Do
	if err = client.Do(req, nil); err != nil {
		log.Error(err, fmt.Sprintf("Failed to send events %+v", events))
	}
}
