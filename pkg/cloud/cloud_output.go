package cloud

import (
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/cloudapi"
	"go.k6.io/k6/lib/consts"
	null "gopkg.in/guregu/null.v3"
)

// TODO: refactor this!
var client *cloudapi.Client

type TestRun struct {
	Name              string              `json:"name"`
	ProjectID         int64               `json:"project_id,omitempty"`
	VUsMax            int64               `json:"vus"`
	Thresholds        map[string][]string `json:"thresholds"`
	Duration          int64               `json:"duration"`
	ProcessThresholds bool                `json:"process_thresholds"`
	Instances         int32               `json:"instances"`
}

func NewClient(log logr.Logger, token, host string) *cloudapi.Client {
	logger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}

	cloudConfig := cloudapi.NewConfig()

	if len(host) == 0 {
		host = cloudConfig.Host.String
	}

	return cloudapi.NewClient(logger, token, host, consts.Version, time.Duration(time.Minute))
}

func CreateTestRun(opts InspectOutput, instances int32, host, token string, log logr.Logger) (*cloudapi.CreateTestRunResponse, error) {
	if client == nil {
		client = NewClient(log, token, host)
	}

	cloudConfig := cloudapi.NewConfig()

	if opts.ProjectID() > 0 {
		cloudConfig.ProjectID = null.NewInt(opts.ProjectID(), true)
	}

	thresholds := make(map[string][]string, len(opts.Thresholds))
	for name, t := range opts.Thresholds {
		for _, threshold := range t.Thresholds {
			thresholds[name] = append(thresholds[name], threshold.Source)
		}
	}

	if len(host) == 0 {
		host = cloudConfig.Host.String
	}

	if client == nil {
		client = NewClient(log, token, host)
	}

	tr := TestRun{
		Name:              opts.TestName(),
		ProjectID:         cloudConfig.ProjectID.Int64,
		VUsMax:            int64(opts.MaxVUs),
		Thresholds:        thresholds,
		Duration:          int64(opts.TotalDuration.TimeDuration().Seconds()),
		ProcessThresholds: true,
		Instances:         instances,
	}
	return createTestRun(client, host, &tr)
}

// We cannot use cloudapi.TestRun struct and cloudapi.Client.CreateTestRun call because they're not aware of
// process_thresholds argument; so let's use custom struct and function instead
func createTestRun(client *cloudapi.Client, host string, testRun *TestRun) (*cloudapi.CreateTestRunResponse, error) {
	url := host + "/v1/tests"
	req, err := client.NewRequest("POST", url, testRun)
	if err != nil {
		return nil, err
	}

	ctrr := cloudapi.CreateTestRunResponse{}
	err = client.Do(req, &ctrr)
	if err != nil {
		return nil, err
	}

	if ctrr.ReferenceID == "" {
		return nil, fmt.Errorf("failed to get a reference ID")
	}

	return &ctrr, nil
}

func FinishTestRun(c *cloudapi.Client, refID string) error {
	if c != nil {
		return c.TestFinished(refID, cloudapi.ThresholdResult(
			map[string]map[string]bool{},
		), false, cloudapi.RunStatusFinished)
	}

	return client.TestFinished(refID, cloudapi.ThresholdResult(
		map[string]map[string]bool{},
	), false, cloudapi.RunStatusFinished)
}
