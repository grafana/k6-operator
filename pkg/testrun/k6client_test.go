package testrun

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_RunTeardownNoHost(t *testing.T) {
	err := RunTeardown(context.Background(), []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no k6 Service is available to run teardown")
}

func Test_SetSetupDataNoHost(t *testing.T) {
	data := json.RawMessage(`{"foo":"bar"}`)
	err := SetSetupData(context.Background(), []string{}, data)
	assert.NoError(t, err)
}
