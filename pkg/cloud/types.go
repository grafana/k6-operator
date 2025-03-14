package cloud

import (
	"fmt"
	"sort"

	"go.k6.io/k6/cloudapi"
	"go.k6.io/k6/lib/types"
	"go.k6.io/k6/metrics"
	corev1 "k8s.io/api/core/v1"
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

// testRunList holds the output from /v4/plz-test-runs call
type testRunList struct {
	List []struct {
		ID uint64 `json:"id"`
	} `json:"object"`
}

// TestRunData holds the output from /loadtests/v4/test_runs(%s)
type TestRunData struct {
	TestRunId     int `json:"id"`
	Instances     int `json:"instances"`
	LZConfig      `json:"k8s_load_zones_config"`
	RunStatus     cloudapi.RunStatus `json:"run_status"`
	RuntimeConfig cloudapi.Config    `json:"k6_runtime_config"`
}

type LZConfig struct {
	RunnerImage   string            `json:"load_runner_image,omitempty"`
	InstanceCount int               `json:"instance_count,omitempty"`
	ArchiveURL    string            `json:"k6_archive_temp_public_url,omitempty"`
	Environment   map[string]string `json:"environment,omitempty"`
}

func (trd *TestRunData) TestRunID() string {
	return fmt.Sprintf("%d", trd.TestRunId)
}

func (lz *LZConfig) EnvVars() []corev1.EnvVar {
	ev := make([]corev1.EnvVar, len(lz.Environment))
	i := 0
	for k, v := range lz.Environment {
		ev[i] = corev1.EnvVar{
			Name:  k,
			Value: v,
		}
		i++
	}
	// to have deterministic order in the array
	sort.Slice(ev, func(i, j int) bool {
		return ev[i].Name < ev[j].Name
	})

	return ev
}

type TestRunStatus cloudapi.RunStatus

func (trs TestRunStatus) Aborted() bool {
	// Abort: on timeout, on any kind of abort and on archived.
	// Ref.: https://github.com/grafana/k6/blob/master/cloudapi/run_status.go
	return cloudapi.RunStatus(trs) >= cloudapi.RunStatusTimedOut
}

// func (trs TestRunStatus) String() string {
// 	TODO: for a pretty output about test run status in the logs?
// }

// PLZRegistrationData holds info that needs to be sent to /v1/load-zones
type PLZRegistrationData struct {
	// defined by user as `name`
	LoadZoneID string       `json:"k6_load_zone_id"`
	Resources  PLZResources `json:"pod_tiers"`

	LZConfig `json:"config"`

	// Unique identifier of PLZ, generated by k6-operator
	// during PLZ registration. It's purpose is to distinguish
	// between PLZs with accidentally duplicate names.
	UID string `json:"provider_id"`
}

type PLZResources struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
}

type Events []*EventPayload

type EventPayload struct {
	EventType `json:"event_type"`
	Event     `json:"event"`
}

type Event struct {
	Origin    `json:"origin,omitempty"`
	ErrorCode `json:"error_code,omitempty"`

	// reason is used for abort events,
	// while details are for any non-abort event
	Reason       string `json:"reason,omitempty"`
	Detail       string `json:"error_detail,omitempty"`
	PublicDetail string `json:"error_detail_public,omitempty"`
}

type EventType string

var (
	abortEvent = EventType("TestRunAbortEvent")
	errorEvent = EventType("TestRunErrorEvent")
)

type ErrorCode uint

var (
	SetupError      = ErrorCode(8030)
	TeardownError   = ErrorCode(8031)
	OOMError        = ErrorCode(8032)
	PanicError      = ErrorCode(8033)
	UnknownError    = ErrorCode(8034)
	ScriptException = ErrorCode(8035)

	K6OperatorStartError  = ErrorCode(8050)
	K6OperatorAbortError  = ErrorCode(8051)
	K6OperatorRunnerError = ErrorCode(8052)
)

type Origin string

var (
	OriginUser = Origin("user")
	OriginK6   = Origin("k6")
)

// WithDetail sets detail only for the 1st event.
// If it's abort, WithDetail sets reason field.
func (e *Events) WithDetail(s string) *Events {
	if len(*e) == 0 {
		return e
	}

	if (*e)[0].EventType == abortEvent {
		(*e)[0].Reason = s
	} else {
		(*e)[0].Detail = s
		(*e)[0].PublicDetail = s
	}
	return e
}

// WithAbort adds abortEvent to errorEvent if it already exists.
func (e *Events) WithAbort() *Events {
	if len(*e) == 0 {
		return e
	}

	if (*e)[0].EventType == errorEvent {
		*e = append(*e, AbortEvent(OriginUser))
	}
	return e
}

func AbortEvent(o Origin) *EventPayload {
	e := &EventPayload{
		EventType: abortEvent,
		Event: Event{
			Origin: o,
		},
	}
	return e
}

func ErrorEvent(ec ErrorCode) *Events {
	e := Events([]*EventPayload{{
		EventType: errorEvent,
		Event: Event{
			ErrorCode: ec,
		},
	}})
	return &e
}
