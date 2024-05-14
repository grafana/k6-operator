package controllers

import (
	"context"
	"io"
	"strings"
	"time"

	k6types "go.k6.io/k6/lib/types"

	"github.com/grafana/k6-operator/pkg/cloud"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	clientgotesting "k8s.io/client-go/testing"
)

type contextKey string // for SA1029 staticcheck

var (
	mockContextKey = contextKey("test-pod-log")

	// to check output from fakePodLogs(), see its impl. below
	mockInspectOutput = cloud.InspectOutput{
		MaxVUs:        2,
		TotalDuration: k6types.NullDurationFrom(time.Second * 5),
	}
)

func fakeGetRestClient() (kubernetes.Interface, error) {
	cset := fake.NewSimpleClientset()
	f := clientgotesting.ReactionFunc(func(action clientgotesting.Action) (handled bool, ret runtime.Object, err error) {
		// nothing to inject ATM, as the runtime.Object is ignored here
		// TODO
		return
	})
	cset.AddReactor("get", "pods/log", f)
	return cset, nil
}

func fakePodLogs(ctx context.Context, ns, name string) (io.ReadCloser, error) {
	s := `{"totalDuration": "5s","maxVUs": 2, "thresholds": null}`
	ctxV, ok := ctx.Value(mockContextKey).(string)
	if ok && len(ctxV) > 0 {
		s = ctxV
	}
	reader := strings.NewReader(s)
	return io.NopCloser(reader), nil
}
