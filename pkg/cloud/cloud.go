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
}

func CreateTestRun(opts InspectOutput, token string, log logr.Logger) (string, error) {
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

	client = cloudapi.NewClient(logger, token, cloudConfig.Host.String, consts.Version, time.Duration(time.Minute))
	resp, err := createTestRun(client, &TestRun{
		Name:       opts.External.Loadimpact.Name,
		ProjectID:  cloudConfig.ProjectID.Int64,
		VUsMax:     int64(opts.MaxVUs),
		Thresholds: opts.Thresholds,
		// This is heuristic increase of duration to take into account that it takes time to start the pods.
		// By current observations, it shouldn't matter that much since we're sending a finish call in the end,
		// but it would be good to come up with another solution.
		Duration:          int64(opts.TotalDuration.TimeDuration().Seconds()) * 2,
		ProcessThresholds: true,
	})

	if err != nil {
		return "", err
	}

	return resp.ReferenceID, nil
}

// We cannot use cloudapi.TestRun struct and cloudapi.Client.CreateTestRun call because they're not aware of
// process_thresholds argument; so let's use custom struct and function instead
func createTestRun(client *cloudapi.Client, testRun *TestRun) (*cloudapi.CreateTestRunResponse, error) {
	url := "https://ingest.k6.io/v1/tests"
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
