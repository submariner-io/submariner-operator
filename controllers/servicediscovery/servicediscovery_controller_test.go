/*
© 2021 Red Hat, Inc. and others

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
package servicediscovery

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	operatorv1 "github.com/openshift/api/operator/v1"
	submariner_v1 "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientset "k8s.io/client-go/kubernetes"
	fakeKubeClient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	submarinerName      = "submariner"
	submarinerNamespace = "submariner-operator"
	clusterlocalConfig  = `clusterset.local:53 {
    forward . `
	superClusterlocalConfig = `supercluster.local:53 {
    forward . `
	IP = "10.10.10.10"
)

var _ = BeforeSuite(func() {
	err := submariner_v1.AddToScheme(scheme.Scheme)
	Expect(err).To(Succeed())
})

var _ = Describe("", func() {
	logf.SetLogger(klogr.New())
	klog.InitFlags(nil)
})

var _ = Describe("ServiceDiscovery controller tests", func() {
	Context("Reconciliation", testReconciliation)
})

func testReconciliation() {
	var (
		initClientObjs   []runtime.Object
		fakeClient       controllerClient.Client
		fakeK8sClient    clientset.Interface
		serviceDiscovery *submariner_v1.ServiceDiscovery
		controller       *ServiceDiscoveryReconciler
		reconcileErr     error
		reconcileResult  reconcile.Result
	)

	newClient := func() controllerClient.Client {
		return fake.NewFakeClientWithScheme(scheme.Scheme, initClientObjs...)
	}

	BeforeEach(func() {
		fakeClient = nil
		serviceDiscovery = newServiceDiscovery()
		initClientObjs = []runtime.Object{serviceDiscovery}
	})

	JustBeforeEach(func() {
		if fakeClient == nil {
			fakeClient = newClient()
		}
		controller = &ServiceDiscoveryReconciler{
			client:            fakeClient,
			scheme:            scheme.Scheme,
			k8sClientSet:      fakeK8sClient,
			operatorClientSet: fakeClient,
		}

		reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: submarinerNamespace,
			Name:      submarinerName,
		}})
	})

	When("ClusterDNS operator should be updated when the lighthouseDNS service IP is updated", func() {
		var dnsconfig *operatorv1.DNS
		var lighthouseDNSService *corev1.Service
		oldClusterIP := IP
		updatedClusterIP := "10.10.10.11"
		BeforeEach(func() {
			dnsconfig = newDNSConfig(oldClusterIP)
			lighthouseDNSService = newDNSService(updatedClusterIP)
			initClientObjs = append(initClientObjs, dnsconfig, lighthouseDNSService)
			fakeK8sClient = fakeKubeClient.NewSimpleClientset()
		})

		It("ClusterDNS operator  not be updated when the lighthouseDNS service IP is not updated", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())

			Expect(fakeClient.Update(context.TODO(), serviceDiscovery)).To(Succeed())

			reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: submarinerNamespace,
				Name:      submarinerName,
			}})

			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(expectDNSConfigUpdated(defaultOpenShiftDNSController, fakeClient).Spec).To(Equal(newDNSConfig(updatedClusterIP).Spec))
		})
	})

	When("ClusterDNS operator should not be updated when the lighthouseDNS service IP is not updated", func() {
		var dnsconfig *operatorv1.DNS
		var lighthouseDNSService *corev1.Service
		clusterIP := IP
		BeforeEach(func() {
			dnsconfig = newDNSConfig(clusterIP)
			lighthouseDNSService = newDNSService(clusterIP)
			initClientObjs = append(initClientObjs, dnsconfig, lighthouseDNSService)
			fakeK8sClient = fakeKubeClient.NewSimpleClientset()
		})

		It("the DNS config should not be updated", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())

			Expect(fakeClient.Update(context.TODO(), serviceDiscovery)).To(Succeed())

			reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: submarinerNamespace,
				Name:      submarinerName,
			}})

			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(expectDNSConfigUpdated(defaultOpenShiftDNSController, fakeClient).Spec).To(Equal(newDNSConfig(clusterIP).Spec))
		})
	})

	When("The coreDNS configmap should be updated if the lighthouse clusterIP is not configured", func() {
		var lighthouseDNSService *corev1.Service
		clusterIP := IP
		BeforeEach(func() {
			lighthouseDNSService = newDNSService(clusterIP)
			configMap := newConfigMap("")
			initClientObjs = append(initClientObjs, lighthouseDNSService)
			fakeK8sClient = fakeKubeClient.NewSimpleClientset(configMap)
		})

		It("the coreDNS config map should be updated", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())

			Expect(fakeClient.Update(context.TODO(), serviceDiscovery)).To(Succeed())

			reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: submarinerNamespace,
				Name:      submarinerName,
			}})

			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(expectCoreMapUpdated(fakeK8sClient).Data["Corefile"]).To(ContainSubstring(clusterlocalConfig + clusterIP + "\n}"))
			Expect(expectCoreMapUpdated(fakeK8sClient).Data["Corefile"]).To(ContainSubstring(superClusterlocalConfig + clusterIP + "\n}"))
		})
	})
	When("The coreDNS configmap should be updated if the lighthouse clusterIP is already configured", func() {
		var lighthouseDNSService *corev1.Service
		clusterIP := IP
		updatedClusterIP := "10.10.10.11"
		BeforeEach(func() {
			lighthouseDNSService = newDNSService(updatedClusterIP)
			lightHouseConfig := clusterlocalConfig + clusterIP + "\n}" + superClusterlocalConfig + clusterIP + "\n}"
			configMap := newConfigMap(lightHouseConfig)
			initClientObjs = append(initClientObjs, lighthouseDNSService)
			fakeK8sClient = fakeKubeClient.NewSimpleClientset(configMap)
		})

		It("the coreDNS config map should be updated", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())

			Expect(fakeClient.Update(context.TODO(), serviceDiscovery)).To(Succeed())

			reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: submarinerNamespace,
				Name:      submarinerName,
			}})

			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(expectCoreMapUpdated(fakeK8sClient).Data["Corefile"]).To(ContainSubstring(clusterlocalConfig + updatedClusterIP))
			Expect(expectCoreMapUpdated(fakeK8sClient).Data["Corefile"]).To(ContainSubstring(superClusterlocalConfig + updatedClusterIP))
		})
	})
}

func newServiceDiscovery() *submariner_v1.ServiceDiscovery {
	return &submariner_v1.ServiceDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Name:      submarinerName,
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
			Name:      coreDNSName,
			Namespace: coreDNSNamespace,
		},
		Data: map[string]string{
			"Corefile": corefile,
		},
		BinaryData: nil,
	}
}

func newDNSConfig(clusterIP string) *operatorv1.DNS {
	return &operatorv1.DNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: defaultOpenShiftDNSController,
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

func newDNSService(clusterIP string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lighthouseCoreDNSName,
			Namespace: submarinerNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: clusterIP,
		},
	}
}

func getDNSConfig(name string, client controllerClient.Client) (*operatorv1.DNS, error) {
	foundDNSConfig := &operatorv1.DNS{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name}, foundDNSConfig)
	return foundDNSConfig, err
}

func expectDNSConfigUpdated(name string, client controllerClient.Client) *operatorv1.DNS {
	foundDNSConfig, err := getDNSConfig(name, client)
	Expect(err).To(Succeed())
	return foundDNSConfig
}

func expectCoreMapUpdated(client clientset.Interface) *corev1.ConfigMap {
	foundCoreMap, err := client.CoreV1().ConfigMaps(coreDNSNamespace).Get(coreDNSName, metav1.GetOptions{})
	Expect(err).To(Succeed())
	return foundCoreMap
}
