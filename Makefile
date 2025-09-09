# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Current Operator version
VERSION ?= 0.0.23
# Image to use for building Go
GO_BUILDER_IMG ?= "golang:1.25"
# Image URL to use all building/pushing image targets
IMG_NAME ?= ghcr.io/grafana/k6-operator
IMG_TAG ?= latest
# Default dockerfile to build
DOCKERFILE ?= "Dockerfile.controller"

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

CRD_OPTIONS ?= "crd:maxDescLen=0"

.PHONY: all
all: build ## Build the manifests and binary (default target).

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# Run tests
GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)
KUBEBUILDER_ASSETS_ROOT=/tmp
KUBEBUILDER_ASSETS=$(KUBEBUILDER_ASSETS_ROOT)/kubebuilder/bin

.PHONY: test
test: manifests generate fmt vet envtest ## Run unit tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

e2e: deploy ## Run an end-to-end test.
	kubectl create configmap crocodile-stress-test --from-file e2e/test.js
	kubectl apply -f e2e/test.yaml

e2e-cleanup: ## Clean up end-to-end test resources.
	kubectl delete configmap crocodile-stress-test
	kubectl delete -f e2e/test.yaml

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

manifests: controller-gen ## Generate manifests (CRD, RBAC, etc.).
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	$(GOLANGCI_LINT) --timeout 5m run ./...

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

generate: controller-gen ## Generate code (controllers, deepcopy, etc.).
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

docker-build: test ## Build the docker image.
	docker build . -t ${IMG_NAME}:${IMG_TAG} -f ${DOCKERFILE} --build-arg GO_BUILDER_IMG=${GO_BUILDER_IMG}

docker-push: ## Push the docker image.
	docker push ${IMG_NAME}:${IMG_TAG}

# TODO: check and re-use in bundle.yaml creation
# .PHONY: build-installer
# build-installer: manifests generate kustomize ## Generate a consolidated YAML with CRDs and deployment.
# 	mkdir -p dist
# 	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
# 	$(KUSTOMIZE) build config/default > dist/install.yaml

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

deploy: manifests kustomize ## Deploy controller to the cluster configured in $KUBECONFIG
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG_NAME}:${IMG_TAG}
	$(KUSTOMIZE) build config/default | $(KUBECTL) apply -f -

undeploy: manifests kustomize ## Remove operator from the cluster.
	$(KUSTOMIZE) build config/default | $(KUBECTL) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: bundle
bundle: manifests ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG_NAME}:${IMG_TAG}
	$(KUSTOMIZE) build config/manifests | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

generate-crd-docs: ## Generate CRD documentation.
	# Generate yamls with full desciption values
	$(CONTROLLER_GEN) "crd" rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	go install fybrik.io/crdoc@v0.6.4
	crdoc --resources ./config/crd/bases --output ./docs/crd-generated.md --template ./docs/crd.tmpl
	# Restore yamls to the original state
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# ===============================================================
# This section is only about the HELM deployment of the operator
# ===============================================================

e2e-helm: deploy-helm ## Run an end-to-end test with Helm deployment.
	kubectl create configmap crocodile-stress-test --from-file e2e/test.js
	kubectl apply -f e2e/test.yaml

deploy-helm: manifests helm ## Deploy operator using local Helm chart.
	$(HELM) upgrade --install --wait k6-operator ./charts/k6-operator -f ./charts/k6-operator/values.yaml --set manager.image.name=$(IMG_NAME) --set manager.image.tag=$(IMG_TAG)

helm-template: manifests helm ## Generate Helm template output from the local chart.
	$(HELM) template k6-operator ./charts/k6-operator -f ./charts/k6-operator/values.yaml --set manager.image.name=$(IMG_NAME) --set manager.image.tag=$(IMG_TAG)

helm-docs: ## Generate Helm chart documentation.
	go install github.com/norwoodj/helm-docs/cmd/helm-docs@v1.14.2
	$(shell go env GOPATH)/bin/helm-docs

helm-schema: ## Generate Helm chart JSON schema.
	go install github.com/dadav/helm-schema/cmd/helm-schema@0.12.0
	$(shell go env GOPATH)/bin/helm-schema --chart-search-root ./charts/k6-operator

delete-helm: manifests helm ## Uninstall Helm deployment.
	$(HELM) uninstall k6-operator

helm: ## Install Helm if not already available.
ifeq (, $(shell which helm))
	@{ \
	set -e ;\
	HELM_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$HELM_GEN_TMP_DIR ;\
	curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3
	chmod 700 get_helm.sh ;\
	./get_helm.sh ;\
	rm -rf $$HELM_GEN_TMP_DIR ;\
	}
HELM=$(shell which helm)
else
HELM=$(shell which helm)
endif

# ===============================================================
# Dependencies
# ===============================================================

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.18.0
# ENVTEST_VERSION ?= latest # ref. https://github.com/kubernetes-sigs/controller-runtime/tree/main/tools/setup-envtest
ENVTEST_VERSION ?= release-0.19
GOLANGCI_LINT_VERSION ?= v2.4.0
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.31.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef
