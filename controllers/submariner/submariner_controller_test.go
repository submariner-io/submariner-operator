package submariner

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	submariner_v1 "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner-operator/pkg/versions"
	appsv1 "k8s.io/api/apps/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	submarinerName          = "submariner"
	submarinerNamespace     = "submariner-operator"
	engineDaemonSetName     = "submariner-gateway"
	routeAgentDaemonSetName = "submariner-routeagent"
)

type failingClient struct {
	controllerClient.Client
	onCreate reflect.Type
	onGet    reflect.Type
	onUpdate reflect.Type
}

func (c *failingClient) Create(ctx context.Context, obj runtime.Object, opts ...controllerClient.CreateOption) error {
	if c.onCreate == reflect.TypeOf(obj) {
		return fmt.Errorf("Mock Create error")
	}

	return c.Client.Create(ctx, obj, opts...)
}

func (c *failingClient) Get(ctx context.Context, key controllerClient.ObjectKey, obj runtime.Object) error {
	if c.onGet == reflect.TypeOf(obj) {
		return fmt.Errorf("Mock Get error")
	}

	return c.Client.Get(ctx, key, obj)
}

func (c *failingClient) Update(ctx context.Context, obj runtime.Object, opts ...controllerClient.UpdateOption) error {
	if c.onUpdate == reflect.TypeOf(obj) {
		return fmt.Errorf("Mock Get error")
	}

	return c.Client.Update(ctx, obj, opts...)
}

var _ = BeforeSuite(func() {
	err := submariner_v1.AddToScheme(scheme.Scheme)
	Expect(err).To(Succeed())
	err = apiextensions.AddToScheme(scheme.Scheme)
	Expect(err).To(Succeed())
})

var _ = Describe("", func() {
	logf.SetLogger(klogr.New())
	klog.InitFlags(nil)
})

var _ = Describe("Submariner controller tests", func() {
	Context("Reconciliation", testReconciliation)
})

const testDetectedServiceCIDR = "100.94.0.0/16"
const testDetectedClusterCIDR = "10.244.0.0/16"

const testConfiguredServiceCIDR = "192.168.66.0/24"
const testConfiguredClusterCIDR = "192.168.67.0/24"

