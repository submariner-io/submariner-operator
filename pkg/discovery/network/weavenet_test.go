package network

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("discoverWeaveNetwork", func() {
	When("There are no weave pods", func() {
		It("Should return nil cluster network", func() {
			clusterNet := testDiscoverWeaveWith()
			Expect(clusterNet).To(BeNil())
		})
	})

	When("There are weave pods but no kube api", func() {

		var clusterNet *ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverWeaveWith(
				fakePod("weave-net", []string{"weave-net"}, []v1.EnvVar{{Name: "IPALLOC_RANGE", Value: testPodCIDR}}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})
		It("Should return the ClusterNetwork structure with PodCIDR and no ServiceCIDRs", func() {
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
			Expect(clusterNet.ServiceCIDRs).To(BeEmpty())
		})

		It("Should identify the networkplugin as weave-net", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo("weave-net"))
		})
	})

	When("There are weave and kube api pods", func() {

		var clusterNet *ClusterNetwork

		BeforeEach(func() {
			clusterNet = testDiscoverWeaveWith(
				fakePod("weave-net", []string{"weave-net"}, []v1.EnvVar{{Name: "IPALLOC_RANGE", Value: testPodCIDR}}),
				fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []v1.EnvVar{}),
			)
			Expect(clusterNet).NotTo(BeNil())
		})

		It("Should return ClusterNetwork with Pod and Service CIDRs", func() {
			Expect(clusterNet.ServiceCIDRs).To(Equal([]string{testServiceCIDR}))
			Expect(clusterNet.PodCIDRs).To(Equal([]string{testPodCIDR}))
		})

		It("Should identify the network plugin as weave", func() {
			Expect(clusterNet.NetworkPlugin).To(BeIdenticalTo("weave-net"))
		})
	})
})

func testDiscoverWeaveWith(objects ...runtime.Object) *ClusterNetwork {
	clientSet := fake.NewSimpleClientset(objects...)
	clusterNet, err := discoverWeaveNetwork(clientSet)
	Expect(err).NotTo(HaveOccurred())
	return clusterNet
}
