# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

PATH := $(shell go env GOPATH)/bin:$(PATH)

include vmkfstools-wrapper/version.mk

.PHONY: all
all: build

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test: fmt vet
	go test $$(go list ./... | grep -v /e2e) -coverprofile cover.out

.PHONY: test-e2e
test-e2e: build
	@echo "Running E2E tests..."
	@echo "Checking required environment variables..."
	@if [ -z "$$VMWARE_PROVIDER_NAME" ]; then echo "ERROR: VMWARE_PROVIDER_NAME is not set"; exit 1; fi
	@if [ -z "$$VMWARE_URL" ]; then echo "ERROR: VMWARE_URL is not set"; exit 1; fi
	@if [ -z "$$VMWARE_USERNAME" ]; then echo "ERROR: VMWARE_USERNAME is not set"; exit 1; fi
	@if [ -z "$$VMWARE_PASSWORD" ]; then echo "ERROR: VMWARE_PASSWORD is not set"; exit 1; fi
	@if [ -z "$$SOURCE_DATASTORE" ]; then echo "ERROR: SOURCE_DATASTORE is not set"; exit 1; fi
	@if [ -z "$$SOURCE_VM_NAME" ]; then echo "ERROR: SOURCE_VM_NAME is not set"; exit 1; fi
	@if [ -z "$$TARGET_STORAGE_CLASS" ]; then echo "ERROR: TARGET_STORAGE_CLASS is not set"; exit 1; fi
	@if [ -z "$$STORAGE_VENDOR_PRODUCT" ]; then echo "ERROR: STORAGE_VENDOR_PRODUCT is not set"; exit 1; fi
	@if [ -z "$$STORAGE_SECRET_NAME" ]; then echo "ERROR: STORAGE_SECRET_NAME is not set"; exit 1; fi
	@if [ -z "$$TARGET_NAMESPACE" ]; then echo "ERROR: TARGET_NAMESPACE is not set"; exit 1; fi
	@if [ -z "$$KUBECONFIG" ] && [ ! -f "$$HOME/.kube/config" ]; then echo "ERROR: KUBECONFIG is not set and default kubeconfig not found"; exit 1; fi
	@echo "All required environment variables are set."
	@echo "Running E2E test: Migrate a thin disk VM..."
	go test ./e2e -v

.PHONY: generate 
generate:
	go generate ./...

.PHONY: build
build: fmt vet
	go build \
	-ldflags="w -s -X github.com/kubev2v/forklift/cmd/vsphere-xcopy-volume-populator/internal/populator.VibVersion=$(VIB_VERSION)" \
	-o bin/vsphere-xcopy-volume-populator

MOCKGEN := $(shell which mockgen 2>/dev/null)
install-mockgen:
	[ -z "$(MOCKGEN)" ] && go install go.uber.org/mock/mockgen@latest || true

# pre-requisits: ensure a PVC exists.
test-copy-using-cli: build
	bin/vsphere-xcopy-volume-populator \
		--source-vmdk="[eco-iscsi-ds2] vm-2/vm-2.vmdk" \
		--owner-name=test-cli \
		--target-namespace=default \
		--storage-vendor-product=ontap \
		--secret-name=populator-secret \
		--owner-uid=test-cli \
		--kubeconfig=$$KUBECONFIG

test-copy-using-cli-3par: build
	bin/vsphere-xcopy-volume-populator \
		--source-vmdk="[eco-iscsi-ds1] vm-6/vm-6.vmdk" \
	    --owner-name=test1\
		--target-namespace=default \
		--storage-vendor-product=primera3par \
		--secret-name=populator-secret \
		--kubeconfig=$$KUBECONFIG

.PHONY: vmkfstools-wrapper
vmkfstools-wrapper:
	$(MAKE) -C vmkfstools-wrapper build
