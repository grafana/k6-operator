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
	TestRunId     int `json:"id"`
	Instances     int `json:"instances"`
	LZConfig      `json:"k8s_load_zones_config"`
	RunStatus     cloudapi.RunStatus `json:"run_status"`
	RuntimeConfig cloudapi.Config    `json:"k6_runtime_config"`
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
