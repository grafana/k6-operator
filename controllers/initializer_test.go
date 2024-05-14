package controllers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"go.k6.io/k6/cloudapi"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/resources/jobs"
	"github.com/grafana/k6-operator/pkg/types"

	reconcilers "reconciler.io/runtime/reconcilers"
	rtesting "reconciler.io/runtime/testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	clientgotesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Test_InitializerSubReconciler(t *testing.T) {
	getRestClientF = fakeGetRestClient
	podLogsF = fakePodLogs

	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)
	_ = clientgoscheme.AddToScheme(scheme)

	stashed := map[reconcilers.StashKey]interface{}{
		gck6ClientStashKey: &cloudapi.Client{},
	}

	trName := "testrun"
	script := &types.Script{}
	initializerPod := jobs.Job().Name(trName).
		Initializer(script).GetPod()

	initializerPodRunning := initializerPod.DeepCopy()
	initializerPodRunning.Status.Phase = corev1.PodRunning

	initializerPodSucceeded := initializerPod.DeepCopy()
	initializerPodSucceeded.Status.Phase = corev1.PodSucceeded

	t.Run("InitializerSubReconciler_Waiting", func(t *testing.T) {
		// TODO: move out TestRun objects
		tr := &v1alpha1.TestRun{
			ObjectMeta: metav1.ObjectMeta{
				Name: trName,
			},
		}
		// TODO: add initial conditions to tr

		trHighParallelism := tr.DeepCopy()
		trHighParallelism.GetSpec().Parallelism = int32(mockInspectOutput.MaxVUs) + 1

		trCloudOutput := tr.DeepCopy()
		trCloudOutput.GetSpec().Arguments = "--out cloud"

		trPLZ := trCloudOutput.DeepCopy()
		trPLZ.GetSpec().TestRunID = "123"
		trPLZ.GetStatus().TestRunID = trPLZ.GetSpec().TestRunID
		trPLZ.GetStatus().Conditions = []metav1.Condition{
			{
				Type:   v1alpha1.CloudPLZTestRun,
				Status: metav1.ConditionTrue,
				Reason: "CloudPLZTestRunTrue",
			},
			{
				Type:   v1alpha1.CloudTestRunCreated,
				Status: metav1.ConditionTrue,
				Reason: "CloudTestRunCreatedTrue",
			},
			{
				Type:   v1alpha1.CloudTestRunFinalized,
				Status: metav1.ConditionFalse,
				Reason: "CloudTestRunFinalizedFalse",
			},
		}

		rts := rtesting.SubReconcilerTests[v1alpha1.TestRunI]{
			"failure on listing pods": {
				Resource:           tr,
				GivenStashedValues: stashed,
				ExpectedResult:     reconcilers.Result{Requeue: false},
				ShouldErr:          true,
				WithReactors: []clientgotesting.ReactionFunc{
					clientgotesting.ReactionFunc(func(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
						if action.Matches("list", "PodList") {
							err = fmt.Errorf("simulating connection error")
							handled = true
						}
						return
					}),
				},
			},

			"initializer pod not found": {
				Resource:           tr,
				GivenStashedValues: stashed,
				ExpectedResult:     reconcilers.Result{RequeueAfter: time.Second * 5},
			},

			"initializer pod found but not started": {
				Resource:           tr,
				GivenStashedValues: stashed,
				GivenObjects: []client.Object{
					initializerPod,
				},
				ExpectedResult: reconcilers.Result{RequeueAfter: time.Second * 5},
			},

			"initializer pod found and started": {
				Resource:           tr,
				GivenStashedValues: stashed,
				GivenObjects: []client.Object{
					initializerPodRunning,
				},
				ExpectedResult: reconcilers.Result{RequeueAfter: time.Second * 5},
			},

			"initializer pod found and succeeded with error output": {
				Resource:           tr,
				GivenStashedValues: stashed,
				GivenObjects: []client.Object{
					initializerPodSucceeded,
				},
				Prepare: func(t *testing.T, ctx context.Context, tc *rtesting.SubReconcilerTestCase[v1alpha1.TestRunI]) (context.Context, error) {
					ctx = context.WithValue(ctx, mockContextKey, "foobar error")
					return ctx, nil
				},
				ShouldErr:      true,
				ExpectedResult: reconcilers.Result{Requeue: false},
			},

			// the tests below are for correct JSON output of initializer pod

			"initializer pod found and succeeded with correct output": {
				Resource:           tr,
				GivenStashedValues: stashed,
				GivenObjects: []client.Object{
					initializerPodSucceeded,
				},
				ExpectedResult: reconcilers.Result{},
				ExpectStashedValues: map[reconcilers.StashKey]interface{}{
					inspectStashKey: mockInspectOutput,
				},
				ExpectResource: func() v1alpha1.TestRunI {
					updated := tr.DeepCopy()
					updated.GetStatus().Conditions = []metav1.Condition{
						{
							Type:   v1alpha1.CloudTestRun,
							Status: metav1.ConditionFalse,
							Reason: "CloudTestRunFalse",
						},
					}
					return updated
				}(),
			},

			"initializer pod succeeded but test is misconfigured": {
				Resource:           trHighParallelism,
				GivenStashedValues: stashed,
				GivenObjects: []client.Object{
					initializerPodSucceeded,
				},
				ExpectedResult: reconcilers.Result{Requeue: false},
				ShouldErr:      true,
				ExpectStashedValues: map[reconcilers.StashKey]interface{}{
					inspectStashKey: mockInspectOutput,
				},
				ExpectResource: func() v1alpha1.TestRunI {
					updated := trHighParallelism.DeepCopy()
					updated.GetStatus().Stage = "error"
					return updated
				}(),
			},

			"initializer pod succeeded for cloud output testrun": {
				Resource:           trCloudOutput,
				GivenStashedValues: stashed,
				GivenObjects: []client.Object{
					initializerPodSucceeded,
				},
				ExpectedResult: reconcilers.Result{},
				ExpectStashedValues: map[reconcilers.StashKey]interface{}{
					inspectStashKey: mockInspectOutput,
				},
				ExpectResource: func() v1alpha1.TestRunI {
					updated := trCloudOutput.DeepCopy()
					updated.GetStatus().Conditions = []metav1.Condition{
						{
							Type:   v1alpha1.CloudTestRun,
							Status: metav1.ConditionTrue,
							Reason: "CloudTestRunTrue",
						},
						{
							Type:   v1alpha1.CloudTestRunCreated,
							Status: metav1.ConditionFalse,
							Reason: "CloudTestRunCreatedFalse",
						},
						{
							Type:   v1alpha1.CloudTestRunFinalized,
							Status: metav1.ConditionFalse,
							Reason: "CloudTestRunFinalizedFalse",
						},
					}
					return updated
				}(),
			},

			"initializer pod succeeded for PLZ testrun": {
				Resource:           trPLZ,
				GivenStashedValues: stashed,
				GivenObjects: []client.Object{
					initializerPodSucceeded,
				},
				ExpectedResult: reconcilers.Result{},
				ExpectStashedValues: map[reconcilers.StashKey]interface{}{
					inspectStashKey: mockInspectOutput,
				},
				ExpectResource: func() v1alpha1.TestRunI {
					updated := trPLZ.DeepCopy()
					meta.SetStatusCondition(&updated.GetStatus().Conditions, metav1.Condition{
						Type:   v1alpha1.CloudTestRun,
						Status: metav1.ConditionTrue,
						Reason: "CloudTestRunTrue",
					})
					return updated
				}(),
			},
		}

		rts.Run(t, scheme, func(t *testing.T, rtc *rtesting.SubReconcilerTestCase[v1alpha1.TestRunI], c reconcilers.Config) reconcilers.SubReconciler[v1alpha1.TestRunI] {
			return InitializerReconciler(c)
		})
	})
}
