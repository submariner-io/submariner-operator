module github.com/submariner-io/submariner-operator

go 1.13

require (
	github.com/AlecAivazis/survey/v2 v2.2.7
	github.com/go-logr/logr v0.3.0
	github.com/go-openapi/spec v0.20.0
	github.com/hashicorp/go-version v1.2.0
	github.com/kr/pty v1.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12
	github.com/onsi/ginkgo v1.16.0
	github.com/onsi/gomega v1.11.0
	github.com/openshift/api v0.0.0-20200324173355-9b3bdf846ea1
	github.com/openshift/cluster-dns-operator v0.0.0-20200529200012-f9e4dfc90c57
	github.com/operator-framework/operator-lib v0.4.0
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/spf13/cobra v1.1.1
	github.com/submariner-io/admiral v0.9.0-m2
	github.com/submariner-io/lighthouse v0.9.0-m2
	github.com/submariner-io/shipyard v0.9.0-m2
	github.com/submariner-io/submariner v0.9.0-m2
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200204173128-addea2498afe
	sigs.k8s.io/controller-runtime v0.8.0
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/mcs-api v0.0.0-20200908023942-d26176718973
	sigs.k8s.io/yaml v1.2.0
)

// Pinned to kubernetes-1.17.0
replace (
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.0
	k8s.io/client-go => k8s.io/client-go v0.17.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.5.14
)
