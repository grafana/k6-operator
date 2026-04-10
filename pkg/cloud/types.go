package cloud

import (
	"fmt"
	"maps"
	"sort"
	"strings"

	"go.k6.io/k6/cloudapi"
	"go.k6.io/k6/lib/types"
	"go.k6.io/k6/metrics"
	corev1 "k8s.io/api/core/v1"
)

// GCk6 can set only a limited number of env vars to k6 process:
// these are known and whitelisted with this "const" map.
var reservedGCk6EnvVars = map[string]struct{}{}

const (
	// Reserved vars set for PLZ tests, as described here:
	// https://grafana.com/docs/grafana-cloud/testing/k6/author-run/cloud-scripting-extras/cloud-execution-context-variables/
	// These are not passed from GCk6, but set by k6-operator directly.
	lzCloudExecVar    = "K6_CLOUDRUN_LOAD_ZONE"
	distrCloudExecVar = "K6_CLOUDRUN_DISTRIBUTION"
	trIDCloudExecVar  = "K6_CLOUDRUN_TEST_RUN_ID"
	// IIDCloudExecVar is exported as it must be set in external package, as part of TestRun CRD flow
	IIDCloudExecVar = "K6_CLOUDRUN_INSTANCE_ID"

	secretSourceEnvVar      = "K6_SECRET_SOURCE"
	secretSourceURLTemplate = "K6_SECRET_SOURCE_URL_URL_TEMPLATE"
	secretSourceURLRespPath = "K6_SECRET_SOURCE_URL_RESPONSE_PATH"
	secretSourceURLAuthKey  = "K6_SECRET_SOURCE_URL_HEADER_AUTHORIZATION"
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

// SecretsConfig holds the secrets configuration returned by k6 Cloud.
type SecretsConfig struct {
	Endpoint     string `json:"endpoint"`
	ResponsePath string `json:"response_path"`
}

// TestRunData holds the output from /loadtests/v4/test_runs(%s)
type TestRunData struct {
	TestRunId     int `json:"id"`
	Instances     int `json:"instances"`
	LZConfig      `json:"k8s_load_zones_config"`
	RunStatus     cloudapi.RunStatus `json:"run_status"`
	RuntimeConfig cloudapi.Config    `json:"k6_runtime_config"`
	// SecretsToken is a short-lived, test-run-scoped token for read-only access to secrets.
	SecretsToken  string         `json:"test_run_token,omitempty"`
	SecretsConfig *SecretsConfig `json:"secrets_config,omitempty"`
	// LZDistribution holds label -> distribution mapping relevant
	// for the given script and PLZ
	LZDistribution `json:"load_zone_distribution,omitempty"`

	// Pre-processed k6 arguments, populated by Preprocess().
	TagArgs      string `json:"-"`
	EnvArgs      string `json:"-"`
	UserAgentArg string `json:"-"`
}

func (trd *TestRunData) TestRunID() string {
	return fmt.Sprintf("%d", trd.TestRunId)
}

// Preprocess adds specific for GCk6 tags and env vars to data,
// and produces sorted CLI argument strings for tags and environment.
// Returns error if distribution is empty.
func (trd *TestRunData) Preprocess() error {
	if len(trd.LZDistribution) != 1 {
		return fmt.Errorf("only tests with one load zone are supported, provided: %+v", trd.LZDistribution)
	}

	if len(trd.UserAgent) > 0 {
		trd.UserAgentArg = fmt.Sprintf(`--user-agent="%s"`, trd.UserAgent)
	}

	if trd.Tags == nil {
		trd.Tags = make(map[string]string)
	}

	if trd.Environment == nil {
		trd.Environment = make(map[string]string)
	}

	// The potential overwrite here is deliberate: these keys are reserved by
	// GCk6, described in docs and considered higher priority in PLZ tests
	// for the sake of consistency between public & private cloud tests.

	trd.Tags["load_zone"] = trd.LZLabel()

	trd.Environment[lzCloudExecVar] = trd.LZName()
	trd.Environment[distrCloudExecVar] = trd.LZLabel()
	trd.Environment[trIDCloudExecVar] = trd.TestRunID()

	trd.TagArgs = sortedArgs("--tag", trd.Tags)
	trd.EnvArgs = sortedArgs("-e", trd.Environment)

	return nil
}

// sortedArgs builds a CLI argument string from a map, sorted by key.
func sortedArgs(flag string, m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, flag+" "+k+"="+m[k])
	}
	return strings.Join(parts, " ")
}

type LZConfig struct {
	RunnerImage   string `json:"load_runner_image,omitempty"`
	InstanceCount int    `json:"instance_count,omitempty"`
	ArchiveURL    string `json:"k6_archive_temp_public_url,omitempty"`
	CLIArgs       `json:"cli_flags,omitempty"`
	// Environment holds values passed by user via:
	// 1. `-e` CLI option of k6
	// 2. cloud environment variables of GCk6 -> Settings
	// They are passed to k6 runners via `-e`
	Environment map[string]string `json:"environment,omitempty"`
	// GCk6EnvVars holds key-value pairs generated by GCk6 and
	// meant to configure k6 process with reserved env vars.
	GCk6EnvVars map[string]string `json:"gck6_env_vars,omitempty"`
}

type CLIArgs struct {
	BlacklistIPs         []string          `json:"blacklist_ips,omitempty"`
	BlockedHostnames     []string          `json:"blocked_hostnames,omitempty"`
	IncludeSystemEnvVars bool              `json:"include_system_env_vars,omitempty"`
	Tags                 map[string]string `json:"tags,omitempty"`
	UserAgent            string            `json:"user_agent,omitempty"`
}

type LZDistribution map[string]Distribution

type Distribution struct {
	LoadZone string `json:"loadZone"`
	Percent  int    `json:"percent"`
}

// EnvVars makes up the corev1 struct from Go map.
func (lz *LZConfig) EnvVars() []corev1.EnvVar {
	whitelisted := maps.Collect(
		func(yield func(_, _ string) bool) {
			for k, v := range lz.GCk6EnvVars {
				if _, ok := reservedGCk6EnvVars[k]; ok {
					if !yield(k, v) {
						return
					}
				}
			}
		},
	)

	ev := make([]corev1.EnvVar, len(whitelisted))
	i := 0
	for k, v := range whitelisted {
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

// SecretsEnvVars returns the env vars required by the k6 URL secret source.
// Returns nil when no secrets configuration is present.
// TODO: make this private and move it to EnvVars() / Preprocess()
func (trd *TestRunData) SecretsEnvVars() []corev1.EnvVar {
	if trd.SecretsConfig == nil {
		return nil
	}
	ev := []corev1.EnvVar{
		{Name: secretSourceEnvVar, Value: "url"},
		{Name: secretSourceURLTemplate, Value: trd.SecretsConfig.Endpoint},
		{Name: secretSourceURLRespPath, Value: trd.SecretsConfig.ResponsePath},
	}
	if trd.SecretsToken != "" {
		ev = append(ev, corev1.EnvVar{
			Name:  secretSourceURLAuthKey,
			Value: "Bearer " + trd.SecretsToken,
		})
	}
	return ev
}

// LZLabel assumes there is only one LZ.
func (lzd *LZDistribution) LZLabel() string {
	for k := range *lzd {
		return k
	}
	return "unknown_lz_label"
}

// LZName assumes there is only one LZ.
func (lzd *LZDistribution) LZName() string {
	for _, v := range *lzd {
		return v.LoadZone
	}
	return "unknown_lz_name"
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
