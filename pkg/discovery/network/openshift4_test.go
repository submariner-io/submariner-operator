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

package network

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner/pkg/routeagent_driver/constants"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var _ = Describe("discoverOpenShift4Network", func() {
	When("JSON contains a pod network", func() {
		It("Should parse properly pod and service networks", func() {
			cr := unstructuredParse(getNetworkJSON())
			cn, err := parseOS4Network(cr)
			Expect(err).NotTo(HaveOccurred())
			Expect(cn.PodCIDRs).To(HaveLen(2))
			Expect(cn.ServiceCIDRs).To(HaveLen(1))
			Expect(cn.PodCIDRs).To(Equal([]string{"10.128.0.0/14", "10.132.0.0/14"}))
			Expect(cn.ServiceCIDRs).To(Equal([]string{"172.30.0.0/16"}))
			Expect(cn.NetworkPlugin).To(Equal(constants.NetworkPluginOpenShiftSDN))
		})
	})

	When("JSON is missing the clusterNetworks list", func() {
		It("Should return error", func() {
			cr := unstructuredParse(getNetworkJSONMissingCN())
			_, err := parseOS4Network(cr)
			Expect(err).To(HaveOccurred())
		})
	})

	When("JSON is missing the serviceNetwork field", func() {
		It("Should return error", func() {
			cr := unstructuredParse(getNetworkJSONMissingSN())
			_, err := parseOS4Network(cr)
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
