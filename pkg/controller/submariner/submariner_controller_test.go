package submariner

import (
	"context"
	"strconv"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	submariner_v1 "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const submarinerNamespace = "submariner-operator"

var _ = BeforeSuite(func() {
	err := submariner_v1.AddToScheme(scheme.Scheme)
	Expect(err).To(Succeed())
})

var _ = Describe("Submariner controller tests", func() {
	Context("Reconciliation", testReconciliation)
})

func testReconciliation() {
	var (
		initClientObjs  []runtime.Object
		client          client.Client
		submariner      *submariner_v1.Submariner
		reconcileErr    error
		reconcileResult reconcile.Result
	)

	BeforeEach(func() {
		initClientObjs = nil
		submariner = newSubmariner()
	})

	JustBeforeEach(func() {
		initClientObjs = append(initClientObjs, submariner)
		client = fake.NewFakeClientWithScheme(scheme.Scheme, initClientObjs...)

		controller := &ReconcileSubmariner{client, scheme.Scheme}

		reconcileResult, reconcileErr = controller.Reconcile(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: submariner.Namespace,
			Name:      submariner.Name,
		}})
	})

	When("the submariner engine DaemonSet doesn't exist", func() {
		It("should create it", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			verifyEngineDaemonSet(submariner, client)
		})
	})

	When("the submariner engine DaemonSet already exists", func() {
		var existingDaemonSet *appsv1.DaemonSet

		BeforeEach(func() {
			existingDaemonSet = newEngineDaemonSet(submariner)
			initClientObjs = append(initClientObjs, existingDaemonSet)
		})

		It("should return success", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			Expect(getEngineDaemonSet(client)).To(Equal(existingDaemonSet))
		})
	})

	When("a previous submariner engine Deployment exists", func() {
		BeforeEach(func() {
			initClientObjs = append(initClientObjs, &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "submariner",
					Namespace: submarinerNamespace,
				},
			})
		})

		It("should delete it", func() {
			Expect(reconcileErr).To(Succeed())
			Expect(reconcileResult.Requeue).To(BeFalse())
			verifyEngineDaemonSet(submariner, client)

			err := client.Get(context.TODO(), types.NamespacedName{Name: "submariner", Namespace: submarinerNamespace}, &appsv1.Deployment{})
			Expect(errors.IsNotFound(err)).To(BeTrue(), "IsNotFound error")
		})
	})
}

func verifyEngineDaemonSet(submariner *submariner_v1.Submariner, client client.Client) {
	daemonSet := getEngineDaemonSet(client)

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
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_SERVICECIDR", submariner.Spec.ServiceCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_CLUSTERCIDR", submariner.Spec.ClusterCIDR))
	Expect(envMap).To(HaveKeyWithValue("SUBMARINER_NAMESPACE", submariner.Spec.Namespace))
}

func newSubmariner() *submariner_v1.Submariner {
	return &submariner_v1.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "submariner",
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
			ServiceCIDR:              "100.94.0.0/16",
			ClusterCIDR:              "10.244.0.0/16",
			Namespace:                "submariner_ns",
		},
	}
}

func getDaemonSet(name string, client client.Client) *appsv1.DaemonSet {
	foundDaemonSet := &appsv1.DaemonSet{}
	err := client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: submarinerNamespace}, foundDaemonSet)
	Expect(err).To(Succeed())
	return foundDaemonSet
}

func getEngineDaemonSet(client client.Client) *appsv1.DaemonSet {
	return getDaemonSet("submariner", client)
}
