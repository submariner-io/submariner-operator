status ?= onetime
lighthouse ?= false
globalnet ?= false
version ?= 1.14.6

TARGETS := $(shell ls scripts)
SCRIPTS_DIR ?= /opt/shipyard/scripts

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

cleanup: .dapper
	./.dapper -m bind $(SCRIPTS_DIR)/cleanup.sh

clusters: build
	./.dapper -m bind $(SCRIPTS_DIR)/clusters.sh --k8s_version $(version) --globalnet $(globalnet)

e2e: clusters
	./.dapper -m bind scripts/kind-e2e/e2e.sh --status $(status) --lighthouse $(lighthouse) --globalnet $(globalnet)

$(TARGETS): .dapper vendor/modules.txt
	./.dapper -m bind $@

shell: .dapper
	./.dapper -s -m bind

build-subctl: pkg/subctl/operator/common/embeddedyamls/yamls.go $(shell find pkg/subctl/ -name "*.go")

bin/subctl: build-subctl

pkg/subctl/operator/common/embeddedyamls/yamls.go: pkg/subctl/operator/common/embeddedyamls/generators/yamls2go.go $(shell find deploy/ -name "*.yaml")
	$(MAKE) generate-embeddedyamls

vendor/modules.txt: .dapper go.mod
	./.dapper -m bind vendor

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)
