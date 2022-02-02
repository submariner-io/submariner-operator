/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package servicediscovery_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	submariner_v1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/servicediscovery"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeKubeClient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	serviceDiscoveryName   = "test-service-discovery"
	submarinerNamespace    = "test-ns"
	openShiftDNSConfigName = "default"
	clusterlocalConfig     = `clusterset.local:53 {
    forward . `
	superClusterlocalConfig = `supercluster.local:53 {
    forward . `
	clusterIP = "10.10.10.10"
)

var _ = BeforeSuite(func() {
	Expect(submariner_v1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(operatorv1.Install(scheme.Scheme)).To(Succeed())
})

var _ = Describe("", func() {
	kzerolog.InitK8sLogging()
})

func TestSubmariner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServiceDiscovery Test Suite")
}

type testDriver struct {
	initClientObjs   []controllerClient.Object
	fakeClient       controllerClient.Client
	kubeClient       *fakeKubeClient.Clientset
	serviceDiscovery *submariner_v1.ServiceDiscovery
	controller       *servicediscovery.Reconciler
}

func newTestDriver() *testDriver {
	t := &testDriver{}

	BeforeEach(func() {
		t.fakeClient = nil
		t.serviceDiscovery = newServiceDiscovery()
		t.initClientObjs = []controllerClient.Object{t.serviceDiscovery}
		t.kubeClient = fakeKubeClient.NewSimpleClientset()
	})

	JustBeforeEach(func() {
		if t.fakeClient == nil {
			t.fakeClient = t.newClient()
		}

		t.controller = servicediscovery.NewReconciler(&servicediscovery.Config{
			Client:         t.fakeClient,
			Scheme:         scheme.Scheme,
			KubeClient:     t.kubeClient,
			OperatorClient: t.fakeClient,
		})
	})

	return t
}

func (t *testDriver) newClient() controllerClient.Client {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(t.initClientObjs...).Build()
}

func (t *testDriver) doReconcile() (reconcile.Result, error) {
	return t.controller.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: submarinerNamespace,
		Name:      serviceDiscoveryName,
	}})
}

func (t *testDriver) assertReconcileSuccess() {
	r, err := t.doReconcile()
	Expect(err).To(Succeed())
	Expect(r.Requeue).To(BeFalse())
	Expect(r.RequeueAfter).To(BeNumerically("==", 0))
}

func (t *testDriver) getDNSConfig() (*operatorv1.DNS, error) {
	foundDNSConfig := &operatorv1.DNS{}
	err := t.fakeClient.Get(context.TODO(), types.NamespacedName{Name: openShiftDNSConfigName}, foundDNSConfig)

	return foundDNSConfig, err
}

func (t *testDriver) assertDNSConfig() *operatorv1.DNS {
	foundDNSConfig, err := t.getDNSConfig()
	Expect(err).To(Succeed())

	return foundDNSConfig
}

func (t *testDriver) assertCoreDNSConfigMap() *corev1.ConfigMap {
	foundCoreMap, err := t.kubeClient.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "coredns", metav1.GetOptions{})
	Expect(err).To(Succeed())

	return foundCoreMap
}

func (t *testDriver) createConfigMap(cm *corev1.ConfigMap) {
	_, err := t.kubeClient.CoreV1().ConfigMaps(cm.Namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	Expect(err).To(Succeed())
}

func newDNSConfig(clusterIP string) *operatorv1.DNS {
	return &operatorv1.DNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: openShiftDNSConfigName,
		},
		Spec: operatorv1.DNSSpec{
			Servers: []operatorv1.Server{
				{
					Name:  "lighthouse",
					Zones: []string{"clusterset.local"},
					ForwardPlugin: operatorv1.ForwardPlugin{
						Upstreams: []string{clusterIP},
					},
				},
				{
					Name:  "lighthouse",
					Zones: []string{"supercluster.local"},
					ForwardPlugin: operatorv1.ForwardPlugin{
						Upstreams: []string{clusterIP},
					},
				},
			},
		},
	}
}

func newServiceDiscovery() *submariner_v1.ServiceDiscovery {
	return &submariner_v1.ServiceDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceDiscoveryName,
			Namespace: submarinerNamespace,
		},
		Spec: submariner_v1.ServiceDiscoverySpec{
			GlobalnetEnabled:         false,
			Repository:               "quay.io/submariner",
			Version:                  "1.0.0",
			BrokerK8sRemoteNamespace: "submariner-broker",
			BrokerK8sApiServer:       "https://192.168.99.110:8443",
			BrokerK8sApiServerToken:  "MIIDADCCAeigAw",
			BrokerK8sCA:              "client.crt",
			ClusterID:                "east",
			Namespace:                "submariner_ns",
			Debug:                    true,
			CustomDomains:            []string{"supercluster.local"},
		},
	}
}

func newConfigMap(lighthouseConfig string) *corev1.ConfigMap {
	corefile := lighthouseConfig + `.:53 {
		errors
		health {
		lameduck 5s
	}
		ready
		kubernetes cluster1.local in-addr.arpa ip6.arpa {
		pods insecure
		fallthrough in-addr.arpa ip6.arpa
		ttl 30
	}
		prometheus :9153
		forward . /etc/resolv.conf
		cache 30
		loop
		reload
		loadbalance
	}`

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"Corefile": corefile,
		},
		BinaryData: nil,
	}
}

func newDNSService(clusterIP string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "submariner-lighthouse-coredns",
			Namespace: submarinerNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: clusterIP,
		},
	}
}
