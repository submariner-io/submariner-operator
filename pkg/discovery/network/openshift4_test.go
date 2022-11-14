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

package network_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"github.com/submariner-io/submariner/pkg/cni"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("OpenShift4 Network", func() {
	When("JSON contains a pod network", func() {
		It("Should parse properly pod and service networks", func() {
			cn, err := testOS4DiscoveryWith(getNetworkJSON())
			Expect(err).NotTo(HaveOccurred())
			Expect(cn.PodCIDRs).To(HaveLen(2))
			Expect(cn.ServiceCIDRs).To(HaveLen(1))
			Expect(cn.PodCIDRs).To(Equal([]string{"10.128.0.0/14", "10.132.0.0/14"}))
			Expect(cn.ServiceCIDRs).To(Equal([]string{"172.30.0.0/16"}))
			Expect(cn.NetworkPlugin).To(Equal(cni.OpenShiftSDN))
		})
	})

	When("JSON is missing the clusterNetworks list", func() {
		It("Should return error", func() {
			_, err := testOS4DiscoveryWith(getNetworkJSONMissingCN())
			Expect(err).To(HaveOccurred())
		})
	})

	When("JSON is missing the serviceNetwork field", func() {
		It("Should return error", func() {
			_, err := testOS4DiscoveryWith(getNetworkJSONMissingSN())
			Expect(err).To(HaveOccurred())
		})
	})
})

func testOS4DiscoveryWith(json []byte) (*network.ClusterNetwork, error) {
	obj := &unstructured.Unstructured{}
	err := obj.UnmarshalJSON(json)
	Expect(err).NotTo(HaveOccurred())

	return network.Discover(fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(obj).Build(), "")
}

func getNetworkJSON() []byte {
	return []byte(`
        {
            "apiVersion": "config.openshift.io/v1",
            "kind": "Network",
            "metadata": {
                "creationTimestamp": "2020-09-10T07:36:54Z",
                "generation": 2,
                "name": "cluster",
                "resourceVersion": "2664",
                "selfLink": "/apis/config.openshift.io/v1/networks/cluster",
                "uid": "3b90ece9-94c7-49f7-b615-3d092efb5cd7"
            },
            "spec": {
                "clusterNetwork": [
					{
                        "cidr": "10.128.0.0/14",
                        "hostPrefix": 23
                    },
                    {
                        "cidr": "10.132.0.0/14",
                        "hostPrefix": 23
                    }
                ],
                "externalIP": {
                    "policy": {}
                },
                "networkType": "OpenShiftSDN",
                "serviceNetwork": [
                    "172.30.0.0/16"
                ]
            },
            "status": {
                "clusterNetwork": [
					{
                        "cidr": "10.128.0.0/14",
                        "hostPrefix": 23
                    },
                    {
                        "cidr": "10.132.0.0/14",
                        "hostPrefix": 23
                    }
                ],
                "clusterNetworkMTU": 8951,
                "networkType": "OpenShiftSDN",
                "serviceNetwork": [
                    "172.30.0.0/16"
                ]
            }
        }
    `)
}

func getNetworkJSONMissingCN() []byte {
	return []byte(`
		{
			"apiVersion": "config.openshift.io/v1",
			"kind": "Network",
			"metadata": {
				"creationTimestamp": "2020-09-10T07:36:54Z",
				"generation": 2,
				"name": "cluster",
				"resourceVersion": "2664",
				"selfLink": "/apis/config.openshift.io/v1/networks/cluster",
				"uid": "3b90ece9-94c7-49f7-b615-3d092efb5cd7"
			},
			"spec": {
				"externalIP": {
				"policy": {}
			},
				"networkType": "OpenShiftSDN",
				"serviceNetwork": [
				"172.31.0.0/16"
			]
			},
			"status": {
				"clusterNetwork": [
				{
					"cidr": "10.132.0.0/14",
					"hostPrefix": 23
				}
			],
				"clusterNetworkMTU": 8951,
				"networkType": "OpenShiftSDN",
				"serviceNetwork": [
				"172.31.0.0/16"
			]
			}
		}
`)
}

func getNetworkJSONMissingSN() []byte {
	return []byte(`{
            "apiVersion": "config.openshift.io/v1",
            "kind": "Network",
            "metadata": {
                "creationTimestamp": "2020-09-10T07:36:54Z",
                "generation": 2,
                "name": "cluster",
                "resourceVersion": "2664",
                "selfLink": "/apis/config.openshift.io/v1/networks/cluster",
                "uid": "3b90ece9-94c7-49f7-b615-3d092efb5cd7"
            },
            "spec": {
                "clusterNetwork": [
                    {
                        "cidr": "10.132.0.0/14",
                        "hostPrefix": 23
                    }
                ],
                "externalIP": {
                    "policy": {}
                },
                "networkType": "OpenShiftSDN"
            },
            "status": {
                "clusterNetwork": [
                    {
                        "cidr": "10.132.0.0/14",
                        "hostPrefix": 23
                    }
                ],
                "clusterNetworkMTU": 8951,
                "networkType": "OpenShiftSDN",
                "serviceNetwork": [
                    "172.31.0.0/16"
                ]
            }
        }

`)
}
