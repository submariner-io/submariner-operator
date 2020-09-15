package servicediscovery

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	operatorv1 "github.com/openshift/api/operator/v1"
	submariner_v1 "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
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
		controller       *ReconcileServiceDiscovery
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

		if fakeK8sClient == nil {
			fakeK8sClient = fakeKubeClient.NewSimpleClientset()
		}

		controller = &ReconcileServiceDiscovery{
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

	When("the lighthouseDNS service IP is updated", func() {
		var dnsconfig *operatorv1.DNS
		var lighthouseDNSService *corev1.Service
		oldClusterIp := "10.10.10.10"
		updatedClusterIp := "10.10.10.11"
		BeforeEach(func() {
			dnsconfig = newDNSConfig(oldClusterIp)
			lighthouseDNSService = newDNSService(updatedClusterIp)
			initClientObjs = append(initClientObjs, dnsconfig, lighthouseDNSService)
		})

		It("should update the DNS operator config", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())

			Expect(fakeClient.Update(context.TODO(), serviceDiscovery)).To(Succeed())

			reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: submarinerNamespace,
				Name:      submarinerName,
			}})

			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(expectDNSConfigUpdated(defaultOpenShiftDNSController, fakeClient).Spec).To(Equal(newDNSConfig(updatedClusterIp).Spec))
		})
	})

	When("the lighthouseDNS service IP is not updated", func() {
		var dnsconfig *operatorv1.DNS
		var lighthouseDNSService *corev1.Service
		clusterIp := "10.10.10.10"
		BeforeEach(func() {
			dnsconfig = newDNSConfig(clusterIp)
			lighthouseDNSService = newDNSService(clusterIp)
			initClientObjs = append(initClientObjs, dnsconfig, lighthouseDNSService)
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
			Expect(expectDNSConfigUpdated(defaultOpenShiftDNSController, fakeClient).Spec).To(Equal(newDNSConfig(clusterIp).Spec))
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
		},
	}
}

func newDNSConfig(clusterIp string) *operatorv1.DNS {
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
						Upstreams: []string{clusterIp},
					},
				},
				{
					Name:  "lighthouse",
					Zones: []string{"supercluster.local"},
					ForwardPlugin: operatorv1.ForwardPlugin{
						Upstreams: []string{clusterIp},
					},
				},
			},
		},
	}
}

func newDNSService(clusterIp string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lighthouseCoreDNSName,
			Namespace: submarinerNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: clusterIp,
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
