package cloud

import (
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/sirupsen/logrus"
	"go.k6.io/k6/cloudapi"
	"go.k6.io/k6/lib"
	"go.k6.io/k6/lib/consts"
)

var client *cloudapi.Client

func CreateTestRun(name string, token string, vus uint64, duration float64, log logr.Logger) (string, error) {
	cloudConfig := cloudapi.NewConfig()
	projectId := cloudConfig.ProjectID.ValueOrZero()

	logger := &logrus.Logger{
		Out:       os.Stdout,
		Formatter: new(logrus.TextFormatter),
		Hooks:     make(logrus.LevelHooks),
		Level:     logrus.InfoLevel,
	}

	client = cloudapi.NewClient(logger, token, cloudConfig.Host.String, consts.Version, time.Duration(time.Minute))
	resp, err := client.CreateTestRun(&cloudapi.TestRun{
		Name:       name,
		ProjectID:  projectId,
		VUsMax:     int64(vus),
		Thresholds: map[string][]string{},
		// This is heuristic increase of duration to take into account that it takes time to start the pods.
		// By current observations, it shouldn't matter that much since we're sending a finish call in the end,
		// but it would be good to come up with another solution.
		Duration: int64(duration) * 2,
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
