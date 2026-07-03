package cloud

import (
	"fmt"
	"sort"
	"strings"

	"go.k6.io/k6/v2/cloudapi"
	corev1 "k8s.io/api/core/v1"
)

// GCk6 can set only a limited number of env vars to k6 process:
// these are known and whitelisted with this "const" map.
var reservedGCk6EnvVars = map[string]struct{}{
	// Future candidates:
	// "K6_CLOUD_TOKEN": struct{}{},
	// K6_LOG_OUTPUT, K6_TRACES_OUTPUT, K6_BROWSER_ENABLED_MSG, K6_CLOUD_TRACES_ENABLED, K6_BROWSER_SCREENSHOTS_OUTPUT
}

const (
	// Reserved vars set for PLZ tests, as described here:
	// https://grafana.com/docs/grafana-cloud/testing/k6/author-run/cloud-scripting-extras/cloud-execution-context-variables/
	// These are not passed from GCk6, but set by k6-operator directly.
	lzCloudExecVar    = "K6_CLOUDRUN_LOAD_ZONE"
	distrCloudExecVar = "K6_CLOUDRUN_DISTRIBUTION"
	trIDCloudExecVar  = "K6_CLOUDRUN_TEST_RUN_ID"
	// IIDCloudExecVar is exported as it must be set in external package, as part of TestRun CRD flow
	IIDCloudExecVar = "K6_CLOUDRUN_INSTANCE_ID"

	// These are not in GCk6 reserved only because current API sends them in
	// separate fields. It'd be nice to refactor the API to reduce complexity.
	secretSourceEnvVar      = "K6_SECRET_SOURCE"
	secretSourceURLTemplate = "K6_SECRET_SOURCE_URL_URL_TEMPLATE"
	secretSourceURLRespPath = "K6_SECRET_SOURCE_URL_RESPONSE_PATH"
	secretSourceURLAuthKey  = "K6_SECRET_SOURCE_URL_HEADER_AUTHORIZATION"
)

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

	// Pre-processed k6 arguments and env vars, populated by Preprocess().
	TagArgs string `json:"-"`
	EnvArgs string `json:"-"`
	// RunnerEnvVars holds k6 option env vars (user agent, blacklists) and
	// K6_CLOUD_OPERATOR_ENV_* helper env vars referenced from EnvArgs.
	RunnerEnvVars []corev1.EnvVar `json:"-"`
}

func (trd *TestRunData) TestRunID() string {
	return fmt.Sprintf("%d", trd.TestRunId)
}

