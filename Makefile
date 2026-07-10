# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Current Operator version
VERSION ?= 1.5.0
# Image URL to use all building/pushing image targets
IMG_NAME ?= ghcr.io/grafana/k6-operator
IMG_TAG ?= latest
YQ_IMAGE ?= mikefarah/yq:4.53.2@sha256:0cb4a78491b6e62ee8a9bf4fbeacbd15b5013d19bc420591b05383a696315e60
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
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test -race $$(go list ./... | grep -v /e2e) -coverprofile cover.out

e2e: deploy ## Run an end-to-end test.
	kubectl create configmap crocodile-stress-test --from-file e2e/test.js
	kubectl apply -f e2e/test.yaml

e2e-cleanup: ## Clean up end-to-end test resources.
	kubectl delete configmap crocodile-stress-test
	kubectl delete -f e2e/test.yaml

e2e-update-latest: kustomize ## Update e2e/latest folder with the bundle.yaml.
	echo -e "Regenerate ./e2e/latest from the bundle.yaml"
	rm e2e/latest/*
	cp bundle.yaml ./e2e/latest/bundle-to-test.yaml
	cd e2e/latest && \
	docker run --user "$$(id -u):$$(id -g)" --rm -v "${PWD}/e2e/latest":/workdir $(YQ_IMAGE) --no-doc  -s  '.kind + "-" + .metadata.name' bundle-to-test.yaml && \
	for f in $$(find . -type f  -name '*.k6.io'); do mv $$f $${f}.yaml; done && \
	rm bundle-to-test.yaml && \
	$(KUSTOMIZE) create --autodetect --recursive .

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
	docker build . -t ${IMG_NAME}:${IMG_TAG} -f ${DOCKERFILE}

docker-push: ## Push the docker image.
	docker push ${IMG_NAME}:${IMG_TAG}

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

.PHONY: release-prepare
release-prepare: kustomize ## Prepare generated files for a release. Requires OPERATOR_VERSION=vX.Y.Z and CHART_VERSION=X.Y.Z.
	OPERATOR_VERSION="$(OPERATOR_VERSION)" \
	CHART_VERSION="$(CHART_VERSION)" \
	PREVIOUS_OPERATOR_VERSION="$(PREVIOUS_OPERATOR_VERSION)" \
	RUNNER_K6_VERSION="$(RUNNER_K6_VERSION)" \
	./hack/release/prepare-release.sh
	$(MAKE) generate-crd-docs
	$(MAKE) bundle-yaml OPERATOR_VERSION=$(OPERATOR_VERSION)
	$(MAKE) patch-helm-crd
	$(MAKE) helm-docs
	$(MAKE) helm-schema
	$(MAKE) e2e-update-latest

# In the kustomize image override below, the "*" newName tells kustomize to keep
# the existing newName (ghcr.io/grafana/k6-operator) and set only the tag.
.PHONY: bundle-yaml
bundle-yaml: kustomize ## Generate bundle.yaml with the controller image set to controller-$(OPERATOR_VERSION). Requires OPERATOR_VERSION=vX.Y.Z.
	tmp_dir="$$(mktemp -d)" ;\
	trap 'rm -rf "$$tmp_dir"' EXIT ;\
	cp -R config "$$tmp_dir/config" ;\
	cd "$$tmp_dir/config/default" ;\
	$(KUSTOMIZE) edit set image ghcr.io/grafana/k6-operator=*:controller-$(OPERATOR_VERSION) ;\
	$(KUSTOMIZE) build . > "$(CURDIR)/bundle.yaml"

generate-crd-docs: controller-gen ## Generate CRD documentation.
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
	$(HELM) template k6-operator ./charts/k6-operator -f ./charts/k6-operator/values.yaml --set manager.image.repository=$(IMG_NAME) --set manager.image.tag=$(IMG_TAG)

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
	curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/$(HELM_VERSION)/scripts/get-helm-3 ;\
	chmod 700 get_helm.sh ;\
	DESIRED_VERSION=$(HELM_VERSION) ./get_helm.sh ;\
	rm -rf $$HELM_GEN_TMP_DIR ;\
	}
HELM=$(shell which helm)
else
HELM=$(shell which helm)
endif

.PHONY: patch-helm-crd
patch-helm-crd: kustomize ## Copy CRDs from config/crd/bases to Helm chart templates with Helm templating.
	cp config/crd/bases/k6.io_testruns.yaml charts/k6-operator/templates/crds/testrun.yaml
	sed -i '1i\{{- if .Values.installCRDs -}}' charts/k6-operator/templates/crds/testrun.yaml
	sed -i '/^metadata:$$/a\  labels:\n    app.kubernetes.io/component: controller\n    {{- include "k6-operator.labels" . | nindent 4 }}\n    {{- include "k6-operator.customLabels" . | nindent 4 }}' charts/k6-operator/templates/crds/testrun.yaml
	sed -i '0,/^  annotations:$$/s//  annotations:\n    {{- include "k6-operator.customAnnotations" . | nindent 4 }}/' charts/k6-operator/templates/crds/testrun.yaml
	echo '{{- end -}}' >> charts/k6-operator/templates/crds/testrun.yaml
	$(KUSTOMIZE) build config/crd | docker run -i --rm $(YQ_IMAGE) 'select(.metadata.name == "privateloadzones.k6.io")' > charts/k6-operator/templates/crds/plz.yaml
	sed -i '1i\{{- if .Values.installCRDs -}}' charts/k6-operator/templates/crds/plz.yaml
	sed -i '/^metadata:$$/a\  labels:\n    app.kubernetes.io/component: controller\n    {{- include "k6-operator.labels" . | nindent 4 }}\n    {{- include "k6-operator.customLabels" . | nindent 4 }}' charts/k6-operator/templates/crds/plz.yaml
	sed -i '0,/^  annotations:$$/s//  annotations:\n    {{- include "k6-operator.customAnnotations" . | nindent 4 }}/' charts/k6-operator/templates/crds/plz.yaml
	echo '{{- end -}}' >> charts/k6-operator/templates/crds/plz.yaml

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
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint-kube-api-linter

## Tool Versions
KUSTOMIZE_VERSION ?= v5.5.0
CONTROLLER_TOOLS_VERSION ?= v0.19.0
HELM_VERSION ?= v3.7.2
# ENVTEST_VERSION ?= latest # ref. https://github.com/kubernetes-sigs/controller-runtime/tree/main/tools/setup-envtest
ENVTEST_VERSION ?= release-0.19
GOLANGCI_LINT_VERSION ?= v2.12.1
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
golangci-lint: $(GOLANGCI_LINT) ## Build golangci-lint with kube-api-linter plugin.
# Rebuild whenever .custom-gcl.yml changes (e.g. plugin or version bumps) so a
# stale binary is never reused with an outdated plugin set. The bootstrap
# golangci-lint is installed via go-install-tool, then `golangci-lint custom`
# builds the plugin-enabled binary. Per the kube-api-linter docs the custom
# binary is named distinctly (golangci-lint-kube-api-linter) so it never
# overwrites the running bootstrap binary (which fails on Linux with
# "text file busy").
$(GOLANGCI_LINT): $(LOCALBIN) .custom-gcl.yml
	@echo "Building golangci-lint with kube-api-linter plugin..."
	$(call go-install-tool,$(LOCALBIN)/golangci-lint,github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))
	$(LOCALBIN)/golangci-lint custom

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
