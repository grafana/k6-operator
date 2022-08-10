# Current Operator version
VERSION ?= 0.0.7
# Default bundle image tag
BUNDLE_IMG ?= k6-controller-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image to use for building Go
GO_BUILDER_IMG ?= "golang:1.17"
# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/grafana/operator:latest
# Default dockerfile to build
DOCKERFILE ?= "Dockerfile.controller"
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,crdVersions=v1"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
ENVTEST_K8S_VERSION ?= 1.19.2
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
KUBEBUILDER_ASSETS_ROOT=/tmp
KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS_ROOT)/kubebuilder/bin
test: generate fmt vet manifests
	export KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS); setup-envtest use --use-env -p env $(ENVTEST_K8S_VERSION); go test ./... -coverprofile cover.out

test-setup:
	curl -L -O "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-$(ENVTEST_K8S_VERSION)-$(GOOS)-$(GOARCH).tar.gz"
	tar -zxvf kubebuilder-tools-$(ENVTEST_K8S_VERSION)-$(GOOS)-$(GOARCH).tar.gz
	mv kubebuilder $(KUBEBUILDER_ASSETS_ROOT)
	export KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS); go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

test-setup-ci:
	curl -L -O "https://storage.googleapis.com/kubebuilder-tools/kubebuilder-tools-$(ENVTEST_K8S_VERSION)-$(GOOS)-$(GOARCH).tar.gz"
	tar -xvf kubebuilder-tools-$(ENVTEST_K8S_VERSION)-$(GOOS)-$(GOARCH).tar.gz
	mv kubebuilder $(KUBEBUILDER_ASSETS_ROOT)
	export KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS); go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

e2e: deploy
	kubectl create configmap crocodile-stress-test --from-file e2e/test.js
	kubectl apply -f e2e/test.yaml

e2e-cleanup:
	kubectl delete configmap crocodile-stress-test
	kubectl delete -f e2e/test.yaml

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

# Delete operator from a cluster
delete: manifests kustomize
	$(KUSTOMIZE) build config/default | kubectl delete -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
	docker build . -t ${IMG} -f ${DOCKERFILE} --build-arg GO_BUILDER_IMG=${GO_BUILDER_IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.5 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

# Build the bundle image.
.PHONY: bundle-build
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
