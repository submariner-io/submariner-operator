BASE_BRANCH ?= devel
# Denotes the default operator image version, exposed as a variable for the automated release
DEFAULT_IMAGE_VERSION ?= $(BASE_BRANCH)
export BASE_BRANCH
export DEFAULT_IMAGE_VERSION

ifneq (,$(DAPPER_HOST_ARCH))

OPERATOR_SDK_VERSION := 1.0.1
OPERATOR_SDK := $(CURDIR)/bin/operator-sdk

# Running in Dapper

IMAGES=submariner-operator
PRELOAD_IMAGES := $(IMAGES) submariner-gateway submariner-route-agent lighthouse-agent lighthouse-coredns

include $(SHIPYARD_DIR)/Makefile.inc

CROSS_TARGETS := linux-amd64 linux-arm64 linux-arm linux-s390x linux-ppc64le windows-amd64.exe darwin-amd64
BINARIES := bin/subctl
CROSS_BINARIES := $(foreach cross,$(CROSS_TARGETS),$(patsubst %,bin/subctl-$(VERSION)-%,$(cross)))
CROSS_TARBALLS := $(foreach cross,$(CROSS_TARGETS),$(patsubst %,dist/subctl-$(VERSION)-%.tar.xz,$(cross)))

ifneq (,$(filter ovn,$(_using)))
CLUSTER_SETTINGS_FLAG = --cluster_settings $(DAPPER_SOURCE)/scripts/kind-e2e/cluster_settings.ovn
else
CLUSTER_SETTINGS_FLAG = --cluster_settings $(DAPPER_SOURCE)/scripts/kind-e2e/cluster_settings
endif

override CLUSTERS_ARGS += $(CLUSTER_SETTINGS_FLAG)
override DEPLOY_ARGS += $(CLUSTER_SETTINGS_FLAG)
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

# Options for 'submariner-operator-bundle' image
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Options for 'packagemanifests'
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
REPO ?= quay.io/submariner
IMG ?= $(REPO)/submariner-operator:$(VERSION)
# Produce v1 CRDs, requiring Kubernetes 1.16 or later
CRD_OPTIONS ?= "crd:crdVersions=v1,trivialVersions=false"
# Semantic versioning regex
PATTERN := ^([0-9]|[1-9][0-9]*)\.([0-9]|[1-9][0-9]*)\.([0-9]|[1-9][0-9]*)$

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Ensure we prefer binaries we build
export PATH := $(CURDIR)/bin:$(PATH)

# Targets to make

images: build

# Build subctl before deploying to ensure we use that
# (with the PATH set above)
deploy: bin/subctl

e2e: deploy
	scripts/kind-e2e/e2e.sh

clean:
	rm -f $(BINARIES) $(CROSS_BINARIES) $(CROSS_TARBALLS)

build: $(BINARIES)

build-cross: $(CROSS_TARBALLS)

licensecheck: BUILD_ARGS=--noupx
licensecheck: build bin/lichen bin/submariner-operator
	bin/lichen -c .lichen.yaml $(BINARIES) bin/submariner-operator

bin/lichen: vendor/modules.txt
	mkdir -p $(@D)
	go build -o $@ github.com/uw-labs/lichen

package/Dockerfile.submariner-operator: bin/submariner-operator

bin/submariner-operator: vendor/modules.txt main.go generate-embeddedyamls
	${SCRIPTS_DIR}/compile.sh \
	--ldflags "-X=github.com/submariner-io/submariner-operator/pkg/version.Version=$(VERSION)" \
	$@ main.go $(BUILD_ARGS)

bin/subctl: bin/subctl-$(VERSION)-$(GOOS)-$(GOARCH)$(GOEXE)
	ln -sf $(<F) $@

dist/subctl-%.tar.xz: bin/subctl-%
	mkdir -p dist
	tar -cJf $@ --transform "s/^bin/subctl-$(VERSION)/" $<

