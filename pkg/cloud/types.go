package cloud

import (
	"fmt"

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

// testRunList holds the output from /v4/plz-test-runs call
type testRunList struct {
	List []struct {
		ID uint64 `json:"id"`
	} `json:"object"`
}

// TestRunData holds the output from /loadtests/v4/test_runs(%s)
type TestRunData struct {
	TestRunId int `json:"id"`
	Instances int `json:"instances"`
	LZConfig  `json:"k8s_load_zones_config"`
	RunStatus cloudapi.RunStatus `json:"run_status"`
}

type LZConfig struct {
	RunnerImage   string `json:"load_runner_image"`
	InstanceCount int    `json:"instance_count"`
	ArchiveURL    string `json:"k6_archive_temp_public_url"`
}

func (trd *TestRunData) TestRunID() string {
	return fmt.Sprintf("%d", trd.TestRunId)
}

type TestRunStatus cloudapi.RunStatus

func (trs TestRunStatus) Aborted() bool {
	return cloudapi.RunStatus(trs) >= cloudapi.RunStatusAbortedUser
}

// func (trs TestRunStatus) String() string {
// 	TODO: for a pretty output about test run status in the logs?
// }

// PLZRegistrationData holds info that needs to be sent to /v1/load-zones
type PLZRegistrationData struct {
	LoadZoneID string       `json:"k6_load_zone_id"`
	Resources  PLZResources `json:"pod_tiers"`
}

type PLZResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}
