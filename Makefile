BASE_BRANCH ?= devel
# Denotes the default operator image version, exposed as a variable for the automated release
DEFAULT_IMAGE_VERSION ?= $(BASE_BRANCH)
export BASE_BRANCH
export DEFAULT_IMAGE_VERSION

# Define LOCAL_BUILD to build directly on the host and not inside a Dapper container
ifdef LOCAL_BUILD
DAPPER_HOST_ARCH ?= $(shell go env GOHOSTARCH)
SHIPYARD_DIR ?= ../shipyard
SCRIPTS_DIR ?= $(SHIPYARD_DIR)/scripts/shared

export DAPPER_HOST_ARCH
export SHIPYARD_DIR
export SCRIPTS_DIR
endif

ifneq (,$(DAPPER_HOST_ARCH))

OPERATOR_SDK_VERSION := 1.0.1
OPERATOR_SDK := $(CURDIR)/bin/operator-sdk

KUSTOMIZE_VERSION := 3.10.0
KUSTOMIZE := $(CURDIR)/bin/kustomize

CONTROLLER_GEN := $(CURDIR)/bin/controller-gen

# Running in Dapper

# Semantic versioning regex
PATTERN := ^([0-9]|[1-9][0-9]*)\.([0-9]|[1-9][0-9]*)\.([0-9]|[1-9][0-9]*)$
# Test if VERSION matches the semantic versioning rule
IS_SEMANTIC_VERSION = $(shell [[ $(or $(BUNDLE_VERSION),$(VERSION),'undefined') =~ $(PATTERN) ]] && echo true || echo false)

gotodockerarch = $(patsubst arm,arm/v7,$(1))
dockertogoarch = $(patsubst arm/v7,arm,$(1))

PLATFORMS ?= linux/amd64,linux/arm64
IMAGES = submariner-operator submariner-operator-index
PRELOAD_IMAGES := $(IMAGES) submariner-gateway submariner-globalnet submariner-route-agent lighthouse-agent lighthouse-coredns
MULTIARCH_IMAGES := submariner-operator
undefine SKIP
undefine FOCUS
undefine E2E_TESTDIR

ifneq (,$(filter ovn,$(USING)))
SETTINGS = $(DAPPER_SOURCE)/.shipyard.e2e.ovn.yml
else
SETTINGS = $(DAPPER_SOURCE)/.shipyard.e2e.yml
endif

include $(SHIPYARD_DIR)/Makefile.inc

override E2E_ARGS += cluster1 cluster2
export DEPLOY_ARGS
override UNIT_TEST_ARGS += test internal/env
override VALIDATE_ARGS += --skip-dirs pkg/client

GO ?= go
GOARCH = $(shell $(GO) env GOARCH)
GOEXE = $(shell $(GO) env GOEXE)
GOOS = $(shell $(GO) env GOOS)

# Options for 'submariner-operator-bundle' image
ifeq ($(IS_SEMANTIC_VERSION),true)
BUNDLE_VERSION := $(VERSION)
else
BUNDLE_VERSION := $(shell (git describe --abbrev=0 --tags --match=v[0-9]*\.[0-9]*\.[0-9]* 2>/dev/null || echo v9.9.9) \
| cut -d'-' -f1 | cut -c2-)
endif
FROM_VERSION ?= $(shell (git tag -l --sort=-v:refname v[0-9]*\.[0-9]*\.[0-9]* | awk '/^$(BUNDLE_VERSION)$$/ { seen = 1; next } seen { print; exit } END { exit !seen }' || echo v0.0.0) \
          | head -n1 | cut -d'-' -f1 | cut -c2-)
