package plz

import (
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	// according to the docs, scheme should be thread-safe at this point:
	// https://pkg.go.dev/k8s.io/apimachinery/pkg/runtime#Scheme
	// so let's try this
	scheme *runtime.Scheme
)

func SetScheme(s *runtime.Scheme) {
	scheme = s
}
