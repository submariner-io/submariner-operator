status ?= onetime

TARGETS := $(shell ls scripts)

.dapper:
	@echo Downloading dapper
	@curl -sL https://releases.rancher.com/dapper/latest/dapper-`uname -s`-`uname -m` > .dapper.tmp
	@@chmod +x .dapper.tmp
	@./.dapper.tmp -v
	@mv .dapper.tmp .dapper

$(TARGETS): .dapper
	./.dapper -m bind $@ $(status)

shell: .dapper
	./.dapper -s -m bind

bin/subctl: pkg/subctl/operator/install/embeddedyamls/yamls.go pkg/subctl/lighthouse/install/embeddedyamls/yamls.go $(shell find pkg/subctl/ -name "*.go")
	$(MAKE) build-subctl

pkg/subctl/operator/install/embeddedyamls/yamls.go: deploy/*.yaml deploy/crds/submariner.io*_crd.yaml
	$(MAKE) generate-embeddedyamls

pkg/subctl/lighthouse/install/embeddedyamls/yamls.go: deploy/lighthouse/crds/*.yaml
	$(MAKE) generate-lighthouse-embeddedyamls

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)