SHORT_VERSION := $(shell echo ${BUNDLE_VERSION} | cut -d'.' -f1,2)
CHANNEL ?= alpha-$(SHORT_VERSION)
CHANNELS ?= $(CHANNEL)
DEFAULT_CHANNEL ?= $(CHANNEL)
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Options for 'packagemanifests'
IS_CHANNEL_DEFAULT ?= 1
ifneq ($(origin FROM_VERSION), undefined)
ifneq ($(FROM_VERSION), 0.0.0)
PKG_FROM_VERSION := --from-version=$(FROM_VERSION)
REPLACES_OP := add
else
REPLACES_OP := remove
endif
endif
ifneq ($(origin CHANNEL), undefined)
PKG_CHANNELS := --channel=$(CHANNEL)
endif
ifeq ($(IS_CHANNEL_DEFAULT), 1)
PKG_IS_DEFAULT_CHANNEL := --default-channel
endif
PKG_MAN_OPTS ?= $(PKG_FROM_VERSION) $(PKG_CHANNELS) $(PKG_IS_DEFAULT_CHANNEL)

# Set the kustomize base path
ifeq ($(IS_OCP), true)
KUSTOMIZE_BASE_PATH := $(CURDIR)/config/openshift
else
KUSTOMIZE_BASE_PATH := $(CURDIR)/config/manifests
endif

# Image URL to use all building/pushing image targets
REPO ?= quay.io/submariner
IMG ?= $(REPO)/submariner-operator:$(VERSION)
# Produce v1 CRDs, requiring Kubernetes 1.16 or later
CRD_OPTIONS ?= "crd:crdVersions=v1,trivialVersions=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell $(GO) env GOBIN))
GOBIN=$(shell $(GO) env GOPATH)/bin
else
GOBIN=$(shell $(GO) env GOBIN)
endif

# Ensure we prefer binaries we build
export PATH := $(CURDIR)/bin:$(PATH)

# Targets to make

e2e: $(VENDOR_MODULES)
	scripts/test/e2e.sh $(E2E_ARGS)

# [system-test] runs system level tests that validate the operator is properly deployed
system-test:
	scripts/test/system.sh

clean:
	rm -f bin/submariner-operator

licensecheck: BUILD_ARGS=--noupx
licensecheck: build bin/linux/amd64/submariner-operator | bin/lichen
	bin/lichen -c .lichen.yaml bin/linux/amd64/submariner-operator

bin/lichen: $(VENDOR_MODULES)
	mkdir -p $(@D)
	$(GO) build -o $@ github.com/uw-labs/lichen

# Generate deep-copy code
CONTROLLER_DEEPCOPY := api/submariner/v1alpha1/zz_generated.deepcopy.go
$(CONTROLLER_DEEPCOPY): $(VENDOR_MODULES) | $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) object:headerFile="$(CURDIR)/hack/boilerplate.go.txt,year=$(shell date +"%Y")" paths="./..."

# Generate embedded YAMLs
EMBEDDED_YAMLS := pkg/embeddedyamls/yamls.go
$(EMBEDDED_YAMLS): pkg/embeddedyamls/generators/yamls2go.go deploy/crds/submariner.io_servicediscoveries.yaml deploy/crds/submariner.io_brokers.yaml deploy/crds/submariner.io_submariners.yaml deploy/submariner/crds/submariner.io_clusters.yaml deploy/submariner/crds/submariner.io_endpoints.yaml deploy/submariner/crds/submariner.io_gateways.yaml $(shell find deploy/ -name "*.yaml") $(shell find config/rbac/ -name "*.yaml") $(VENDOR_MODULES) $(CONTROLLER_DEEPCOPY)
	$(GO) generate pkg/embeddedyamls/generate.go

bin/%/submariner-operator: $(VENDOR_MODULES) main.go $(EMBEDDED_YAMLS)
	GOARCH=$(call dockertogoarch,$(patsubst bin/linux/%/,%,$(dir $@))) ${SCRIPTS_DIR}/compile.sh \
	--ldflags "-X=github.com/submariner-io/submariner-operator/pkg/version.Version=$(VERSION)" \
	$@ . $(BUILD_ARGS)

ci: $(EMBEDDED_YAMLS) golangci-lint markdownlint unit build images