// Preprocess adds specific for GCk6 tags and env vars to data,
// and produces CLI argument strings and env vars for the runners.
// Returns error if distribution is not a single load zone.
func (trd *TestRunData) Preprocess() error {
	if len(trd.LZDistribution) != 1 {
		return fmt.Errorf("only tests with one load zone are supported, provided: %+v", trd.LZDistribution)
	}

	trd.TagArgs = "--tag load_zone=" + trd.LZName()

	// Handle k6 CLI options that need to be passed as env vars to the runners.

	if len(trd.UserAgent) > 0 {
		trd.RunnerEnvVars = append(trd.RunnerEnvVars, corev1.EnvVar{
			Name: "K6_USER_AGENT", Value: trd.UserAgent,
		})
	}
	if len(trd.BlacklistIPs) > 0 {
		trd.RunnerEnvVars = append(trd.RunnerEnvVars, corev1.EnvVar{
			Name: "K6_BLACKLIST_IPS", Value: strings.Join(trd.BlacklistIPs, ","),
		})
	}
	if len(trd.BlockedHostnames) > 0 {
		trd.RunnerEnvVars = append(trd.RunnerEnvVars, corev1.EnvVar{
			Name: "K6_BLOCK_HOSTNAMES", Value: strings.Join(trd.BlockedHostnames, ","),
		})
	}

	// Handle env vars that configure k6 CLI as received from the Cloud.
	// Note: GCk6 reserved env vars are handled separately ATM.

	trd.RunnerEnvVars = append(trd.RunnerEnvVars, AggregationEnvVars(&trd.RuntimeConfig)...)
	trd.RunnerEnvVars = append(trd.RunnerEnvVars, trd.secretsEnvVars()...)

	trd.RunnerEnvVars = append(trd.RunnerEnvVars, trd.reservedEnvVars()...)
	trd.RunnerEnvVars = append(trd.RunnerEnvVars, corev1.EnvVar{
		Name:  "K6_CLOUD_HOST",
		Value: K6CloudHost(),
	})

	if trd.Environment == nil {
		trd.Environment = make(map[string]string)
	}

	// Handle env vars that describe cloud tests and should be reachable from k6.

	// The potential overwrite here is deliberate: these keys are reserved by
	// GCk6, described in docs and considered higher priority in PLZ tests
	// for the sake of consistency between public & private cloud tests.
	trd.Environment[lzCloudExecVar] = trd.LZName()
	trd.Environment[distrCloudExecVar] = trd.LZLabel()
	trd.Environment[trIDCloudExecVar] = trd.TestRunID()
	delete(trd.Environment, IIDCloudExecVar) // populated later

	// Handle env vars set by user in GCk6 UI.

	keys := make([]string, 0, len(trd.Environment))
	for k := range trd.Environment {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// These env vars are set as env vars with the name `K6_CLOUD_OPERATOR_ENV_0..N`
	// in the pods and then passed to k6 command with `-e KEY=$(K6_CLOUD_OPERATOR_ENV_0..N)`.
	// Why: this keeps values with spaces, quotes etc. intact, as we don't have a guarantee
	// that the values are sanitized here.
	// Also, see https://github.com/grafana/k6/issues/2730 for additional context.
	envArgs := make([]string, 0, len(keys))
	for i, k := range keys {
		helper := fmt.Sprintf("K6_CLOUD_OPERATOR_ENV_%d", i)
		envArgs = append(envArgs, fmt.Sprintf("-e %s=$(%s)", k, helper))
		trd.RunnerEnvVars = append(trd.RunnerEnvVars, corev1.EnvVar{
			Name: helper, Value: trd.Environment[k],
		})
	}
	trd.EnvArgs = strings.Join(envArgs, " ")

	return nil
}

type LZConfig struct {
	RunnerImage   string `json:"load_runner_image,omitempty"`
	InstanceCount int    `json:"instance_count,omitempty"`
	ArchiveURL    string `json:"k6_archive_temp_public_url,omitempty"`
	CLIArgs       `json:"cli_flags,omitempty"`
	// Environment holds values passed by user via:
	// 1. cloud environment variables of GCk6 -> Settings
	// 2. `-e` CLI option of k6 when 1) is non-empty
	// (otherwise, these are passed via the archive)
	//
	// They are passed to k6 runners via `-e`.
	Environment map[string]string `json:"environment,omitempty"`
	// GCk6EnvVars holds key-value pairs generated by GCk6 and
	// meant to configure k6 process with reserved env vars.
	GCk6EnvVars map[string]string `json:"gck6_env_vars,omitempty"`
}

type CLIArgs struct {
	BlacklistIPs         []string `json:"blacklist_ips,omitempty"`
	BlockedHostnames     []string `json:"blocked_hostnames,omitempty"`
	IncludeSystemEnvVars bool     `json:"include_system_env_vars,omitempty"`
	UserAgent            string   `json:"user_agent,omitempty"`
	// not used ATM
	// Tags                 map[string]string `json:"tags,omitempty"`
}

type LZDistribution map[string]Distribution

type Distribution struct {
	LoadZone string `json:"loadZone"`
	Percent  int    `json:"percent"`
}

// reservedEnvVars makes up the corev1 struct with the reserved GCk6 env vars.
func (lz *LZConfig) reservedEnvVars() []corev1.EnvVar {
	ev := make([]corev1.EnvVar, 0, len(lz.GCk6EnvVars))
	for k, v := range lz.GCk6EnvVars {
		if _, ok := reservedGCk6EnvVars[k]; ok {
			ev = append(ev, corev1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}

	// to have deterministic order in the array
	sort.Slice(ev, func(i, j int) bool {
		return ev[i].Name < ev[j].Name
	})
	return ev
}

// secretsEnvVars returns the env vars required by the k6 URL secret source.
// Returns nil when no secrets configuration is present.
func (trd *TestRunData) secretsEnvVars() []corev1.EnvVar {
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
