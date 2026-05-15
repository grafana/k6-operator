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

func defaultTestRunForStarter() *v1alpha1.TestRun {
	return &v1alpha1.TestRun{
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
}

func defaultExpectedJobForStarter() *batchv1.Job {

	automountServiceAccountToken := true
	zero := int32(0)

	return &batchv1.Job{
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
					SchedulerName:                "default-scheduler",
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
}
func Test_NewStarterJob(t *testing.T) {
	tests := []struct {
		name             string
		hostname         []string
		setupTestRun     func(*v1alpha1.TestRun)
		setupExpectedJob func(*batchv1.Job)
	}{
		{
			name:             "base",
			hostname:         []string{"testing"},
			setupTestRun:     func(k6 *v1alpha1.TestRun) {},
			setupExpectedJob: func(j *batchv1.Job) {},
		},
		{
			name:     "custom scheduler name",
			hostname: []string{"testing"},
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Starter.SchedulerName = "custom-scheduler"
			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.SchedulerName = "custom-scheduler"
			},
		},
		{
			name:     "custom resources",
			hostname: []string{"testing"},
			setupTestRun: func(k6 *v1alpha1.TestRun) {
				k6.Spec.Starter.Resources = corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("50m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				}

			},
			setupExpectedJob: func(j *batchv1.Job) {
				j.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("2Gi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("50m"),
						corev1.ResourceMemory: resource.MustParse("1Gi"),
					},
				}
			},
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {

			k6 := defaultTestRunForStarter()

			if tt.setupTestRun != nil {
				tt.setupTestRun(k6)
			}

			expectedJob := defaultExpectedJobForStarter()

			if tt.setupExpectedJob != nil {
				tt.setupExpectedJob(expectedJob)
			}

			job := NewStarterJob(k6, tt.hostname)

			diff := deep.Equal(job, expectedJob)

			if diff != nil {
				t.Errorf("NewStarterJob difference: %v", diff)
			}
		})
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
					SchedulerName:                "default-scheduler",
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
