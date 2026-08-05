package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-logr/logr"
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.k6.io/k6/v2/cloudapi"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFindOOMKilledContainer(t *testing.T) {
	tests := []struct {
		name string
		pod  corev1.Pod
		want bool
	}{
		{
			name: "terminated runner",
			pod: corev1.Pod{
				Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
					Name:  "k6",
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: oomKilledReason}},
				}}},
			},
			want: true,
		},
		{
			name: "restarted runner",
			pod: corev1.Pod{
				Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
					Name:                 "k6",
					LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: oomKilledReason}},
				}}},
			},
			want: true,
		},
		{
			name: "init container",
			pod: corev1.Pod{
				Status: corev1.PodStatus{InitContainerStatuses: []corev1.ContainerStatus{{
					Name:  "init",
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: oomKilledReason}},
				}}},
			},
			want: true,
		},
		{
			name: "other failure",
			pod: corev1.Pod{
				Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
					Name:  "k6",
					State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: "Error"}},
				}}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			podName, containerName, found := findOOMKilledContainer([]corev1.Pod{tt.pod})

			require.Equal(t, tt.want, found)
			if tt.want {
				require.Equal(t, tt.pod.Name, podName)
				require.NotEmpty(t, containerName)
			}
		})
	}
}

func TestCheckRunnerOOMKilled(t *testing.T) {
	for _, stage := range []v1alpha1.Stage{"created", "started"} {
		t.Run("moves an OSS "+string(stage)+" test run to error", func(t *testing.T) {
			k6 := newOOMTestRun(false)
			k6.Status.Stage = stage
			v1alpha1.UpdateCondition(k6, v1alpha1.TestRunRunning, metav1.ConditionTrue)
			r := newOOMReconciler(t, k6, newOOMPod(k6))

			found, err := checkRunnerOOMKilled(context.Background(), logr.Discard(), k6, r, nil)

			require.NoError(t, err)
			require.True(t, found)
			updated := &v1alpha1.TestRun{}
			require.NoError(t, r.Get(context.Background(), client.ObjectKeyFromObject(k6), updated))
			require.Equal(t, v1alpha1.Stage("error"), updated.Status.Stage)
			require.True(t, v1alpha1.IsFalse(updated, v1alpha1.TestRunRunning))
		})
	}

	t.Run("requests a cloud test run abort", func(t *testing.T) {
		k6 := newOOMTestRun(true)
		r := newOOMReconciler(t, k6, newOOMPod(k6))
		var received cloud.Events
		var method, path string
		var decodeErr error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			method = req.Method
			path = req.URL.Path
			decodeErr = json.NewDecoder(req.Body).Decode(&received)
			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()
		cloudClient := cloudapi.NewClient(logrus.New(), "", server.URL, "test", time.Second)

		found, err := checkRunnerOOMKilled(context.Background(), logr.Discard(), k6, r, cloudClient)

		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, http.MethodPost, method)
		require.Equal(t, "/orchestrator/v1/testruns/123/events", path)
		require.NoError(t, decodeErr)
		require.Len(t, received, 2)
		require.Equal(t, "TestRunErrorEvent", string(received[0].EventType))
		require.Equal(t, cloud.OOMError, received[0].ErrorCode)
		require.Equal(t, "runner pod sample-1 container k6 was terminated with reason OOMKilled", received[0].Detail)
		require.Equal(t, "TestRunAbortEvent", string(received[1].EventType))
	})

	t.Run("ignores unrelated pods", func(t *testing.T) {
		k6 := newOOMTestRun(false)
		pod := newOOMPod(k6)
		pod.Labels = nil
		r := newOOMReconciler(t, k6, pod)

		found, err := checkRunnerOOMKilled(context.Background(), logr.Discard(), k6, r, nil)

		require.NoError(t, err)
		require.False(t, found)
	})
}

func newOOMTestRun(cloudRun bool) *v1alpha1.TestRun {
	k6 := &v1alpha1.TestRun{
		ObjectMeta: metav1.ObjectMeta{Name: "sample", Namespace: "default"},
		Spec:       v1alpha1.TestRunSpec{Parallelism: 1},
		Status:     v1alpha1.TestRunStatus{Stage: "started"},
	}
	v1alpha1.Initialize(k6)
	k6.Status.Stage = "started"
	if cloudRun {
		v1alpha1.UpdateCondition(k6, v1alpha1.CloudTestRun, metav1.ConditionTrue)
		k6.Status.TestRunID = "123"
	}
	return k6
}

func newOOMPod(k6 *v1alpha1.TestRun) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sample-1",
			Namespace: k6.Namespace,
			Labels:    map[string]string{"app": "k6", "k6_cr": k6.Name, "runner": "true"},
		},
		Status: corev1.PodStatus{ContainerStatuses: []corev1.ContainerStatus{{
			Name:  "k6",
			State: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{Reason: oomKilledReason}},
		}}},
	}
}

func newOOMReconciler(t *testing.T, k6 *v1alpha1.TestRun, pods ...*corev1.Pod) *TestRunReconciler {
	scheme := runtime.NewScheme()
	require.NoError(t, v1alpha1.AddToScheme(scheme))
	require.NoError(t, corev1.AddToScheme(scheme))
	objects := []client.Object{k6.DeepCopy()}
	for _, pod := range pods {
		objects = append(objects, pod)
	}
	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&v1alpha1.TestRun{}).
		WithObjects(objects...).
		Build()
	return &TestRunReconciler{Client: k8sClient}
}
