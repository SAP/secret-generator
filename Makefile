# Image URL to use all building/pushing image targets
IMG ?= secret-generator:latest
# K8s version used by envtest
ENVTEST_K8S_VERSION = 1.30.3

# Set shell to bash
SHELL = /usr/bin/env bash
.SHELLFLAGS = -o pipefail -ec

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

##@ Testing

.PHONY: test
test: fmt vet envtest ## Run tests
	KUBEBUILDER_ASSETS="$(LOCALBIN)/k8s/current" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: fmt vet ## Build webhook binary
	go build -o bin/webhook ./cmd/webhook

.PHONY: run
run: fmt vet ## Run a controller from your host
	go run ./cmd/webhook

# Build docker image in current architecture and tag it as ${IMG}
.PHONY: docker-build
docker-build: ## Build docker image with the webhook
	docker build -t ${IMG} .

# Push docker image to the target specified in ${IMG}
.PHONY: docker-push
docker-push: ## Push docker image with the webhook
	docker push ${IMG}

# Build and push docker image for all given platforms
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the webhook for cross-platform support
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} .
	- docker buildx rm project-v3-builder

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	@mkdir -p $(LOCALBIN)

.PHONY: setup-envtest
setup-envtest: $(LOCALBIN) ## Install setup-envtest
	@go mod download sigs.k8s.io/controller-runtime/tools/setup-envtest && \
	VERSION=$$(go list -m -f '{{.Version}}' sigs.k8s.io/controller-runtime/tools/setup-envtest) && \
	if [ ! -L $(LOCALBIN)/setup-envtest ] || [ "$$(readlink $(LOCALBIN)/setup-envtest)" != "setup-envtest-$$VERSION" ]; then \
	echo "Installing setup-envtest $$VERSION" && \
	rm -f $(LOCALBIN)/setup-envtest && \
	GOBIN=$(LOCALBIN) go install $$(go list -m -f '{{.Dir}}' sigs.k8s.io/controller-runtime/tools/setup-envtest) && \
	mv $(LOCALBIN)/setup-envtest $(LOCALBIN)/setup-envtest-$$VERSION && \
	ln -s setup-envtest-$$VERSION $(LOCALBIN)/setup-envtest; \
	fi

.PHONY: envtest
envtest: setup-envtest ## Install envtest binaries
	@ENVTESTDIR=$$($(LOCALBIN)/setup-envtest use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path) && \
	chmod -R u+w $$ENVTESTDIR && \
	rm -f $(LOCALBIN)/k8s/current && \
	ln -s $$ENVTESTDIR $(LOCALBIN)/k8s/current

