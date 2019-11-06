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

bin/subctl:  pkg/subctl/operator/crds/crdyamls.go $(shell find pkg/subctl/ -name "*.go")
	$(MAKE) build-subctl

pkg/subctl/operator/crds/crdyamls.go: deploy/*.yaml deploy/crds/submariner.io*_crd.yaml
	$(MAKE) generate-embeddedyamls

.DEFAULT_GOAL := ci

.PHONY: $(TARGETS)

