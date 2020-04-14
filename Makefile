build_debug ?= false
lighthouse ?= false

ifneq (,$(DAPPER_HOST_ARCH))

# Running in Dapper

include $(SHIPYARD_DIR)/Makefile.inc

TARGETS := $(shell ls -p scripts | grep -v -e /)
CLUSTERS_ARGS = --cluster_settings scripts/kind-e2e/cluster_settings

clusters: build

e2e: clusters
	scripts/kind-e2e/e2e.sh --status $(status) --lighthouse $(lighthouse) --globalnet $(globalnet)

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
