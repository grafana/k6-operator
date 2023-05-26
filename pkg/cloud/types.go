package cloud

import (
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
	TestRunId string `json:"testRunId"`
	// ArchiveURL
	Instances int `json:"instances"`
}
