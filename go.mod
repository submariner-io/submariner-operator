module github.com/submariner-io/submariner-operator

go 1.13

require (
	github.com/AlecAivazis/survey/v2 v2.2.7
	github.com/coreos/go-systemd v0.0.0-20190719114852-fd7a80b32e1f // indirect
	github.com/go-logr/logr v0.2.0
	github.com/go-openapi/spec v0.20.0
	github.com/kr/pty v1.1.8 // indirect
	github.com/mattn/go-isatty v0.0.12
	github.com/onsi/ginkgo v1.15.0
	github.com/onsi/gomega v1.10.5
	github.com/openshift/api v0.0.0-20200324173355-9b3bdf846ea1
	github.com/openshift/cluster-dns-operator v0.0.0-20200529200012-f9e4dfc90c57
	github.com/operator-framework/operator-lib v0.4.0
	github.com/operator-framework/operator-sdk v0.17.0
	github.com/pkg/errors v0.9.1
	github.com/projectcalico/go-json v0.0.0-20161128004156-6219dc7339ba // indirect
	github.com/projectcalico/go-yaml-wrapper v0.0.0-20191112210931-090425220c54 // indirect
	github.com/projectcalico/libcalico-go v0.0.0-00010101000000-000000000000
	github.com/prometheus/client_golang v1.9.0
	github.com/spf13/cobra v1.1.1
	github.com/spf13/pflag v1.0.5
	github.com/submariner-io/admiral v0.9.0-m1.0.20210303181719-5bb1de3368b5
	github.com/submariner-io/lighthouse v0.9.0-m1
	github.com/submariner-io/shipyard v0.9.0-m1
	github.com/submariner-io/submariner v0.9.0-m1.0.20210322110323-300db4f5d90f
	go.etcd.io/etcd v0.5.0-alpha.5.0.20201125193152-8a03d2e9614b // indirect
	go.uber.org/zap v1.15.0 // indirect
	gopkg.in/tchap/go-patricia.v2 v2.2.6 // indirect
	k8s.io/api v0.20.2
	k8s.io/apiextensions-apiserver v0.20.1
	k8s.io/apimachinery v0.20.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/gengo v0.0.0-20200428234225-8167cfdcfc14 // indirect
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.2.0 // indirect
	k8s.io/kube-openapi v0.0.0-20200805222855-6aeccd4b50c6
	sigs.k8s.io/controller-runtime v0.8.0
	sigs.k8s.io/controller-tools v0.4.1
	sigs.k8s.io/mcs-api v0.0.0-20200908023942-d26176718973
	sigs.k8s.io/structured-merge-diff/v4 v4.0.1 // indirect
	sigs.k8s.io/yaml v1.2.0
)

// Pinned to kubernetes-1.17.0
replace (
	github.com/googleapis/gnostic => github.com/googleapis/gnostic v0.4.0
	github.com/projectcalico/libcalico-go => github.com/projectcalico/libcalico-go v1.7.2-0.20210323165747-09a4ca98afec
	k8s.io/api => k8s.io/api v0.17.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.17.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.17.0
	k8s.io/client-go => k8s.io/client-go v0.17.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.17.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.5.14
)
