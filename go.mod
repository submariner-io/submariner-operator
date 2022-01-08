module github.com/submariner-io/submariner-operator

go 1.16

require (
	github.com/AlecAivazis/survey/v2 v2.2.13
	github.com/asaskevich/govalidator v0.0.0-20210307081110-f21760c49a8d // indirect
	github.com/aws/aws-sdk-go v1.40.59
	github.com/coreos/go-semver v0.3.0
	github.com/coreos/prometheus-operator v0.38.1-0.20200424145508-7e176fda06cc
	github.com/go-errors/errors v1.2.0 // indirect
	github.com/go-logr/logr v0.4.0
	github.com/go-openapi/spec v0.20.3 // indirect
	github.com/go-openapi/swag v0.19.15 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/hashicorp/go-version v1.3.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.13
	github.com/mattn/go-runewidth v0.0.12 // indirect
	github.com/mgutz/ansi v0.0.0-20200706080929-d51e80ef957d // indirect
	github.com/mitchellh/go-testing-interface v1.14.1 // indirect
	github.com/olekukonko/tablewriter v0.0.5 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/openshift/api v0.0.0-20200324173355-9b3bdf846ea1
	github.com/openshift/cluster-dns-operator v0.0.0-20200529200012-f9e4dfc90c57
	github.com/operator-framework/operator-lib v0.4.0
	github.com/operator-framework/operator-sdk v0.19.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.11.0
	github.com/rivo/uniseg v0.2.0 // indirect
	github.com/spf13/cobra v1.2.0
	github.com/submariner-io/admiral v0.11.1
	github.com/submariner-io/cloud-prepare v0.11.1
	github.com/submariner-io/lighthouse v0.11.1
	github.com/submariner-io/shipyard v0.11.1
	github.com/submariner-io/submariner v0.11.1
	github.com/submariner-io/submariner-operator/apis v0.0.0-20210817145008-861856b068a1
	github.com/submariner-io/submariner/pkg/apis v0.0.0-20210816153739-c8e6654e3930
	github.com/ulikunitz/xz v0.5.10 // indirect
	github.com/uw-labs/lichen v0.1.4
	github.com/xlab/treeprint v1.1.0 // indirect
	go.starlark.net v0.0.0-20210506034541-84642328b1f0 // indirect
	golang.org/x/oauth2 v0.0.0-20210819190943-2bc19b11175f
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	google.golang.org/api v0.58.0
	gopkg.in/ini.v1 v1.62.0
	k8s.io/api v0.21.0
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.8.0 // indirect
	sigs.k8s.io/controller-runtime v0.8.3
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/kustomize/cmd/config v0.9.11 // indirect
	sigs.k8s.io/kustomize/kustomize/v3 v3.10.0
	sigs.k8s.io/kustomize/kyaml v0.10.19 // indirect
	sigs.k8s.io/mcs-api v0.1.0
	sigs.k8s.io/structured-merge-diff/v4 v4.1.1 // indirect
	sigs.k8s.io/yaml v1.2.0
)

// When changing pins, check the dependabot configuration too
// in .github/dependabot.yml

// Pinned to kubernetes-1.19.10
replace (
	k8s.io/api => k8s.io/api v0.19.10
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.19.10
	k8s.io/apimachinery => k8s.io/apimachinery v0.19.10
	k8s.io/client-go => k8s.io/client-go v0.19.10
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.19.10
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.7.0
)

// Local packages
replace github.com/submariner-io/submariner-operator/apis => ./apis
