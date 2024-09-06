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

package cidr_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/cidr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Allocate", func() {
	var cidrInfo cidr.Info

	BeforeEach(func() {
		cidrInfo = cidr.Info{
			CIDR:           "169.254.0.0/16",
			AllocationSize: 8192,
			Clusters:       map[string]*cidr.ClusterInfo{},
		}
	})

	When("no CIDRs are already allocated", func() {
		It("should allocate the next CIDR in sequence", func() {
			// First

			result, err := cidr.Allocate(&cidrInfo)
			Expect(err).To(Succeed())
			Expect(result).To(Equal("169.254.0.0/19"))

			// Second

			cidrInfo.Clusters["cluster1"] = &cidr.ClusterInfo{
				ClusterID: "cluster1",
				CIDRs:     []string{result},
			}

			result, err = cidr.Allocate(&cidrInfo)
			Expect(err).To(Succeed())
			Expect(result).To(Equal("169.254.32.0/19"))

			// Third

			cidrInfo.Clusters["cluster2"] = &cidr.ClusterInfo{
				ClusterID: "cluster2",
				CIDRs:     []string{result},
			}

			result, err = cidr.Allocate(&cidrInfo)
			Expect(err).To(Succeed())
			Expect(result).To(Equal("169.254.64.0/19"))
		})
	})

	When("there is an unallocated block available at beginning", func() {
		It("should allocate the CIDR block at the beginning", func() {
			cidrInfo.Clusters["cluster1"] = &cidr.ClusterInfo{
				ClusterID: "cluster1",
				CIDRs:     []string{"169.254.32.0/19"},
			}

			result, err := cidr.Allocate(&cidrInfo)
			Expect(err).To(Succeed())
			Expect(result).To(Equal("169.254.0.0/19"))
		})
	})

	When("there is an unallocated block between two allocated blocks", func() {
		It("should allocate the unallocated block", func() {
			cidrInfo.Clusters["cluster1"] = &cidr.ClusterInfo{
				ClusterID: "cluster1",
				CIDRs:     []string{"169.254.0.0/19"},
			}

			cidrInfo.Clusters["cluster2"] = &cidr.ClusterInfo{
				ClusterID: "cluster2",
				CIDRs:     []string{"169.254.64.0/19"},
			}

			result, err := cidr.Allocate(&cidrInfo)
			Expect(err).To(Succeed())
			Expect(result).To(Equal("169.254.32.0/19"))
		})
	})

	When("all CIDRs are already allocated", func() {
		It("should return an error", func() {
			cidrInfo.AllocationSize = 32768

			cidrInfo.Clusters["cluster1"] = &cidr.ClusterInfo{
				ClusterID: "cluster1",
				CIDRs:     []string{"169.254.0.0/17"},
			}

			cidrInfo.Clusters["cluster2"] = &cidr.ClusterInfo{
				ClusterID: "cluster2",
				CIDRs:     []string{"169.254.128.0/17"},
			}

			_, err := cidr.Allocate(&cidrInfo)
			Expect(err).To(HaveOccurred())
		})
	})

	When("there's not enough space for new allocation", func() {
		It("should return an error", func() {
			cidrInfo.AllocationSize = 32768

			cidrInfo.Clusters["cluster1"] = &cidr.ClusterInfo{
				ClusterID: "cluster1",
				CIDRs:     []string{"169.254.0.0/18"},
			}

			cidrInfo.Clusters["cluster2"] = &cidr.ClusterInfo{
				ClusterID: "cluster2",
				CIDRs:     []string{"169.254.128.0/17"},
			}

			_, err := cidr.Allocate(&cidrInfo)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("CheckForOverlappingCIDRs", func() {
	var (
		existingCIDRs []string
		requestedCIDR string
		retError      error
	)

	BeforeEach(func() {
		existingCIDRs = []string{}
		requestedCIDR = ""
	})

	JustBeforeEach(func() {
		retError = cidr.CheckForOverlappingCIDRs(map[string]*cidr.ClusterInfo{
			"east": {
				ClusterID: "east",
				CIDRs:     existingCIDRs,
			},
		}, requestedCIDR, "west")
	})

	When("there are no existing CIDRs", func() {
		BeforeEach(func() {
			requestedCIDR = "10.10.10.0/24"
		})

		It("should not return error", func() {
			Expect(retError).To(Succeed())
		})
	})

	When("there is at least one existing CIDR that is a superset of the requested CIDR", func() {
		BeforeEach(func() {
			requestedCIDR = "10.10.10.0/24"
			existingCIDRs = []string{"10.10.0.0/16", "10.20.30.0/24"}
		})

		It("should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})

	When("there is at least one existing CIDR that is subset of the requested CIDR", func() {
		BeforeEach(func() {
			requestedCIDR = "10.10.30.0/16"
			existingCIDRs = []string{"10.10.10.0/24", "10.10.20.0/24"}
		})

		It("should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})

	When("the requested CIDR partially overlaps with at least one existing CIDR", func() {
		BeforeEach(func() {
			requestedCIDR = "10.10.10.128/22"
			existingCIDRs = []string{"10.10.10.0/24"}
		})

		It("should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})

	When("the requested CIDR lies between any two existing CIDR", func() {
		BeforeEach(func() {
			requestedCIDR = "10.10.20.128/24"
			existingCIDRs = []string{"10.10.10.0/24", "10.10.30.0/24"}
		})

		It("should not return error", func() {
			Expect(retError).To(Succeed())
		})
	})

	When("there is at least one existing CIDR that is invalid", func() {
		BeforeEach(func() {
			requestedCIDR = "10.10.20.128/24"
			existingCIDRs = []string{"10.10.10.0/33", "10.10.30.0/24"}
		})

		It("should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})

	When("the requested CIDR is invalid", func() {
		BeforeEach(func() {
			requestedCIDR = "0.10.20.300/24"
			existingCIDRs = []string{"10.10.10.0/24", "10.10.30.0/24"}
		})

		It("should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})
})

var _ = Describe("IsValid", func() {
	Specify("a valid CIDR should succeed", func() {
		Expect(cidr.IsValid("10.10.20.128/24")).To(Succeed())
	})

	Specify("an invalid CIDR should return an error", func() {
		Expect(cidr.IsValid("")).ToNot(Succeed())
		Expect(cidr.IsValid("1.2")).ToNot(Succeed())
	})

	Specify("an loopback CIDR should return an error", func() {
		Expect(cidr.IsValid("127.0.0.0/16")).ToNot(Succeed())
	})

	Specify("a Link-Local CIDR should return an error", func() {
		Expect(cidr.IsValid("169.254.0.0/16")).ToNot(Succeed())
	})

	Specify("Link-Local Multicast CIDR should return an error", func() {
		Expect(cidr.IsValid("224.0.0.0/24")).ToNot(Succeed())
	})
})

var _ = Describe("AddClusterInfoData", func() {
	It("should succeed", func() {
		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test",
			},
		}

		// Add first cluster

		clusterInfo1 := cidr.ClusterInfo{
			ClusterID: "east",
			CIDRs:     []string{"169.254.0.0/19"},
		}

		err := cidr.AddClusterInfoData(configMap, clusterInfo1)
		Expect(err).To(Succeed())

		infoMap, err := cidr.ExtractClusterInfo(configMap)
		Expect(err).To(Succeed())

		Expect(infoMap).To(Equal(map[string]*cidr.ClusterInfo{
			clusterInfo1.ClusterID: &clusterInfo1,
		}))

		// Update first cluster

		clusterInfo1.CIDRs = []string{"169.254.64.0/19"}

		err = cidr.AddClusterInfoData(configMap, clusterInfo1)
		Expect(err).To(Succeed())

		infoMap, err = cidr.ExtractClusterInfo(configMap)
		Expect(err).To(Succeed())

		Expect(infoMap).To(Equal(map[string]*cidr.ClusterInfo{
			clusterInfo1.ClusterID: &clusterInfo1,
		}))

		// Add second cluster

		clusterInfo2 := cidr.ClusterInfo{
			ClusterID: "west",
			CIDRs:     []string{"169.254.32.0/19"},
		}

		err = cidr.AddClusterInfoData(configMap, clusterInfo2)
		Expect(err).To(Succeed())

		infoMap, err = cidr.ExtractClusterInfo(configMap)
		Expect(err).To(Succeed())

		Expect(infoMap).To(Equal(map[string]*cidr.ClusterInfo{
			clusterInfo1.ClusterID: &clusterInfo1,
			clusterInfo2.ClusterID: &clusterInfo2,
		}))
	})
})
