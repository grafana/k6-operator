package jobs

import (
	"testing"

	deep "github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/resources/containers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewStopperJob(t *testing.T) {
	automountServiceAccountToken := true

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stopper",
			Namespace: "test",
			Labels: map[string]string{
				"app":    "k6",
				"k6_cr":  "test",
				"label1": "awesome",
			},
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":    "k6",
						"k6_cr":  "test",
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &automountServiceAccountToken,
					ServiceAccountName:           "default",
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					TopologySpreadConstraints:    nil,
					RestartPolicy:                corev1.RestartPolicyNever,
					SecurityContext:              &corev1.PodSecurityContext{},
					Containers: []corev1.Container{
						containers.NewStopContainer([]string{"testing"}, "image", corev1.PullNever, []string{"sh", "-c"},
							[]corev1.EnvVar{}, corev1.SecurityContext{}, corev1.ResourceRequirements{}),
					},
				},
			},
		},
	}

	k6 := &v1alpha1.TestRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.TestRunSpec{
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Starter: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Image:           "image",
				ImagePullPolicy: corev1.PullNever,
			},
		},
	}

	job := NewStopJob(k6, []string{"testing"})
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Error(diff)
	}

}

func TestNewStopJobIstio(t *testing.T) {
	automountServiceAccountToken := true

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-stopper",
			Namespace: "test",
			Labels: map[string]string{
				"app":    "k6",
				"k6_cr":  "test",
				"label1": "awesome",
			},
			Annotations: map[string]string{
				"awesomeAnnotation": "dope",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":    "k6",
						"k6_cr":  "test",
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: &automountServiceAccountToken,
					ServiceAccountName:           "default",
					Affinity:                     nil,
					NodeSelector:                 nil,
					Tolerations:                  nil,
					TopologySpreadConstraints:    nil,
					RestartPolicy:                corev1.RestartPolicyNever,
					SecurityContext:              &corev1.PodSecurityContext{},
					Containers: []corev1.Container{
						containers.NewStopContainer([]string{"testing"}, "image", "", []string{"scuttle", "sh", "-c"}, []corev1.EnvVar{
							{
								Name:  "ENVOY_ADMIN_API",
								Value: "http://127.0.0.1:15000",
							},
							{
								Name:  "ISTIO_QUIT_API",
								Value: "http://127.0.0.1:15020",
							},
							{
								Name:  "WAIT_FOR_ENVOY_TIMEOUT",
								Value: "15",
							}},
							corev1.SecurityContext{},
							corev1.ResourceRequirements{},
						),
					},
				},
			},
		},
	}

	k6 := &v1alpha1.TestRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.TestRunSpec{
			Scuttle: v1alpha1.K6Scuttle{
				Enabled: "true",
			},
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{
					Name: "test",
					File: "test.js",
				},
			},
			Starter: v1alpha1.Pod{
				Metadata: v1alpha1.PodMetadata{
					Labels: map[string]string{
						"label1": "awesome",
					},
					Annotations: map[string]string{
						"awesomeAnnotation": "dope",
					},
				},
				Image: "image",
			},
		},
	}

	job := NewStopJob(k6, []string{"testing"})
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Error(diff)
	}
}
