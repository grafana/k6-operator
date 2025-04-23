package testrun

import (
	"fmt"

	"github.com/grafana/k6-operator/api/v1alpha1"
)

// Template is a draft of TestRun CR that can be used to create
// a new TestRun by copying and injecting new values
type Template v1alpha1.TestRun

func (t *Template) Create() *v1alpha1.TestRun {
	tr := v1alpha1.TestRun(*t)
	return tr.DeepCopy()
}

func PLZTestName(testRunId string) string {
	return fmt.Sprintf("plz-test-%s", testRunId)
}
