package controllers

import (
	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/grafana/k6-operator/pkg/types"
)

// hasCloudLifecycle reports whether the test run goes through k6 Cloud lifecycle
// (token, events, finalization): that is the case for a test run with cloud output
// and for any PLZ test run, whichever output it sends metrics with.
func hasCloudLifecycle(k6 *v1alpha1.TestRun, cli *types.CLI) bool {
	return cli.HasCloudOut || v1alpha1.IsTrue(k6, v1alpha1.CloudPLZTestRun)
}
