module github.com/grafana/k6-operator

go 1.13

require (
	github.com/go-logr/logr v0.1.0
	github.com/go-test/deep v1.0.7
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	k8s.io/api v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v0.18.6
	sigs.k8s.io/controller-runtime v0.6.2
)

replace github.com/grafana/k6-operator => ./
