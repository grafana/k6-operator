package cloud

import (
	"fmt"
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
