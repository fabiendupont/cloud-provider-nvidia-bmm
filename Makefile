# Image URL for building/pushing image targets
IMG ?= ghcr.io/fabiendupont/cloud-provider-nvidia-bmm:latest

# Get the currently used golang install path
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

.PHONY: all
all: build

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: ## Run tests.
	go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: fmt vet ## Build cloud controller manager binary.
	go build -o bin/nvidia-bmm-cloud-controller-manager cmd/nvidia-bmm-cloud-controller-manager/main.go

.PHONY: run
run: fmt vet ## Run cloud controller manager from your host (requires kubeconfig and cloud config).
	go run ./cmd/nvidia-bmm-cloud-controller-manager/main.go \
		--cloud-provider=nvidia-bmm \
		--cloud-config=./config/cloud-config.yaml \
		--use-service-account-credentials=false \
		--kubeconfig=${KUBECONFIG}

.PHONY: docker-build
docker-build: ## Build docker image.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image.
	docker push ${IMG}

##@ Deployment

.PHONY: deploy
deploy: ## Deploy cloud controller manager to K8s cluster.
	kubectl apply -f deploy/rbac/
	kubectl apply -f deploy/manifests/

.PHONY: undeploy
undeploy: ## Undeploy cloud controller manager from K8s cluster.
	kubectl delete -f deploy/manifests/ --ignore-not-found=true
	kubectl delete -f deploy/rbac/ --ignore-not-found=true

.PHONY: clean
clean: ## Clean build artifacts.
	rm -rf bin/
	rm -f cover.out