func testReconciliation() {
	var (
		initClientObjs  []runtime.Object
		fakeClient      controllerClient.Client
		submariner      *submariner_v1.Submariner
		controller      *SubmarinerReconciler
		reconcileErr    error
		reconcileResult reconcile.Result
		clusterNetwork  *network.ClusterNetwork
	)

	newClient := func() controllerClient.Client {
		return fake.NewFakeClientWithScheme(scheme.Scheme, initClientObjs...)
	}

	BeforeEach(func() {
		fakeClient = nil
		submariner = newSubmariner()
		initClientObjs = []runtime.Object{submariner}

		clusterNetwork = &network.ClusterNetwork{
			NetworkPlugin: "fake",
			ServiceCIDRs:  []string{testDetectedServiceCIDR},
			PodCIDRs:      []string{testDetectedClusterCIDR},
		}
	})

	JustBeforeEach(func() {
		if fakeClient == nil {
			fakeClient = newClient()
		}

		controller = &SubmarinerReconciler{
			client:         fakeClient,
			scheme:         scheme.Scheme,
			clusterNetwork: clusterNetwork,
		}

		reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: submarinerNamespace,
			Name:      submarinerName,
		}})
	})

	When("the network details are not provided", func() {
		It("should use the detected network", func() {
			reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: submarinerNamespace,
				Name:      submarinerName,
			}})
			updated := &submariner_v1.Submariner{}
			err := fakeClient.Get(context.TODO(), types.NamespacedName{Name: submarinerName, Namespace: submarinerNamespace}, updated)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Status.ServiceCIDR).To(Equal(testDetectedServiceCIDR))
			Expect(updated.Status.ClusterCIDR).To(Equal(testDetectedClusterCIDR))
		})
	})

	When("the network details are provided", func() {
		It("should use the provided ones instead of the detected ones", func() {
			submariner.Spec.ServiceCIDR = testConfiguredServiceCIDR
			submariner.Spec.ClusterCIDR = testConfiguredClusterCIDR

			Expect(fakeClient.Update(context.TODO(), submariner)).To(Succeed())

			reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: submarinerNamespace,
				Name:      submarinerName,
			}})

			updated := &submariner_v1.Submariner{}
			err := fakeClient.Get(context.TODO(), types.NamespacedName{Name: submarinerName, Namespace: submarinerNamespace}, updated)
			Expect(err).NotTo(HaveOccurred())
			Expect(updated.Status.ServiceCIDR).To(Equal(testConfiguredServiceCIDR))
			Expect(updated.Status.ClusterCIDR).To(Equal(testConfiguredClusterCIDR))
		})
	})

	When("the submariner engine DaemonSet doesn't exist", func() {
		It("should create it", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			verifyEngineDaemonSet(withNetworkDiscovery(submariner, clusterNetwork), fakeClient)
		})
	})

	When("the submariner engine DaemonSet already exists", func() {
		var existingDaemonSet *appsv1.DaemonSet

		BeforeEach(func() {
			existingDaemonSet = newEngineDaemonSet(submariner)
			initClientObjs = append(initClientObjs, existingDaemonSet)
		})

		It("should update it", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())

			submariner.Spec.ServiceCIDR = "101.96.1.0/16"
			Expect(fakeClient.Update(context.TODO(), submariner)).To(Succeed())

			reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: submarinerNamespace,
				Name:      submarinerName,
			}})

			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(expectDaemonSet(engineDaemonSetName, fakeClient).Spec).To(
				Equal(newEngineDaemonSet(withNetworkDiscovery(submariner, clusterNetwork)).Spec))
		})
	})

	When("the submariner route-agent DaemonSet doesn't exist", func() {
		It("should create it", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			verifyRouteAgentDaemonSet(withNetworkDiscovery(submariner, clusterNetwork), fakeClient)
		})
	})

	When("the submariner route-agent DaemonSet already exists", func() {
		var existingDaemonSet *appsv1.DaemonSet

		BeforeEach(func() {
			existingDaemonSet = newRouteAgentDaemonSet(withNetworkDiscovery(submariner, clusterNetwork))
			initClientObjs = append(initClientObjs, existingDaemonSet)
		})

		It("should update it", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())

			submariner.Spec.ClusterCIDR = "11.245.1.0/16"
			Expect(fakeClient.Update(context.TODO(), submariner)).To(Succeed())

			reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: submarinerNamespace,
				Name:      submarinerName,
			}})

			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(expectDaemonSet(routeAgentDaemonSetName, fakeClient).Spec).To(Equal(newRouteAgentDaemonSet(
				withNetworkDiscovery(submariner, clusterNetwork)).Spec))
		})
	})

	When("the Submariner resource doesn't exist", func() {
		BeforeEach(func() {
			initClientObjs = nil
		})

		It("should return success without creating any resources", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			expectNoDaemonSet(engineDaemonSetName, fakeClient)
			expectNoDaemonSet(routeAgentDaemonSetName, fakeClient)
		})
	})

	When("the Submariner resource is missing values for certain fields", func() {
		BeforeEach(func() {
			submariner.Spec.Repository = ""
			submariner.Spec.Version = ""
		})

		It("should update the resource with defaults", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())

			updated := &submariner_v1.Submariner{}
			err := fakeClient.Get(context.TODO(), types.NamespacedName{Name: submarinerName, Namespace: submarinerNamespace}, updated)
			Expect(err).To(Succeed())

			Expect(updated.Spec.Repository).To(Equal(versions.DefaultRepo))
			Expect(updated.Spec.Version).To(Equal(versions.DefaultSubmarinerVersion))
		})
	})

	When("DaemonSet creation fails", func() {
		BeforeEach(func() {
			fakeClient = &failingClient{Client: newClient(), onCreate: reflect.TypeOf(&appsv1.DaemonSet{})}
		})

		It("should return an error", func() {
			Expect(reconcileErr).To(HaveOccurred())
		})
	})

	When("DaemonSet retrieval fails", func() {
		BeforeEach(func() {
			fakeClient = &failingClient{Client: newClient(), onGet: reflect.TypeOf(&appsv1.DaemonSet{})}
		})

		It("should return an error", func() {
			Expect(reconcileErr).To(HaveOccurred())
		})
	})

	When("Submariner resource retrieval fails", func() {
		BeforeEach(func() {
			fakeClient = &failingClient{Client: newClient(), onGet: reflect.TypeOf(&submariner_v1.Submariner{})}
		})

		It("should return an error", func() {
			Expect(reconcileErr).To(HaveOccurred())
		})
	})

	When("Submariner resource update fails", func() {
		BeforeEach(func() {
			fakeClient = &failingClient{Client: newClient(), onUpdate: reflect.TypeOf(&submariner_v1.Submariner{})}
		})

		It("should return an error", func() {
			Expect(reconcileErr).To(HaveOccurred())
		})
	})
}

