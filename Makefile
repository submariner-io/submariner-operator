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

# Targets to make

clusters: build

deploy: clusters preload-images

e2e: deploy
	scripts/kind-e2e/e2e.sh

test: unit-test

clean:
	rm -f $(BINARIES) $(CROSS_BINARIES) $(CROSS_TARBALLS)

build: operator-image $(BINARIES)

build-cross: $(CROSS_TARBALLS)

operator-image: vendor/modules.txt
# We check BUILD_ARGS since that's what the compile script uses
ifeq (--debug,$(findstring --debug,$(BUILD_ARGS)))
	operator-sdk build quay.io/submariner/submariner-operator:$(DEV_VERSION) --verbose
else
	operator-sdk build quay.io/submariner/submariner-operator:$(DEV_VERSION) --verbose --go-build-args "-ldflags -s -ldflags -w"
endif

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

pkg/subctl/operator/common/embeddedyamls/yamls.go: pkg/subctl/operator/common/embeddedyamls/generators/yamls2go.go $(shell find deploy/ -name "*.yaml") vendor/modules.txt
	go generate pkg/subctl/operator/common/embeddedyamls/generate.go

# generate-clientset generates the clientset for the Submariner APIs
# It needs to be run when the Submariner APIs change
generate-clientset:
	git clone https://github.com/kubernetes/code-generator -b kubernetes-1.17.0 $${GOPATH}/src/k8s.io/code-generator
	cd $${GOPATH}/src/k8s.io/code-generator && go mod vendor
	GO111MODULE=on $${GOPATH}/src/k8s.io/code-generator/generate-groups.sh \
		client,deepcopy \
		github.com/submariner-io/submariner-operator/pkg/client \
		github.com/submariner-io/submariner-operator/pkg/apis \
		submariner:v1alpha1

# generate-operator-api updates the generated operator code
# It needs to be run when the CRDs or APIs change
generate-operator-api: vendor/modules.txt
	operator-sdk generate k8s
	operator-sdk generate openapi

preload-images:
	source $(SCRIPTS_DIR)/lib/debug_functions; \
	source $(SCRIPTS_DIR)/lib/deploy_funcs; \
	source $(SCRIPTS_DIR)/lib/version; \
	set -e; \
	for image in submariner submariner-route-agent submariner-operator lighthouse-agent submariner-globalnet lighthouse-coredns; do \
		import_image quay.io/submariner/$${image}; \
	done

validate: pkg/subctl/operator/common/embeddedyamls/yamls.go

.PHONY: test validate build ci clean generate-clientset generate-embeddedyamls generate-operator-api operator-image preload-images

else

# Not running in Dapper

include Makefile.dapper

endif

# Disable rebuilding Makefile
Makefile Makefile.dapper Makefile.inc: ;
