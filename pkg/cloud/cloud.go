package cloud

import (
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/cloudapi"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/consts"
	"go.k6.io/k6/lib/types"
)

var client *cloudapi.Client

type InspectOutput struct {
	External struct {
		Loadimpact struct {
			Name      string `json:"name"`
			ProjectID int64  `json:"projectID"`
		} `json:"loadimpact"`
	} `json:"ext"`
	TotalDuration types.NullDuration `json:"totalDuration"`
	MaxVUs        uint64             `json:"maxVUs"`
}

func CreateTestRun(opts InspectOutput, token string, log logr.Logger) (string, error) {
	if len(opts.External.Loadimpact.Name) < 1 {
		opts.External.Loadimpact.Name = "k6-operator-test"
	}
	projectId := opts.External.Loadimpact.ProjectID

	cloudConfig := cloudapi.NewConfig()

	logger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}

	client = cloudapi.NewClient(logger, token, cloudConfig.Host.String, consts.Version, time.Duration(time.Minute))
	resp, err := client.CreateTestRun(&cloudapi.TestRun{
		Name:       opts.External.Loadimpact.Name,
		ProjectID:  projectId,
		VUsMax:     int64(opts.MaxVUs),
		Thresholds: map[string][]string{},
		// This is heuristic increase of duration to take into account that it takes time to start the pods.
		// By current observations, it shouldn't matter that much since we're sending a finish call in the end,
		// but it would be good to come up with another solution.
		Duration: int64(opts.TotalDuration.TimeDuration().Seconds()) * 2,
	})

	if err != nil {
		return "", err
	}

	return resp.ReferenceID, nil
}

func FinishTestRun(refID string) error {
	return client.TestFinished(refID, cloudapi.ThresholdResult(
		map[string]map[string]bool{},
	), false, lib.RunStatusFinished)
}
