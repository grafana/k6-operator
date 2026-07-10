package cloud

import (
	"go.k6.io/k6/v2/lib/types"
	"go.k6.io/k6/v2/metrics"
)

// InspectOutput is the parsed output from `k6 inspect --execution-requirements`.
type InspectOutput struct {
	External struct { // legacy way of defining the options.cloud
		Loadimpact struct {
			Name      string `json:"name"`
			ProjectID int64  `json:"projectID"`
		} `json:"loadimpact"`
	} `json:"ext"`
	Cloud struct { // actual way of defining the options.cloud
		Name      string `json:"name"`
		ProjectID int64  `json:"projectID"`
	} `json:"cloud"`
	TotalDuration types.NullDuration             `json:"totalDuration"`
	MaxVUs        uint64                         `json:"maxVUs"`
	Thresholds    map[string]*metrics.Thresholds `json:"thresholds,omitempty"`
}

// ProjectID returns the project ID from the inspect output.
func (io *InspectOutput) ProjectID() int64 {
	if io.Cloud.ProjectID > 0 {
		return io.Cloud.ProjectID
	}

	return io.External.Loadimpact.ProjectID
}

// TestName returns the test name from the inspect output.
func (io *InspectOutput) TestName() string {
	if len(io.Cloud.Name) > 0 {
		return io.Cloud.Name
	}

	return io.External.Loadimpact.Name
}

// SetTestName sets the name in the inspect output.
func (io *InspectOutput) SetTestName(name string) {
	io.Cloud.Name = name
}