func verifyRouteAgentDaemonSet(submariner *submariner_v1.Submariner, client controllerClient.Client) {
	daemonSet := expectDaemonSet(routeAgentDaemonSetName, client)

	Expect(daemonSet.ObjectMeta.Labels["app"]).To(Equal("submariner-routeagent"))
	Expect(daemonSet.Spec.Selector).To(Equal(&metav1.LabelSelector{MatchLabels: map[string]string{"app": "submariner-routeagent"}}))
	Expect(daemonSet.Spec.Template.ObjectMeta.Labels["app"]).To(Equal("submariner-routeagent"))
	Expect(daemonSet.Spec.Template.Spec.Containers).To(HaveLen(1))
	Expect(daemonSet.Spec.Template.Spec.Containers[0].Image).To(Equal(submariner.Spec.Repository + "/submariner-route-agent:" +
		submariner.Spec.Version))

	envMap := map[string]string{}
	for _, envVar := range daemonSet.Spec.Template.Spec.Containers[0].Env {
		envMap[envVar.Name] = envVar.Value
	}

	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NAMESPACE", submariner.Spec.Namespace))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERID", submariner.Spec.ClusterID))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", submariner.Status.ClusterCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", submariner.Status.ServiceCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NETWORKPLUGIN", "fake"))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_DEBUG", strconv.FormatBool(submariner.Spec.Debug)))
}

func verifyEngineDaemonSet(submariner *submariner_v1.Submariner, client controllerClient.Client) {
	daemonSet := expectDaemonSet(engineDaemonSetName, client)

	Expect(daemonSet.ObjectMeta.Labels["app"]).To(Equal("submariner-engine"))
	Expect(daemonSet.Spec.Template.ObjectMeta.Labels["app"]).To(Equal("submariner-engine"))
	Expect(daemonSet.Spec.Template.Spec.NodeSelector["submariner.io/gateway"]).To(Equal("true"))
	Expect(daemonSet.Spec.Template.Spec.Containers).To(HaveLen(1))
	Expect(daemonSet.Spec.Template.Spec.Containers[0].Image).To(Equal(submariner.Spec.Repository + "/submariner:" + submariner.Spec.Version))

	envMap := map[string]string{}
	for _, envVar := range daemonSet.Spec.Template.Spec.Containers[0].Env {
		envMap[envVar.Name] = envVar.Value
	}

	Expect(envMap).To(HaveKeyWithValue("CE_IPSEC_PSK", submariner.Spec.CeIPSecPSK))
	Expect(envMap).To(HaveKeyWithValue("CE_IPSEC_IKEPORT", strconv.Itoa(submariner.Spec.CeIPSecIKEPort)))
	Expect(envMap).To(HaveKeyWithValue("CE_IPSEC_NATTPORT", strconv.Itoa(submariner.Spec.CeIPSecNATTPort)))
	Expect(envMap).To(HaveKeyWithValue("BROKER_K8S_REMOTENAMESPACE", submariner.Spec.BrokerK8sRemoteNamespace))
	Expect(envMap).To(HaveKeyWithValue("BROKER_K8S_APISERVER", submariner.Spec.BrokerK8sApiServer))
	Expect(envMap).To(HaveKeyWithValue("BROKER_K8S_APISERVERTOKEN", submariner.Spec.BrokerK8sApiServerToken))
	Expect(envMap).To(HaveKeyWithValue("BROKER_K8S_CA", submariner.Spec.BrokerK8sCA))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_BROKER", submariner.Spec.Broker))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NATENABLED", strconv.FormatBool(submariner.Spec.NatEnabled)))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERID", submariner.Spec.ClusterID))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", submariner.Status.ServiceCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", submariner.Status.ClusterCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_GLOBALCIDR", submariner.Spec.GlobalCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NAMESPACE", submariner.Spec.Namespace))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_DEBUG", strconv.FormatBool(submariner.Spec.Debug)))
}

