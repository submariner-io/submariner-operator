build_debug ?= false
lighthouse ?= false

ifneq (,$(DAPPER_HOST_ARCH))

# Running in Dapper

include $(SHIPYARD_DIR)/Makefile.inc

TARGETS := $(shell ls -p scripts | grep -v -e /)
CLUSTER_SETTINGS_FLAG = --cluster_settings $(DAPPER_SOURCE)/scripts/kind-e2e/cluster_settings
CLUSTERS_ARGS += $(CLUSTER_SETTINGS_FLAG)
DEPLOY_ARGS += $(CLUSTER_SETTINGS_FLAG) --cable_driver strongswan

clusters: build-all

deploy: clusters preload_images

e2e: deploy
	scripts/kind-e2e/e2e.sh

$(TARGETS): vendor/modules.txt
	./scripts/$@ --build_debug $(build_debug)

build-subctl: pkg/subctl/operator/common/embeddedyamls/yamls.go $(shell find pkg/subctl/ -name "*.go")

bin/subctl: build-subctl

pkg/subctl/operator/common/embeddedyamls/yamls.go: pkg/subctl/operator/common/embeddedyamls/generators/yamls2go.go $(shell find deploy/ -name "*.yaml")
	$(MAKE) generate-embeddedyamls

.PHONY: $(TARGETS)

else

# Not running in Dapper

include Makefile.dapper

endif

# Disable rebuilding Makefile
Makefile Makefile.dapper Makefile.inc: ;
