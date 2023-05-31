package cloud

import (
	"go.k6.io/k6/cloudapi"
	"go.k6.io/k6/lib/types"
	"go.k6.io/k6/metrics"
)

// InspectOutput is the parsed output from `k6 inspect --execution-requirements`.
type InspectOutput struct {
	External struct {
		Loadimpact struct {
			Name      string `json:"name"`
			ProjectID int64  `json:"projectID"`
		} `json:"loadimpact"`
	} `json:"ext"`
	TotalDuration types.NullDuration             `json:"totalDuration"`
	MaxVUs        uint64                         `json:"maxVUs"`
	Thresholds    map[string]*metrics.Thresholds `json:"thresholds,omitempty"`
}

// testRunList holds the output from /get-tests call
type testRunList struct {
	List []string `json:"list"`
}

// TestRunData holds the output from /get-test-data call
type TestRunData struct {
	TestRunId string `json:"id"`
	Instances int    `json:"instances"`
	LZConfig  `json:"k8s_load_zones_config"`
	RunStatus cloudapi.RunStatus `json:"run_status"`
}

type LZConfig struct {
	RunnerImage   string `json:"load_runner_image"`
	InstanceCount int    `json:"instance_count"`
	ArchiveURL    string `json:"k6_archive_temp_public_url"`
}

type TestRunStatus cloudapi.RunStatus

func (trs TestRunStatus) Aborted() bool {
	return cloudapi.RunStatus(trs) >= cloudapi.RunStatusAbortedUser
}

// func (trs TestRunStatus) String() string {
// 	TODO
// }
