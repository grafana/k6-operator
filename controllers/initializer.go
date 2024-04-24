package controllers

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"time"

	"go.k6.io/k6/cloudapi"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reconciler.io/runtime/reconcilers"
)

func InitializerReconciler(c reconcilers.Config) reconcilers.SubReconciler[v1alpha1.TestRunI] {
	return &reconcilers.SyncReconciler[v1alpha1.TestRunI]{
		Name:                   "InitializerJob",
		SyncDuringFinalization: false,
		SyncWithResult: func(ctx context.Context, tr v1alpha1.TestRunI) (res reconcilers.Result, err error) {
			// initializer is a quick job so check in frequently
			res = reconcilers.Result{RequeueAfter: time.Second * 5}

			log := logr.FromContextOrDiscard(ctx)
			gck6, ok := reconcilers.RetrieveValue(ctx, gck6ClientStashKey).(*cloudapi.Client)
			if !ok {
				return res, fmt.Errorf("expected stashed value for key %q", gck6ClientStashKey)
			}

			inspectOutput, inspectReady, err := inspectTestRun(ctx, log, tr, c.Client)

			if err != nil {
				// TODO: move to separate events handling
				//  input: tr, gck6 client, code of event, log. As method of gck6 client?

				// Cloud output test run is not created yet at this point, so sending
				// events is possible only for PLZ test run.
				if v1alpha1.IsTrue(tr, v1alpha1.CloudPLZTestRun) {
					// This error won't allow to start a test so let k6 Cloud know of it
					events := cloud.ErrorEvent(cloud.K6OperatorStartError).
						WithDetail(fmt.Sprintf("Failed to inspect the test script: %v", err)).
						WithAbort()
					cloud.SendTestRunEvents(gck6, v1alpha1.TestRunID(tr), log, events)
				}

				// inspectTestRun made a log message already so just return error without requeue
				return reconcilers.Result{Requeue: false}, err
			}
			if !inspectReady {
				return res, nil
			}
			log.Info(fmt.Sprintf("k6 inspect: %+v", inspectOutput))
			reconcilers.StashValue(ctx, inspectStashKey, inspectOutput)

			if int32(inspectOutput.MaxVUs) < tr.GetSpec().Parallelism {
				err = fmt.Errorf("number of instances > number of VUs")
				// TODO: surface this error as an event
				log.Error(err, "Parallelism argument cannot be larger than maximum VUs in the script",
					"maxVUs", inspectOutput.MaxVUs,
					"parallelism", tr.GetSpec().Parallelism)

				tr.GetStatus().Stage = "error"

				// if _, err := r.UpdateStatus(ctx, k6, log); err != nil {
				// 	return ctrl.Result{}, ready, err
				// }

				// Don't requeue in case of this error; unless it's made into a warning as described above.
				return reconcilers.Result{Requeue: false}, err
			}

			cli := types.ParseCLI(tr.GetSpec().Arguments)
			if cli.HasCloudOut {
				v1alpha1.UpdateCondition(tr, v1alpha1.CloudTestRun, metav1.ConditionTrue)

				if v1alpha1.IsUnknown(tr, v1alpha1.CloudTestRunCreated) {
					// In case of PLZ test run, this is already set to true
					v1alpha1.UpdateCondition(tr, v1alpha1.CloudTestRunCreated, metav1.ConditionFalse)
				}

				v1alpha1.UpdateCondition(tr, v1alpha1.CloudTestRunFinalized, metav1.ConditionFalse)
			} else {
				v1alpha1.UpdateCondition(tr, v1alpha1.CloudTestRun, metav1.ConditionFalse)
			}

			// if _, err := r.UpdateStatus(ctx, k6, log); err != nil {
			// 	return ctrl.Result{}, ready, err
			// }

			// targetImage, err := resolveTargetImage(ctx, c.Client, resource)
			// if err != nil {
			// return err
			// }
			// resource.Status.MarkImageResolved()
			// resource.Status.TargetImage = targetImage

			return reconcilers.Result{}, nil
		},
	}
}
