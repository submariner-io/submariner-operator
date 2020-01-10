status ?= onetime
lighthouse ?= false

TARGETS := $(shell ls scripts)

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper
	./.dapper -m bind $@ $(status) $(lighthouse)

shell: .dapper
	./.dapper -s -m bind

bin/subctl: pkg/subctl/operator/install/embeddedyamls/yamls.go $(shell find pkg/subctl/ -name "*.go")
	$(MAKE) build-subctl

pkg/subctl/operator/install/embeddedyamls/yamls.go: pkg/subctl/operator/install/embeddedyamls/generators/yamls2go.go $(shell find deploy/ -name "*.yaml")
	$(MAKE) generate-embeddedyamls

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)
