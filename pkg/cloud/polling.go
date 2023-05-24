package cloud

import (
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/cloudapi"
	"go.k6.io/k6/lib/consts"
)

type TestRunPoller struct {
	token    string
	host     string
	logger   logr.Logger
	interval time.Duration

	running bool
	Client  *cloudapi.Client
	done    chan bool
}

func NewTestRunPoller(host, token string, logger logr.Logger) *TestRunPoller {
	logrusLogger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}

	return &TestRunPoller{
		token:    token,
		host:     host,
		logger:   logger,
		interval: 10 * time.Second,

		running: false,
		Client:  cloudapi.NewClient(logrusLogger, token, host, consts.Version, time.Duration(time.Minute)),
		done:    make(chan bool),
	}
}

func (poller TestRunPoller) IsPolling() bool {
	return poller.running
}

func (poller *TestRunPoller) Start() chan string {
	testRunsCh := make(chan string)
	ticker := time.NewTicker(poller.interval)

	go func() {
		for {
			select {
			case <-poller.done:
				ticker.Stop()
				close(testRunsCh)
				return

			case <-ticker.C:
				list, err := poller.getTestRuns()
				if err != nil {
					poller.logger.Error(err, "Failed to get test runs from k6 Cloud.")

				} else {
					poller.logger.Info(fmt.Sprintf("Retrieved test runs: %+v", list))

					for _, testRunId := range list {
						testRunsCh <- testRunId
					}
				}
			}
		}
	}()

	poller.running = true
	return testRunsCh
}

func (poller *TestRunPoller) Stop() {
	if poller.IsPolling() {
		poller.done <- true

		poller.running = false
	}
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