func newSubmariner() *submariner_v1.Submariner {
	return &submariner_v1.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name:      submarinerName,
			Namespace: submarinerNamespace,
		},
		Spec: submariner_v1.SubmarinerSpec{
			Repository:               "quay.io/submariner",
			Version:                  "1.0.0",
			CeIPSecNATTPort:          4500,
			CeIPSecIKEPort:           500,
			CeIPSecPSK:               "DJaA2kVW72w8kjQCEpzkDhwZuniDwgePKFE7FaxVNMWqbpmT2qvp68XW52MO70ho",
			BrokerK8sRemoteNamespace: "submariner-broker",
			BrokerK8sApiServer:       "https://192.168.99.110:8443",
			BrokerK8sApiServerToken:  "MIIDADCCAeigAw",
			BrokerK8sCA:              "client.crt",
			Broker:                   "k8s",
			NatEnabled:               true,
			ClusterID:                "east",
			ServiceCIDR:              "",
			ClusterCIDR:              "",
			GlobalCIDR:               "169.254.0.0/16",
			ColorCodes:               "red",
			Namespace:                "submariner_ns",
			Debug:                    true,
		},
	}
}

func withNetworkDiscovery(submariner *submariner_v1.Submariner, clusterNetwork *network.ClusterNetwork) *submariner_v1.Submariner {
	submariner.Status.ClusterCIDR = getClusterCIDR(submariner, clusterNetwork)
	submariner.Status.ServiceCIDR = getServiceCIDR(submariner, clusterNetwork)
	submariner.Status.NetworkPlugin = clusterNetwork.NetworkPlugin
	return submariner
}

func getClusterCIDR(submariner *submariner_v1.Submariner, clusterNetwork *network.ClusterNetwork) string {
	if submariner.Spec.ClusterCIDR != "" {
		return submariner.Spec.ClusterCIDR
	} else {
		return clusterNetwork.PodCIDRs[0]
	}
}

func getServiceCIDR(submariner *submariner_v1.Submariner, clusterNetwork *network.ClusterNetwork) string {
	if submariner.Spec.ServiceCIDR != "" {
		return submariner.Spec.ServiceCIDR
	} else {
		return clusterNetwork.ServiceCIDRs[0]
	}
}

func getDaemonSet(name string, client controllerClient.Client) (*appsv1.DaemonSet, error) {
	foundDaemonSet := &appsv1.DaemonSet{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: submarinerNamespace}, foundDaemonSet)
	return foundDaemonSet, err
}

func expectDaemonSet(name string, client controllerClient.Client) *appsv1.DaemonSet {
	foundDaemonSet, err := getDaemonSet(name, client)
	Expect(err).To(Succeed())
	return foundDaemonSet
}

func expectNoDaemonSet(name string, client controllerClient.Client) {
	_, err := getDaemonSet(name, client)
	Expect(err).To(HaveOccurred())
	Expect(errors.IsNotFound(err)).To(BeTrue(), "IsNotFound error")
}
