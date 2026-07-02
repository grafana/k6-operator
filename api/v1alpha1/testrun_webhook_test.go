package v1alpha1

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestRunDefaulter(t *testing.T) {
	d := &TestRunDefaulter{}
	ctx := context.Background()

	t.Run("sets defaults on empty spec", func(t *testing.T) {
		tr := &TestRun{}
		require.NoError(t, d.Default(ctx, tr))

		assert.Equal(t, DefaultParallelism, tr.Spec.Parallelism)
		assert.Equal(t, DefaultRunnerImage, tr.Spec.Runner.Image)
		assert.Equal(t, DefaultStarterImage, tr.Spec.Starter.Image)
	})

	t.Run("does not overwrite user-specified values", func(t *testing.T) {
		tr := &TestRun{}
		tr.Spec.Parallelism = 3
		tr.Spec.Runner.Image = "grafana/k6:v0.55.0"
		tr.Spec.Starter.Image = "my-registry/k6-operator:custom"

		require.NoError(t, d.Default(ctx, tr))

		assert.Equal(t, int32(3), tr.Spec.Parallelism)
		assert.Equal(t, "grafana/k6:v0.55.0", tr.Spec.Runner.Image)
		assert.Equal(t, "my-registry/k6-operator:custom", tr.Spec.Starter.Image)
	})

	t.Run("returns error for non-TestRun object", func(t *testing.T) {
		err := d.Default(ctx, &TestRunList{})
		assert.Error(t, err)
	})
}
