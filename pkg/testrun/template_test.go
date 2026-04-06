package testrun

import (
	"testing"

	"github.com/grafana/k6-operator/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_PLZTestName(t *testing.T) {
	assert.Equal(t, "plz-test-abc123", PLZTestName("abc123"))
	assert.Equal(t, "plz-test-0", PLZTestName("0"))
	assert.Equal(t, "plz-test-", PLZTestName("")) // no validation at this point
}

func Test_TemplateCreate(t *testing.T) {
	tmpl := Template(v1alpha1.TestRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "base",
			Namespace: "ns",
		},
	})

	tr1 := tmpl.Create()
	tr2 := tmpl.Create()

	assert.Equal(t, "base", tr1.Name)
	assert.Equal(t, "ns", tr1.Namespace)
	assert.Equal(t, "base", tr2.Name)
	assert.Equal(t, "ns", tr2.Namespace)

	// mutations to one copy must not have side effects
	tr1.Name = "modified"

	assert.Equal(t, "modified", tr1.Name)
	assert.Equal(t, "base", tr2.Name)

	assert.Equal(t, "base", Template(v1alpha1.TestRun(tmpl)).Name)
}
