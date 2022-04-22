package cloud

import (
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/cloudapi"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/consts"
	"go.k6.io/k6/lib/types"
	"gopkg.in/guregu/null.v3"
)

var client *cloudapi.Client

type InspectOutput struct {
	External struct {
		Loadimpact struct {
			Name      string `json:"name"`
			ProjectID int64  `json:"projectID"`
		} `json:"loadimpact"`
	} `json:"ext"`
	TotalDuration types.NullDuration  `json:"totalDuration"`
	MaxVUs        uint64              `json:"maxVUs"`
	Thresholds    map[string][]string `json:"thresholds,omitempty"`
}

type TestRun struct {
	Name              string              `json:"name"`
	ProjectID         int64               `json:"project_id,omitempty"`
	VUsMax            int64               `json:"vus"`
	Thresholds        map[string][]string `json:"thresholds"`
	Duration          int64               `json:"duration"`
	ProcessThresholds bool                `json:"process_thresholds"`
	Instances         int32               `json:"instances"`
}

func CreateTestRun(opts InspectOutput, instances int32, host, token string, log logr.Logger) (string, error) {
	if len(opts.External.Loadimpact.Name) < 1 {
		opts.External.Loadimpact.Name = "k6-operator-test"
	}

	cloudConfig := cloudapi.NewConfig()

	if opts.External.Loadimpact.ProjectID > 0 {
		cloudConfig.ProjectID = null.NewInt(opts.External.Loadimpact.ProjectID, true)
	}

	logger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}

	if opts.Thresholds == nil {
		opts.Thresholds = make(map[string][]string)
	}

	if len(host) == 0 {
		host = cloudConfig.Host.String
	}

	client = cloudapi.NewClient(logger, token, host, consts.Version, time.Duration(time.Minute))
	resp, err := createTestRun(client, host, &TestRun{
		Name:              opts.External.Loadimpact.Name,
		ProjectID:         cloudConfig.ProjectID.Int64,
		VUsMax:            int64(opts.MaxVUs),
		Thresholds:        opts.Thresholds,
		Duration:          int64(opts.TotalDuration.TimeDuration().Seconds()),
		ProcessThresholds: true,
		Instances:         instances,
	})

	if err != nil {
		return "", err
	}

	return resp.ReferenceID, nil
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

func FinishTestRun(refID string) error {
	return client.TestFinished(refID, cloudapi.ThresholdResult(
		map[string]map[string]bool{},
	), false, lib.RunStatusFinished)
}
