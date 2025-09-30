package jobs

import (
	"testing"

	deep "github.com/go-test/deep"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/resources/containers"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewStarterJob(t *testing.T) {

	automountServiceAccountToken := true
	zero := int32(0)

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-starter",
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
			BackoffLimit: &zero,
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
						containers.NewStartContainer([]string{"testing"}, "image", corev1.PullNever, []string{"sh", "-c"},
							[]corev1.EnvVar{}, corev1.SecurityContext{}, corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(50, resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(2097152, resource.BinarySI),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(209715200, resource.BinarySI),
								},
							},
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

	job := NewStarterJob(k6, []string{"testing"})
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Error(diff)
	}

}

func TestNewStarterJobIstio(t *testing.T) {

	automountServiceAccountToken := true
	zero := int32(0)

	expectedOutcome := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-starter",
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
			BackoffLimit: &zero,
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
						containers.NewStartContainer([]string{"testing"}, "image", "", []string{"scuttle", "sh", "-c"}, []corev1.EnvVar{
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
							corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(50, resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(2097152, resource.BinarySI),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
									corev1.ResourceMemory: *resource.NewQuantity(209715200, resource.BinarySI),
								},
							},
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

	job := NewStarterJob(k6, []string{"testing"})
	if diff := deep.Equal(job, expectedOutcome); diff != nil {
		t.Error(diff)
	}

}

func TestNewStarterJobCustomResources(t *testing.T) {
	// Test case 1: Default resources should be applied when no custom resources are specified
	k6Default := &v1alpha1.TestRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.TestRunSpec{
			Starter: v1alpha1.Pod{
				Image: "image",
			},
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{Name: "test", File: "test.js"},
			},
		},
	}

	jobDefault := NewStarterJob(k6Default, []string{"testing"})
	gotDefaultRes := jobDefault.Spec.Template.Spec.Containers[0].Resources

	expectedDefaultRes := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(50, resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(2097152, resource.BinarySI),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
			corev1.ResourceMemory: *resource.NewQuantity(209715200, resource.BinarySI),
		},
	}

	if diff := deep.Equal(gotDefaultRes, expectedDefaultRes); diff != nil {
		t.Errorf("default resources not applied: %v", diff)
	}

	// Test case 2: Custom resources should override defaults
	reqs := corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(100, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(64*1024*1024, resource.BinarySI), // 64 Mi
	}
	lims := corev1.ResourceList{
		corev1.ResourceCPU:    *resource.NewMilliQuantity(250, resource.DecimalSI),
		corev1.ResourceMemory: *resource.NewQuantity(160*1024*1024, resource.BinarySI), // 160 Mi
	}
	customRes := corev1.ResourceRequirements{Requests: reqs, Limits: lims}

	k6Custom := &v1alpha1.TestRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
		Spec: v1alpha1.TestRunSpec{
			Starter: v1alpha1.Pod{
				Image:     "image",
				Resources: customRes,
			},
			Script: v1alpha1.K6Script{
				ConfigMap: v1alpha1.K6Configmap{Name: "test", File: "test.js"},
			},
		},
	}

	jobCustom := NewStarterJob(k6Custom, []string{"testing"})
	gotCustomRes := jobCustom.Spec.Template.Spec.Containers[0].Resources

	if diff := deep.Equal(gotCustomRes, customRes); diff != nil {
		t.Errorf("custom resources not applied: %v", diff)
	}
}