# Versions may include hyphens so it's easier to use $(VERSION) than to extract them from the target
bin/subctl-%: generate-embeddedyamls $(shell find pkg/subctl/ -name "*.go") vendor/modules.txt
	mkdir -p $(@D)
	target=$@; \
	target=$${target%.exe}; \
	components=($$(echo $${target//-/ })); \
	GOOS=$${components[-2]}; \
	GOARCH=$${components[-1]}; \
	export GOARCH GOOS; \
	$(SCRIPTS_DIR)/compile.sh \
		--ldflags "-X github.com/submariner-io/submariner-operator/pkg/version.Version=$(VERSION) \
			   -X=github.com/submariner-io/submariner-operator/pkg/versions.DefaultSubmarinerOperatorVersion=$${DEFAULT_IMAGE_VERSION#v}" \
		--noupx $@ ./pkg/subctl/main.go $(BUILD_ARGS)

ci: generate-embeddedyamls golangci-lint markdownlint unit build images

generate-embeddedyamls: generate pkg/subctl/operator/common/embeddedyamls/yamls.go

pkg/subctl/operator/common/embeddedyamls/yamls.go: pkg/subctl/operator/common/embeddedyamls/generators/yamls2go.go deploy/crds/submariner.io_servicediscoveries.yaml deploy/crds/submariner.io_brokers.yaml deploy/crds/submariner.io_submariners.yaml deploy/submariner/crds/submariner.io_clusters.yaml deploy/submariner/crds/submariner.io_endpoints.yaml deploy/submariner/crds/submariner.io_gateways.yaml $(shell find deploy/ -name "*.yaml") $(shell find config/rbac/ -name "*.yaml") vendor/modules.txt
	go generate pkg/subctl/operator/common/embeddedyamls/generate.go

# Operator CRDs
CONTROLLER_GEN := $(CURDIR)/bin/controller-gen
$(CONTROLLER_GEN): vendor/modules.txt
	mkdir -p $(@D)
	go build -o $@ sigs.k8s.io/controller-tools/cmd/controller-gen

deploy/crds/submariner.io_servicediscoveries.yaml: $(CONTROLLER_GEN) ./apis/submariner/v1alpha1/servicediscovery_types.go vendor/modules.txt
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=deploy/crds

deploy/crds/submariner.io_brokers.yaml deploy/crds/submariner.io_submariners.yaml: $(CONTROLLER_GEN) ./apis/submariner/v1alpha1/submariner_types.go vendor/modules.txt
	$(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=deploy/crds

# Submariner CRDs
deploy/submariner/crds/submariner.io_clusters.yaml deploy/submariner/crds/submariner.io_endpoints.yaml deploy/submariner/crds/submariner.io_gateways.yaml: $(CONTROLLER_GEN) vendor/modules.txt
	cd vendor/github.com/submariner-io/submariner && $(CONTROLLER_GEN) $(CRD_OPTIONS) paths="./..." output:crd:artifacts:config=../../../../deploy/submariner/crds

# Generate the clientset for the Submariner APIs
# It needs to be run when the Submariner APIs change
generate-clientset: vendor/modules.txt
	git clone https://github.com/kubernetes/code-generator -b kubernetes-1.19.10 $${GOPATH}/src/k8s.io/code-generator
	cd $${GOPATH}/src/k8s.io/code-generator && go mod vendor
	GO111MODULE=on $${GOPATH}/src/k8s.io/code-generator/generate-groups.sh \
		client,deepcopy \
		github.com/submariner-io/submariner-operator/pkg/client \
		github.com/submariner-io/submariner-operator/apis \
		submariner:v1alpha1

# Generate code
generate: $(CONTROLLER_GEN) vendor/modules.txt
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt,year=$(shell date +"%Y")" paths="./..."

# Generate manifests e.g. CRD, RBAC etc
manifests: generate $(CONTROLLER_GEN) vendor/modules.txt
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

# test if VERSION matches the semantic versioning rule
is-semantic-version:
	[[ $(VERSION) =~ $(PATTERN) ]] || \
	(printf '\nerror: VERSION does not match the format required by operator-sdk.\n\n' && exit 1)

# Generate kustomization.yaml for bundle
KUSTOMIZE := $(CURDIR)/bin/kustomize
$(KUSTOMIZE): vendor/modules.txt
	mkdir -p $(@D)
	go build -o $@ sigs.k8s.io/kustomize/kustomize/v3

kustomization: $(OPERATOR_SDK) $(KUSTOMIZE) is-semantic-version manifests
	$(OPERATOR_SDK) generate kustomize manifests -q && \
	(cd config/manifests && $(KUSTOMIZE) edit set image controller=$(IMG) && \
	 $(KUSTOMIZE) edit set image repo=$(REPO)) && \
	(cd config/bundle && \
	sed -e 's/$${VERSION}/$(VERSION)/g' kustomization.template.yaml > kustomization.yaml && \
	$(KUSTOMIZE) edit add annotation createdAt:"$(shell date "+%Y-%m-%d %T")" -f)

# Generate bundle manifests and metadata, then validate generated files
bundle: $(KUSTOMIZE) $(OPERATOR_SDK) kustomization
	($(KUSTOMIZE) build config/manifests \
	| $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)) && \
	(cd config/bundle && $(KUSTOMIZE) edit add resource ../../bundle/manifests/submariner.clusterserviceversion.yaml && cd ../../) && \
	$(KUSTOMIZE) build config/bundle/ --load_restrictor=LoadRestrictionsNone --output bundle/manifests/submariner.clusterserviceversion.yaml && \
	$(OPERATOR_SDK) bundle validate ./bundle

# Generate package manifests
packagemanifests: $(OPERATOR_SDK) $(KUSTOMIZE) kustomization
	($(KUSTOMIZE) build config/manifests \
	| $(OPERATOR_SDK) generate packagemanifests -q --version $(VERSION) $(PKG_MAN_OPTS)) && \
	(cd config/bundle && $(KUSTOMIZE) edit add resource ../../packagemanifests/$(VERSION)/submariner.clusterserviceversion.yaml && cd ../../) && \
	$(KUSTOMIZE) build config/bundle/ --load_restrictor=LoadRestrictionsNone --output packagemanifests/$(VERSION)/submariner.clusterserviceversion.yaml && \
	mv packagemanifests/$(VERSION)/submariner.clusterserviceversion.yaml packagemanifests/$(VERSION)/submariner.v$(VERSION).clusterserviceversion.yaml

golangci-lint: generate-embeddedyamls

unit: generate-embeddedyamls

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

.PHONY: build ci clean generate-clientset generate-embeddedyamls bundle packagemanifests kustomization is-semantic-version

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
