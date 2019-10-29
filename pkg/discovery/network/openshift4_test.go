package network

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("discoverOpenShift4Network", func() {
	When("JSON contains a single pod network", func() {
		It("Should parse properly pod and service networks", func() {
			cr := unstructuredParse(getClusterNetworkJSON())
			cn, err := parseOS4ClusterNetwork(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(cn.PodCIDRs).To(HaveLen(2))
			Expect(cn.ServiceCIDRs).To(HaveLen(1))
			Expect(cn.PodCIDRs).To(Equal([]string{"10.128.0.0/14", "10.132.0.0/14"}))
			Expect(cn.ServiceCIDRs).To(Equal([]string{"172.30.0.0/16"}))

		})
	})

	When("JSON is missing the clusterNetworks list", func() {
		It("Should return error", func() {
			cr := unstructuredParse(getClusterNetworkJSONMissingCN())
			_, err := parseOS4ClusterNetwork(cr)
			Expect(err).To(HaveOccurred())
		})
	})

	When("JSON is missing the serviceNetwork field", func() {
		It("Should return error", func() {
			cr := unstructuredParse(getClusterNetworkJSONMissingSN())
			_, err := parseOS4ClusterNetwork(cr)
			Expect(err).To(HaveOccurred())
		})
	})

})

func unstructuredParse(json []byte) *unstructured.Unstructured {
	crd := &unstructured.Unstructured{}
	err := crd.UnmarshalJSON(json)
	Expect(err).NotTo(HaveOccurred())
	return crd
}

func getClusterNetworkJSON() []byte {
	return []byte(`{
    "apiVersion": "network.openshift.io/v1",
    "clusterNetworks": [
        {
            "CIDR": "10.128.0.0/14",
            "hostSubnetLength": 9
        },
		{
            "CIDR": "10.132.0.0/14",
            "hostSubnetLength": 9
        }
    ],
    "hostsubnetlength": 9,
    "kind": "ClusterNetwork",
    "metadata": {
        "creationTimestamp": "2019-10-28T19:52:03Z",
        "generation": 1,
        "name": "default",
        "ownerReferences": [
            {
                "apiVersion": "operator.openshift.io/v1",
                "blockOwnerDeletion": true,
                "controller": true,
                "kind": "Network",
                "name": "cluster",
                "uid": "61d2c29b-f9bc-11e9-809d-026caba2345a"
            }
        ],
        "resourceVersion": "1422",
        "selfLink": "/apis/network.openshift.io/v1/clusternetworks/default",
        "uid": "69d0bf65-f9bc-11e9-809d-026caba2345a"
    },
    "mtu": 8951,
    "network": "10.128.0.0/14",
    "pluginName": "redhat/openshift-ovs-networkpolicy",
    "serviceNetwork": "172.30.0.0/16",
    "vxlanPort": 4789
}`)
}

func getClusterNetworkJSONMissingCN() []byte {
	return []byte(`{
    "apiVersion": "network.openshift.io/v1",
    "hostsubnetlength": 9,
    "kind": "ClusterNetwork",
    "metadata": {
        "creationTimestamp": "2019-10-28T19:52:03Z",
        "generation": 1,
        "name": "default",
        "ownerReferences": [
            {
                "apiVersion": "operator.openshift.io/v1",
                "blockOwnerDeletion": true,
                "controller": true,
                "kind": "Network",
                "name": "cluster",
                "uid": "61d2c29b-f9bc-11e9-809d-026caba2345a"
            }
        ],
        "resourceVersion": "1422",
        "selfLink": "/apis/network.openshift.io/v1/clusternetworks/default",
        "uid": "69d0bf65-f9bc-11e9-809d-026caba2345a"
    },
    "mtu": 8951,
    "network": "10.128.0.0/14",
    "pluginName": "redhat/openshift-ovs-networkpolicy",
    "serviceNetwork": "172.30.0.0/16",
    "vxlanPort": 4789
}`)
}

func getClusterNetworkJSONMissingSN() []byte {
	return []byte(`{
    "apiVersion": "network.openshift.io/v1",
    "clusterNetworks": [
        {
            "CIDR": "10.128.0.0/14",
            "hostSubnetLength": 9
        }
    ],
    "hostsubnetlength": 9,
    "kind": "ClusterNetwork",
    "metadata": {
        "creationTimestamp": "2019-10-28T19:52:03Z",
        "generation": 1,
        "name": "default",
        "ownerReferences": [
            {
                "apiVersion": "operator.openshift.io/v1",
                "blockOwnerDeletion": true,
                "controller": true,
                "kind": "Network",
                "name": "cluster",
                "uid": "61d2c29b-f9bc-11e9-809d-026caba2345a"
            }
        ],
        "resourceVersion": "1422",
        "selfLink": "/apis/network.openshift.io/v1/clusternetworks/default",
        "uid": "69d0bf65-f9bc-11e9-809d-026caba2345a"
    },
    "mtu": 8951,
    "network": "10.128.0.0/14",
    "pluginName": "redhat/openshift-ovs-networkpolicy",
    "vxlanPort": 4789
}`)
}

func TestOpenShift4NetworkDiscovery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpenShift4 network discovery")
}
