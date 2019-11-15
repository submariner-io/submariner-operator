package network

import (
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("discoverCanalFlannelNetwork", func() {
	When("There are no generic k8s pods to look at", func() {
		It("Should return still return the pod CIDR", func() {
			clusterNet := testDiscoverCanalFlannelWith(&canalFlannelCfgMap)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal("canal-flannel"))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testCannalFlannelPodCIDR}))
		})
	})

	When("There is a kubeapi pod at least ", func() {

		It("Should return the ClusterNetwork structure with ServiceCIDRs too", func() {
			clusterNet := testDiscoverWith(
				&canalFlannelCfgMap,
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []v1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
			Expect(clusterNet.NetworkPlugin).To(Equal("canal-flannel"))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testCannalFlannelPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
		})

	})
})

func testDiscoverCanalFlannelWith(objects ...runtime.Object) *ClusterNetwork {

	clientSet := fake.NewSimpleClientset(objects...)
	clusterNet, err := discoverCanalFlannelNetwork(clientSet)
	Expect(err).NotTo(HaveOccurred())
	return clusterNet
}

func testDiscoverWith(objects ...runtime.Object) *ClusterNetwork {

	clientSet := fake.NewSimpleClientset(objects...)
	clusterNet, err := Discover(nil, clientSet)
	Expect(err).NotTo(HaveOccurred())
	return clusterNet
}

var canalFlannelCfgMap v1.ConfigMap = v1.ConfigMap{
	ObjectMeta: v1meta.ObjectMeta{
		Name:      "canal-config",
		Namespace: "kube-system",
	},
	Data: map[string]string{
		"net-conf.json": `{
			"Network": "10.0.0.0/8",
			"SubnetLen": 20,
			"SubnetMin": "10.10.0.0",
			"SubnetMax": "10.99.0.0",
			"Backend": {
				"Type": "udp",
				"Port": 7890
			}
		}`,
	},
}

const testCannalFlannelPodCIDR = "10.0.0.0/8"
