GO ?= go
GOARCH = $(shell $(GO) env GOARCH)
GOEXE = $(shell $(GO) env GOEXE)
GOOS = $(shell $(GO) env GOOS)

DEFAULT_IMAGE_VERSION ?= devel
VERSION ?= devel

LDFLAGS = "-X github.com/submariner-io/submariner-operator/pkg/version.Version=$(VERSION) \
           -X=github.com/submariner-io/submariner-operator/api.DefaultSubmarinerOperatorVersion=$(DEFAULT_IMAGE_VERSION)"

bin/subctl: bin/subctl-$(VERSION)-$(GOOS)-$(GOARCH)$(GOEXE)
	ln -sf $(<F) $@

# Versions may include hyphens so it's easier to use $(VERSION) than to extract them from the target
bin/subctl-%: $(shell find -name "*.go")
	mkdir -p $(@D)
	target=$(VERSION)-$(GOOS)-$(GOARCH)$(GOEXE); \
	target=$${target%.exe}; \
	CGO_ENABLED=0 $(GO) build -trimpath --ldflags $(LDFLAGS) -o $@ main.go
