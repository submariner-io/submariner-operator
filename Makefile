ifneq (,$(DAPPER_HOST_ARCH))

# Running in Dapper

include $(SHIPYARD_DIR)/Makefile.inc

VERSION := $(shell . scripts/lib/version; echo $$VERSION)

TARGETS := $(shell ls -p scripts | grep -v -e /)
CLUSTER_SETTINGS_FLAG = --cluster_settings $(DAPPER_SOURCE)/scripts/kind-e2e/cluster_settings
override CLUSTERS_ARGS += $(CLUSTER_SETTINGS_FLAG)
override DEPLOY_ARGS += $(CLUSTER_SETTINGS_FLAG) --deploytool_submariner_args '--cable-driver strongswan --operator-image localhost:5000/submariner-operator:local'
export DEPLOY_ARGS

# Process extra flags from the `using=a,b,c` optional flag

ifneq (,$(filter lighthouse,$(_using)))
override DEPLOY_ARGS += --deploytool_broker_args '--service-discovery'
endif

# Targets to make

clusters: build-all

deploy: clusters preload_images

e2e: deploy
	scripts/kind-e2e/e2e.sh

$(TARGETS): vendor/modules.txt
	./scripts/$@

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
