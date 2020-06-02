module github.com/submariner-io/submariner-operator

require (
	cloud.google.com/go v0.47.0 // indirect
	github.com/AlecAivazis/survey/v2 v2.0.7
	github.com/coreos/prometheus-operator v0.31.1 // indirect
	github.com/creack/pty v1.1.9 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/go-openapi/spec v0.19.7
	github.com/grpc-ecosystem/grpc-gateway v1.11.3 // indirect
	github.com/kr/pty v1.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12
	github.com/onsi/ginkgo v1.12.3
	github.com/onsi/gomega v1.10.1
	github.com/operator-framework/operator-sdk v0.10.1-0.20191007233534-070d931e130a
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.2.1 // indirect
	github.com/spf13/cobra v0.0.7
	github.com/spf13/pflag v1.0.5
	github.com/submariner-io/lighthouse v0.3.1-0.20200529100003-0396f214fb58
	github.com/submariner-io/shipyard v0.0.0-20200424120554-752db6dc1c90
	github.com/submariner-io/submariner v0.3.1-0.20200515132708-8214b2793bbd
	go.opencensus.io v0.22.1 // indirect
	go.uber.org/zap v1.12.0 // indirect
	golang.org/x/sync v0.0.0-20190911185100-cd5d95a43a6e // indirect
	google.golang.org/api v0.13.0 // indirect
	google.golang.org/appengine v1.6.5 // indirect
	google.golang.org/genproto v0.0.0-20191028173616-919d9bdd9fe6 // indirect
	google.golang.org/grpc v1.24.0 // indirect
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apiextensions-apiserver v0.0.0-20190918201827-3de75813f604
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v0.4.0
	k8s.io/kube-openapi v0.0.0-20200204173128-addea2498afe
	sigs.k8s.io/controller-runtime v0.3.0
	sigs.k8s.io/yaml v1.2.0
)

// Pinned to kubernetes-1.14.1
replace (
	k8s.io/api => k8s.io/api v0.0.0-20190409021203-6e4e0e4f393b
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20190409022649-727a075fdec8
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190404173353-6a84e37a896d
	k8s.io/client-go => k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20190409023720-1bc0c81fa51d
)

replace (
	github.com/coreos/prometheus-operator => github.com/coreos/prometheus-operator v0.31.1
	// Pinned to v2.10.0 (kubernetes-1.14.1) so https://proxy.golang.org can
	// resolve it correctly.
	github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20190525122359-d20e84d0fb64
)

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v12.2.0+incompatible

go 1.13
