package cloud

import (
	"testing"

	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stretchr/testify/assert"
)

// 1 CPU in Kubernetes = 1 AWS vCPU = 1 GCP Core = 1 Azure vCore
// Docs: https://kubernetes.io/docs/tasks/configure-pod-container/assign-cpu-resource/#cpu-units

func TestConversion(t *testing.T) {
	testCases := []struct {
		k8sResource string
		expected    float64
	}{
		// CPU
		{
			"512m",
			0.512,
		},
		{
			"1000m",
			1,
		},
		{
			"1",
			1,
		},
		{
			"100",
			100,
		},
		// Memory
		{
			"104857600",
			104857600,
		},
		{
			"100M",
			100000000,
		},
		{
			"100Mi",
			104857600,
		},
		{
			"150Mi",
			157286400,
		},
		{
			"1050Mi",
			1101004800,
		},
		{
			"4000M",
			4000000000,
		},
		{
			"4000Mi",
			4194304000,
		},
		{
			"4Gi",
			4294967296,
		},
		{
			"10000Mi",
			10485760000,
		},
		{
			"16G",
			16000000000,
		},
		{
			"32G",
			32000000000,
		},
		{
			"64G",
			64000000000,
		},
	}

	for _, testCase := range testCases {
		q := resource.MustParse(testCase.k8sResource)
		got := q.AsApproximateFloat64()
		assert.Equal(t, testCase.expected, got, "testCase", testCase)
	}
}
