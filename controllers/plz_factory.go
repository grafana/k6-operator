package controllers

import (
	"context"
	"fmt"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/cloud"
	"github.com/grafana/k6-operator/pkg/testrun"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *PrivateLoadZoneReconciler) startFactory(plz *v1alpha1.PrivateLoadZone, testRunCh chan string) {
	go func() {
		for testRunId := range testRunCh {
			// First check if such a test already exists
			namespacedName := types.NamespacedName{
				Name:      testrun.TestName(testRunId),
				Namespace: plz.Namespace,
			}

			k6 := &v1alpha1.K6{}
			if err := r.Get(context.Background(), namespacedName, k6); err == nil || !errors.IsNotFound(err) {
				r.Log.Info(fmt.Sprintf("Test run `%s` has already been started.", testRunId))
				// fmt.Println(k6)
				continue
			}

			// Test does not exist so get its data and create it.

			// TODO get test run data

			trData := cloud.TestRunData{
				TestRunId: testRunId,
				Instances: 1,
			}

			k6 = testrun.NewPLZTestRun(plz, trData)

			r.Log.Info(fmt.Sprintf("PLZ test run has been prepared with image `%s` and `%d` instances",
				k6.Spec.Runner.Image, k6.Spec.Parallelism), "testRunId", testRunId)

			if err := ctrl.SetControllerReference(plz, k6, r.Scheme); err != nil {
				r.Log.Error(err, "Failed to set controller reference for the PLZ test run", "testRunId", testRunId)
			}

			if err := r.Create(context.Background(), k6); err != nil {
				r.Log.Error(err, "Failed to create PLZ test run", "testRunId", testRunId)
			}

			r.Log.Info("Created new test run", "testRunId", testRunId)
		}
	}()
}
