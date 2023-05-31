package cloud

import (
	"fmt"
	"math/rand"
	"os"
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

func NewTestRunPoller(host, token string, logger logr.Logger) *TestRunPoller {
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
		list, err := testRunPoller.getTestRuns()
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

func (poller *TestRunPoller) getTestRuns() ([]string, error) {
	url := poller.host + "/get-tests" // TODO
	req, err := poller.Client.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var list testRunList
	if err = poller.Client.Do(req, &list); err != nil {
		return nil, err
	}

	return list.List, nil
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

func GetTestRunData(client *cloudapi.Client, refID string) (*TestRunData, error) {
	// url := fmt.Sprintf("https://%s/loadtests/v4/test_runs(%s)?select=id,run_status,k8s_load_zones_config", client.Host, refID)
	// return getTestRun(client, url)
	return &TestRunData{
		TestRunId: refID,
		LZConfig: LZConfig{
			RunnerImage:   "grafana/k6:latest",
			InstanceCount: 1,
		},
	}, nil
}

func GetTestRunState(client *cloudapi.Client, refID string, log logr.Logger) (TestRunStatus, error) {
	// url := fmt.Sprintf("https://%s/loadtests/v4/test_runs(%s)?select=id,run_status", client.Host, refID)
	// trData, err := getTestRun(client, url)
	// return TestRunStatus(trData.RunStatus), err

	if rand.Intn(2) > 0 {
		return TestRunStatus(5), nil // mimic aborted
	}
	return TestRunStatus(2), nil
}