# Operator CRDs
$(CONTROLLER_GEN): $(VENDOR_MODULES)
	mkdir -p $(@D)
	$(GO) build -o $@ sigs.k8s.io/controller-tools/cmd/controller-gen

deploy/crds/submariner.io_servicediscoveries.yaml: ./api/submariner/v1alpha1/servicediscovery_types.go $(VENDOR_MODULES) | $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=deploy/crds
	test -f $@

deploy/crds/submariner.io_brokers.yaml deploy/crds/submariner.io_submariners.yaml: ./api/submariner/v1alpha1/submariner_types.go $(VENDOR_MODULES) | $(CONTROLLER_GEN)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=deploy/crds
	test -f $@

# Submariner CRDs
deploy/submariner/crds/submariner.io_clusters.yaml deploy/submariner/crds/submariner.io_endpoints.yaml deploy/submariner/crds/submariner.io_gateways.yaml: $(VENDOR_MODULES) | $(CONTROLLER_GEN)
	cd vendor/github.com/submariner-io/submariner && $(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=../../../../deploy/submariner/crds
	test -f $@

# Generate manifests e.g. CRD, RBAC etc
manifests: $(CONTROLLER_DEEPCOPY) $(CONTROLLER_GEN) $(VENDOR_MODULES)
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# test if VERSION matches the semantic versioning rule
is-semantic-version:
    ifneq ($(IS_SEMANTIC_VERSION),true)
	    $(error 'ERROR: VERSION "$(BUNDLE_VERSION)" does not match the format required by operator-sdk.')
    endif

# TODO: a workaround until this issue will be fixed https://github.com/kubernetes-sigs/kustomize/issues/4008
$(KUSTOMIZE):
	mkdir -p $(@D)
	#GOBIN=$(CURDIR)/bin GO111MODULE=on $(GO) get sigs.k8s.io/kustomize/kustomize/v3
	scripts/kustomize/install_kustomize.sh $(KUSTOMIZE_VERSION) $(CURDIR)/bin

# Generate kustomization.yaml for bundle
kustomization: $(OPERATOR_SDK) $(KUSTOMIZE) is-semantic-version manifests
	$(OPERATOR_SDK) generate kustomize manifests -q
	(cd config/manifests && $(KUSTOMIZE) edit set image controller=$(IMG) && \
	 $(KUSTOMIZE) edit set image repo=$(REPO))
	sed -e 's/$${VERSION}/$(BUNDLE_VERSION)/g' config/bundle/kustomization.template.yaml > config/bundle/kustomization.yaml
	sed -e 's/$${REPLACES_OP}/$(REPLACES_OP)/g' -e 's/$${FROM_VERSION}/$(FROM_VERSION)/g' \
		config/bundle/patches/submariner.csv.template.yaml > config/bundle/patches/submariner.csv.config.yaml
	(cd config/bundle && \
	$(KUSTOMIZE) edit add annotation createdAt:"$(shell date "+%Y-%m-%d %T")" -f)

# Generate bundle manifests and metadata, then validate generated files
bundle: $(KUSTOMIZE) $(OPERATOR_SDK) kustomization
	($(KUSTOMIZE) build $(KUSTOMIZE_BASE_PATH) \
	| $(OPERATOR_SDK) generate bundle -q --overwrite --version $(BUNDLE_VERSION) $(BUNDLE_METADATA_OPTS))
	(cd config/bundle && $(KUSTOMIZE) edit add resource ../../bundle/manifests/submariner.clusterserviceversion.yaml)
	$(KUSTOMIZE) build config/bundle/ --load_restrictor=LoadRestrictionsNone --output bundle/manifests/submariner.clusterserviceversion.yaml
	sed -i -e 's/$$(SHORT_VERSION)/$(SHORT_VERSION)/g' bundle/manifests/submariner.clusterserviceversion.yaml
	sed -i -e 's/$$(VERSION)/$(VERSION)/g' bundle/manifests/submariner.clusterserviceversion.yaml
	$(OPERATOR_SDK) bundle validate ./bundle

# Generate package manifests
packagemanifests: $(OPERATOR_SDK) $(KUSTOMIZE) kustomization
	($(KUSTOMIZE) build $(KUSTOMIZE_BASE_PATH) \
	| $(OPERATOR_SDK) generate packagemanifests -q --version $(BUNDLE_VERSION) $(PKG_MAN_OPTS))
	(cd config/bundle && $(KUSTOMIZE) edit add resource ../../packagemanifests/$(BUNDLE_VERSION)/submariner.clusterserviceversion.yaml)
	$(KUSTOMIZE) build config/bundle/ --load_restrictor=LoadRestrictionsNone --output packagemanifests/$(BUNDLE_VERSION)/submariner.clusterserviceversion.yaml
	sed -i -e 's/$$(SHORT_VERSION)/$(SHORT_VERSION)/g' packagemanifests/$(BUNDLE_VERSION)/submariner.clusterserviceversion.yaml
	sed -i -e 's/$$(VERSION)/$(VERSION)/g' packagemanifests/$(BUNDLE_VERSION)/submariner.clusterserviceversion.yaml
	mv packagemanifests/$(BUNDLE_VERSION)/submariner.clusterserviceversion.yaml packagemanifests/$(BUNDLE_VERSION)/submariner.v$(BUNDLE_VERSION).clusterserviceversion.yaml

# Statically validate the operator bundle using Scorecard.
scorecard: bundle olm clusters
	timeout 60 bash -c "until KUBECONFIG=$(DAPPER_OUTPUT)/kubeconfigs/kind-config-cluster1 \
	$(OPERATOR_SDK) olm status > /dev/null; do sleep 10; done"
	$(OPERATOR_SDK) scorecard --kubeconfig=$(DAPPER_OUTPUT)/kubeconfigs/kind-config-cluster1 -o text ./bundle

# Create the clusters with olm
olm:
	$(eval override CLUSTERS_ARGS += --olm)

golangci-lint: $(EMBEDDED_YAMLS)

unit: $(EMBEDDED_YAMLS)

# Operator SDK
# On version bumps, the checksum will need to be updated manually.
# If necessary, the verification *keys* can be updated as follows:
# * update scripts/operator-sdk-signing-key.asc, import the relevant key,
#   and export it with
#     gpg --armor --export-options export-minimal --export \
#     ${fingerprint} >> scripts/operator-sdk-signing-key.asc
#   (replacing ${fingerprint} with the full fingerprint);
# * to update scripts/operator-sdk-signing-keyring.gpg, run
#     gpg --no-options -q --batch --no-default-keyring \
#     --output scripts/operator-sdk-signing-keyring.gpg \
#     --dearmor scripts/operator-sdk-signing-key.asc
$(OPERATOR_SDK):
	curl -Lo $@ "https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk-v${OPERATOR_SDK_VERSION}-x86_64-linux-gnu"
	curl -Lo $@.asc "https://github.com/operator-framework/operator-sdk/releases/download/v${OPERATOR_SDK_VERSION}/operator-sdk-v${OPERATOR_SDK_VERSION}-x86_64-linux-gnu.asc"
	gpgv --keyring scripts/operator-sdk-signing-keyring.gpg $@.asc $@
	sha256sum -c scripts/operator-sdk.sha256
	chmod a+x $@

.PHONY: build ci clean bundle packagemanifests kustomization is-semantic-version olm scorecard system-test

else

# Not running in Dapper

Makefile.dapper:
	@echo Downloading $@
	@curl -sfLO https://raw.githubusercontent.com/submariner-io/shipyard/$(BASE_BRANCH)/$@

include Makefile.dapper

.PHONY: deploy bundle packagemanifests kustomization is-semantic-version licensecheck

endif

# Disable rebuilding Makefile
Makefile Makefile.inc: ;
