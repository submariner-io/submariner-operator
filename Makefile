status ?= onetime
lighthouse ?= false

TARGETS := $(shell ls scripts)

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper vendor/modules.txt
	./.dapper -m bind $@ $(status) $(lighthouse)

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
