module github.com/submariner-io/submariner-operator

require (
	github.com/AlecAivazis/survey/v2 v2.1.1
	github.com/creack/pty v1.1.9 // indirect
	github.com/go-logr/logr v0.2.0
	github.com/go-openapi/spec v0.19.9
	github.com/kr/pty v1.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12
	github.com/onsi/ginkgo v1.14.0
	github.com/onsi/gomega v1.10.1
	github.com/openshift/api v0.0.0-20200324173355-9b3bdf846ea1
	github.com/openshift/cluster-dns-operator v0.0.0-20200529200012-f9e4dfc90c57
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.5.1
	github.com/spf13/cobra v0.0.7
	github.com/spf13/pflag v1.0.5
	github.com/submariner-io/lighthouse v0.5.1-0.20200804162804-953592dd51dc
	github.com/submariner-io/shipyard v0.5.0
	github.com/submariner-io/submariner v0.5.1-0.20200805073811-64c7ad477abf
	k8s.io/api v0.18.0-rc.1
	k8s.io/apiextensions-apiserver v0.17.8
	k8s.io/apimachinery v0.18.3
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/kube-openapi v0.0.0-20200204173128-addea2498afe
	sigs.k8s.io/controller-runtime v0.5.2
	sigs.k8s.io/yaml v1.2.0
)

// Pinned to kubernetes-1.17.0
replace (
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.0
	k8s.io/client-go => k8s.io/client-go v0.17.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.5.9
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible

go 1.13
