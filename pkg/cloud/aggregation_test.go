package cloud

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.k6.io/k6/cloudapi"
	"go.k6.io/k6/lib/types"
	"gopkg.in/guregu/null.v3"
	corev1 "k8s.io/api/core/v1"
)

func Test_EncodeAggregationConfig(t *testing.T) {
	expected := "2|5s|3s|10s|10"

	testRunResponse := &cloudapi.CreateTestRunResponse{
		ReferenceID: "test-run-id",
		ConfigOverride: &cloudapi.Config{
			AggregationPeriod:     types.NullDurationFrom(time.Second * 5),
			AggregationWaitPeriod: types.NullDurationFrom(time.Second * 3),
			MetricPushInterval:    types.NullDurationFrom(time.Second * 10),
			MetricPushConcurrency: null.IntFrom(10),
		},
	}

	encodedAggregation := EncodeAggregationConfig(testRunResponse.ConfigOverride)
	assert.Equal(t, expected, encodedAggregation)
}

func Test_DecodeAggregationConfig(t *testing.T) {
	var (
		// For now, we support both versions in decoding.
		v1Encoded = "50|3s|8s|6s|10000|10"
		v2Encoded = "2|5s|3s|10s|10"

		v1EnvVars = []corev1.EnvVar{
			{
				Name:  "K6_CLOUD_AGGREGATION_MIN_SAMPLES",
				Value: "50",
			},
			{
				Name:  "K6_CLOUD_AGGREGATION_PERIOD",
				Value: "3s",
			},
			{
				Name:  "K6_CLOUD_AGGREGATION_WAIT_PERIOD",
				Value: "8s",
			},
			{
				Name:  "K6_CLOUD_METRIC_PUSH_INTERVAL",
				Value: "6s",
			},
			{
				Name:  "K6_CLOUD_MAX_METRIC_SAMPLES_PER_PACKAGE",
				Value: "10000",
			},
			{
				Name:  "K6_CLOUD_MAX_METRIC_PUSH_CONCURRENCY",
				Value: "10",
			},
		}

		v2EnvVars = []corev1.EnvVar{
			{
				Name:  "K6_CLOUD_API_VERSION",
				Value: "2",
			},
			{
				Name:  "K6_CLOUD_AGGREGATION_PERIOD",
				Value: "5s",
			},
			{
				Name:  "K6_CLOUD_AGGREGATION_WAIT_PERIOD",
				Value: "3s",
			},
			{
				Name:  "K6_CLOUD_METRIC_PUSH_INTERVAL",
				Value: "10s",
			},
			{
				Name:  "K6_CLOUD_METRIC_PUSH_CONCURRENCY",
				Value: "10",
			},
		}
	)

	envVars, err := DecodeAggregationConfig(v1Encoded)
	assert.Equal(t, nil, err)

	for i, expectedEnvVar := range v1EnvVars {
		assert.Equal(t, expectedEnvVar, envVars[i])
	}

	envVars, err = DecodeAggregationConfig(v2Encoded)
	assert.Equal(t, nil, err)
	for i, expectedEnvVar := range v2EnvVars {
		assert.Equal(t, expectedEnvVar, envVars[i])
	}
}
