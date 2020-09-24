ifneq (,$(DAPPER_HOST_ARCH))

# Running in Dapper

include $(SHIPYARD_DIR)/Makefile.inc

override CALCULATED_VERSION := $(shell . ${SCRIPTS_DIR}/lib/version; echo $$VERSION)
VERSION ?= $(CALCULATED_VERSION)
DEV_VERSION := $(shell . ${SCRIPTS_DIR}/lib/version; echo $$DEV_VERSION)

export VERSION DEV_VERSION

CROSS_TARGETS := linux-amd64 linux-arm64 windows-amd64.exe darwin-amd64
BINARIES := bin/subctl
CROSS_BINARIES := $(foreach cross,$(CROSS_TARGETS),$(patsubst %,bin/subctl-$(VERSION)-%,$(cross)))
CROSS_TARBALLS := $(foreach cross,$(CROSS_TARGETS),$(patsubst %,dist/subctl-$(VERSION)-%.tar.xz,$(cross)))
CLUSTER_SETTINGS_FLAG = --cluster_settings $(DAPPER_SOURCE)/scripts/kind-e2e/cluster_settings
override CLUSTERS_ARGS += $(CLUSTER_SETTINGS_FLAG)
override DEPLOY_ARGS += $(CLUSTER_SETTINGS_FLAG) --deploytool_submariner_args '--cable-driver strongswan'
export DEPLOY_ARGS
override UNIT_TEST_ARGS += cmd pkg/internal
override VALIDATE_ARGS += --skip-dirs pkg/client

# Process extra flags from the `using=a,b,c` optional flag

ifneq (,$(filter lighthouse,$(_using)))
override DEPLOY_ARGS += --deploytool_broker_args '--service-discovery'
endif

GOARCH = $(shell go env GOARCH)
GOEXE = $(shell go env GOEXE)
GOOS = $(shell go env GOOS)

# Default bundle image tag
BUNDLE_IMG ?= quay.io/submariner/submariner-operator-bundle:$(VERSION)
# Options for 'bundle-build'
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Options for "packagemanifests".
ifneq ($(origin FROM_VERSION), undefined)
PKG_FROM_VERSION := --from-version=$(FROM_VERSION)
endif
ifneq ($(origin CHANNEL), undefined)
PKG_CHANNELS := --channel=$(CHANNEL)
endif
ifeq ($(IS_CHANNEL_DEFAULT), 1)
PKG_IS_DEFAULT_CHANNEL := --default-channel
endif
PKG_MAN_OPTS ?= $(PKG_FROM_VERSION) $(PKG_CHANNELS) $(PKG_IS_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= quay.io/submariner/submariner-operator:$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Targets to make

clusters: build

e2e: deploy
	scripts/kind-e2e/e2e.sh

test: unit-test

clean:
	rm -f $(BINARIES) $(CROSS_BINARIES) $(CROSS_TARBALLS)

build: operator-image $(BINARIES)

build-cross: $(CROSS_TARBALLS)

operator-image: vendor/modules.txt pkg/subctl/operator/common/embeddedyamls/yamls.go manager

bin/subctl: bin/subctl-$(VERSION)-$(GOOS)-$(GOARCH)$(GOEXE)
	ln -sf $(<F) $@

dist/subctl-%.tar.xz: bin/subctl-%
	mkdir -p dist
	tar -cJf $@ --transform "s/^bin/subctl-$(VERSION)/" $<

# Versions may include hyphens so it's easier to use $(VERSION) than to extract them from the target
bin/subctl-%: pkg/subctl/operator/common/embeddedyamls/yamls.go $(shell find pkg/subctl/ -name "*.go") vendor/modules.txt
	mkdir -p bin
# We want the calculated version here, not the potentially-overridden target version
	target=$@; \
	target=$${target%.exe}; \
	components=($$(echo $${target//-/ })); \
	GOOS=$${components[-2]}; \
	GOARCH=$${components[-1]}; \
	export GOARCH GOOS; \
	$(SCRIPTS_DIR)/compile.sh \
		--ldflags "-X github.com/submariner-io/submariner-operator/pkg/version.Version=$(CALCULATED_VERSION)" \
		--noupx $@ ./pkg/subctl/main.go

ci: generate-embeddedyamls validate test build

generate-embeddedyamls: pkg/subctl/operator/common/embeddedyamls/yamls.go

pkg/subctl/operator/common/embeddedyamls/yamls.go: pkg/subctl/operator/common/embeddedyamls/generators/yamls2go.go $(shell find config/ -name "*.yaml") vendor/modules.txt
	go generate pkg/subctl/operator/common/embeddedyamls/generate.go

# generate-clientset generates the clientset for the Submariner APIs
# It needs to be run when the Submariner APIs change
generate-clientset:
	git clone https://github.com/kubernetes/code-generator -b kubernetes-1.17.0 $${GOPATH}/src/k8s.io/code-generator
	cd $${GOPATH}/src/k8s.io/code-generator && go mod vendor
	GO111MODULE=on $${GOPATH}/src/k8s.io/code-generator/generate-groups.sh \
		client,deepcopy \
		github.com/submariner-io/submariner-operator/pkg/client \
		github.com/submariner-io/submariner-operator/apis \
		submariner:v1alpha1

preload-images:
	source $(SCRIPTS_DIR)/lib/debug_functions; \
	source $(SCRIPTS_DIR)/lib/deploy_funcs; \
	source $(SCRIPTS_DIR)/lib/version; \
	set -e; \
	for image in submariner submariner-route-agent submariner-operator lighthouse-agent submariner-globalnet lighthouse-coredns; do \
		import_image quay.io/submariner/$${image}; \
	done

validate: pkg/subctl/operator/common/embeddedyamls/yamls.go

all: manager

# Run tests
#test: generate fmt vet manifests
	#go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
# We check BUILD_ARGS since that's what the compile script uses
ifeq (--debug,$(findstring --debug,$(BUILD_ARGS)))
	go build -o bin/submariner-operator main.go
else
	go build -ldflags -s -w -o bin/submariner-operator main.go
endif

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	$(KUSTOMIZE) build config/crd | sed 's/$$(VERSION)/$(VERSION)/g' | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	$(KUSTOMIZE) build config/crd | sed 's/$$(VERSION)/$(VERSION)/g' | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests clusters preload-images
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | sed 's/$$(VERSION)/$(VERSION)/g' | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests:
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate:
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
docker-build: test
ifeq (--debug,$(findstring --debug,$(BUILD_ARGS)))
	docker build --build-arg DEBUG=true . -t ${IMG}
else
	docker build --build-arg DEBUG=false . -t ${IMG}
endif

# Generate bundle manifests and metadata, then validate generated files.
bundle: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | sed 's/$$(VERSION)/$(VERSION)/g' | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	find ./bundle -type f -print0 | xargs -0 sed -i 's/$$(VERSION)/$(VERSION)/g'
	operator-sdk bundle validate ./bundle

# Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

# Generate package manifests.
packagemanifests: manifests
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMG)
	$(KUSTOMIZE) build config/manifests | sed 's/$$(VERSION)/$(VERSION)/g' | operator-sdk generate packagemanifests -q --version $(VERSION) $(PKG_MAN_OPTS)
	find ./packagemanifests -type f -print0 | xargs -0 sed -i 's/$$(VERSION)/$(VERSION)/g'

.PHONY: test validate build ci clean generate-clientset generate-embeddedyamls operator-image preload-images bundle bundle-build packagemanifests

else

# Not running in Dapper

include Makefile.dapper

endif

# Disable rebuilding Makefile
Makefile Makefile.dapper Makefile.inc: ;
