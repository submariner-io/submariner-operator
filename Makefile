ifneq (,$(DAPPER_HOST_ARCH))

# Running in Dapper

include $(SHIPYARD_DIR)/Makefile.inc

VERSION := $(shell . ${SCRIPTS_DIR}/lib/version; echo $$VERSION)
DEV_VERSION := $(shell . ${SCRIPTS_DIR}/lib/version; echo $$DEV_VERSION)

CROSS_TARGETS := linux-amd64 linux-arm64 windows-amd64.exe darwin-amd64
BINARIES := bin/subctl
CROSS_BINARIES := $(foreach cross,$(CROSS_TARGETS),$(patsubst %,bin/subctl-$(VERSION)-%,$(cross)))
CROSS_TARBALLS := $(foreach cross,$(CROSS_TARGETS),$(patsubst %,dist/subctl-$(VERSION)-%.tar.xz,$(cross)))
TARGETS := $(shell ls -p scripts | grep -v -e /)
CLUSTER_SETTINGS_FLAG = --cluster_settings $(DAPPER_SOURCE)/scripts/kind-e2e/cluster_settings
override CLUSTERS_ARGS += $(CLUSTER_SETTINGS_FLAG)
override DEPLOY_ARGS += $(CLUSTER_SETTINGS_FLAG) --deploytool_submariner_args '--cable-driver strongswan --operator-image localhost:5000/submariner-operator:local'
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

deploy: clusters preload_images

e2e: deploy
	scripts/kind-e2e/e2e.sh

test: unit-test

$(TARGETS): vendor/modules.txt
	./scripts/$@

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
	target=$@; \
	target=$${target%.exe}; \
	components=($$(echo $${target//-/ })); \
	GOOS=$${components[-2]}; \
	GOARCH=$${components[-1]}; \
	export GOARCH GOOS; \
	source $(SCRIPTS_DIR)/lib/version; \
	$(SCRIPTS_DIR)/compile.sh \
		--ldflags "-X github.com/submariner-io/submariner-operator/pkg/version.Version=$${VERSION}" \
		--noupx $@ ./pkg/subctl/main.go

ci: generate-embeddedyamls validate test build

generate-embeddedyamls: pkg/subctl/operator/common/embeddedyamls/yamls.go

pkg/subctl/operator/common/embeddedyamls/yamls.go: pkg/subctl/operator/common/embeddedyamls/generators/yamls2go.go $(shell find deploy/ -name "*.yaml") vendor/modules.txt
	go generate pkg/subctl/operator/common/embeddedyamls/generate.go

.PHONY: $(TARGETS) test validate build ci clean generate-embeddedyamls operator-image

else

# Not running in Dapper

include Makefile.dapper

endif

# Disable rebuilding Makefile
Makefile Makefile.dapper Makefile.inc: ;
