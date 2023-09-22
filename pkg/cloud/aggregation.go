package cloud

import (
	"fmt"
	"strings"

	"go.k6.io/k6/cloudapi"
	corev1 "k8s.io/api/core/v1"
)

var aggregationVarNames = map[int][]string{
	1: []string{
		// cloud output v1: to be removed in the future
		"K6_CLOUD_AGGREGATION_MIN_SAMPLES",
		"K6_CLOUD_AGGREGATION_PERIOD",
		"K6_CLOUD_AGGREGATION_WAIT_PERIOD",
		"K6_CLOUD_METRIC_PUSH_INTERVAL",
		"K6_CLOUD_MAX_METRIC_SAMPLES_PER_PACKAGE",
		"K6_CLOUD_MAX_METRIC_PUSH_CONCURRENCY",
	},
	2: []string{
		// cloud output v2
		"K6_CLOUD_API_VERSION",
		"K6_CLOUD_AGGREGATION_PERIOD",
		"K6_CLOUD_AGGREGATION_WAIT_PERIOD",
		"K6_CLOUD_METRIC_PUSH_INTERVAL",
		"K6_CLOUD_METRIC_PUSH_CONCURRENCY",
	},
}

func EncodeAggregationConfig(testRun *cloudapi.Config) string {
	return fmt.Sprintf("%d|%s|%s|%s|%d",
		2, // use v2 for all new test runs
		testRun.AggregationPeriod.String(),
		testRun.AggregationWaitPeriod.String(),
		testRun.MetricPushInterval.String(),
		testRun.MetricPushConcurrency.Int64)
}

func DecodeAggregationConfig(encoded string) ([]corev1.EnvVar, error) {
	values := strings.Split(encoded, "|")

	// in order not to break existing deployments,
	// let's support decoding of cloud output v1 for some time
	var (
		apiV1VarNames = len(aggregationVarNames[1])
		apiV2VarNames = len(aggregationVarNames[2])
	)

	if len(values) != apiV1VarNames && len(values) != apiV2VarNames {
		return nil, fmt.Errorf(
			"Aggregation vars got corrupted: there are %d values instead of %d or %d. Encoded value: `%s`.",
			len(values),
			apiV1VarNames, apiV2VarNames,
			encoded)
	}

	var varNames []string
	if len(values) == apiV1VarNames {
		varNames = aggregationVarNames[1]
	} else {
		varNames = aggregationVarNames[2]
	}

	vars := make([]corev1.EnvVar, len(values))
	for i := range values {
		vars[i] = corev1.EnvVar{
			Name:  varNames[i],
			Value: values[i],
		}
	}

	return vars, nil
}
