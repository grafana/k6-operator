package testrun

import (
	"fmt"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestName(testRunId string) string {
	return fmt.Sprintf("plz-test-%s", testRunId)
}

func NewPLZTestRun(plz *v1alpha1.PrivateLoadZone, trData cloud.TestRunData) *v1alpha1.K6 {
	return &v1alpha1.K6{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestName(trData.TestRunId),
			Namespace: plz.Namespace,
		},
		Spec: v1alpha1.K6Spec{
			Runner: v1alpha1.Pod{
				ServiceAccountName: plz.Spec.ServiceAccountName,
				NodeSelector:       plz.Spec.NodeSelector,
				Resources: corev1.ResourceRequirements{
					Limits: plz.Spec.Resources,
					// Requests will default to the Limits values.
				},
			},
			Starter: v1alpha1.Pod{
				ServiceAccountName: plz.Spec.ServiceAccountName,
				NodeSelector:       plz.Spec.NodeSelector,
			},
			Parallelism: int32(trData.Instances),
			Separate:    true,
			// Arguments: "--out cloud",
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "crocodile-stress-test-short",
					File: "test.js",
				},
			},
			Cleanup: v1alpha1.Cleanup("post"),

			TestRunID: trData.TestRunId,
			Token:     plz.Spec.Token,
		},
	}
}
