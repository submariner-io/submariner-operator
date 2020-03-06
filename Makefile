status ?= onetime
version ?= 1.14.2
logging ?= false
kubefed ?= false
deploytool ?= helm
globalnet ?= false
build_debug ?= false
lighthouse ?= false
globalnet ?= false

TARGETS := $(shell ls scripts)

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper vendor/modules.txt
	./.dapper -m bind $@ --status $(status) --k8s_version $(version) --logging $(logging) --kubefed $(kubefed) --deploytool $(deploytool) --globalnet $(globalnet) --build_debug $(build_debug) --lighthouse $(lighthouse)

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
