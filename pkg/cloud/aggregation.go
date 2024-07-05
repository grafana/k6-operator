package cloud

import (
	"fmt"
	"strings"

	"go.k6.io/k6/cloudapi"
	corev1 "k8s.io/api/core/v1"
)

var aggregationVarNames = []string{
	// cloud output v2
	"K6_CLOUD_API_VERSION",
	"K6_CLOUD_AGGREGATION_PERIOD",
	"K6_CLOUD_AGGREGATION_WAIT_PERIOD",
	"K6_CLOUD_METRIC_PUSH_INTERVAL",
	"K6_CLOUD_METRIC_PUSH_CONCURRENCY",
}

func EncodeAggregationConfig(testRun *cloudapi.Config) string {
	return fmt.Sprintf("%d|%s|%s|%s|%d",
		2, // version of protocol
		testRun.AggregationPeriod.String(),
		testRun.AggregationWaitPeriod.String(),
		testRun.MetricPushInterval.String(),
		testRun.MetricPushConcurrency.Int64)
}

func DecodeAggregationConfig(encoded string) ([]corev1.EnvVar, error) {
	values := strings.Split(encoded, "|")

	if len(values) != len(aggregationVarNames) {
		return nil, fmt.Errorf(
			"Aggregation vars got corrupted: there are %d values instead of %d. Encoded value: `%s`.",
			len(values),
			len(aggregationVarNames),
			encoded)
	}

	vars := make([]corev1.EnvVar, len(values))
	for i := range values {
		vars[i] = corev1.EnvVar{
			Name:  aggregationVarNames[i],
			Value: values[i],
		}
	}

	return vars, nil
}
