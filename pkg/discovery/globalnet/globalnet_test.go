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

package globalnet_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
)

var _ = Describe("CheckOverlappingCidrs", func() {
	var (
		globalCIDRs   []string
		globalnetCIDR string
		retError      error
	)

	BeforeEach(func() {
		globalCIDRs = []string{}
		globalnetCIDR = ""
	})

	JustBeforeEach(func() {
		retError = globalnet.CheckOverlappingCidrs(&globalnet.Info{
			CidrInfo: map[string]*globalnet.GlobalNetwork{
				"east": {
					ClusterID:   "east",
					GlobalCIDRs: globalCIDRs,
				},
			},
		}, globalnet.Config{
			ClusterID:     "west",
			GlobalnetCIDR: globalnetCIDR,
		})
	})

	When("There are no base CIDRs", func() {
		BeforeEach(func() {
			globalnetCIDR = "10.10.10.0/24"
		})

		It("Should not return error", func() {
			Expect(retError).ToNot(HaveOccurred())
		})
	})

	When("At least one Base CIDR is superset of new CIDR", func() {
		BeforeEach(func() {
			globalnetCIDR = "10.10.10.0/24"
			globalCIDRs = []string{"10.10.0.0/16", "10.20.30.0/24"}
		})

		It("Should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})

	When("At least one Base CIDR is subset of new CIDR", func() {
		BeforeEach(func() {
			globalnetCIDR = "10.10.30.0/16"
			globalCIDRs = []string{"10.10.10.0/24", "10.10.20.0/24"}
		})

		It("Should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})

	When("New CIDR partially overlaps with at least one Base CIDR", func() {
		BeforeEach(func() {
			globalnetCIDR = "10.10.10.128/22"
			globalCIDRs = []string{"10.10.10.0/24"}
		})

		It("Should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})

	When("New CIDR lies between any two Base CIDR", func() {
		BeforeEach(func() {
			globalnetCIDR = "10.10.20.128/24"
			globalCIDRs = []string{"10.10.10.0/24", "10.10.30.0/24"}
		})

		It("Should not return error", func() {
			Expect(retError).ToNot(HaveOccurred())
		})
	})

	When("At least one Base CIDR is invalid", func() {
		BeforeEach(func() {
			globalnetCIDR = "10.10.20.128/24"
			globalCIDRs = []string{"10.10.10.0/33", "10.10.30.0/24"}
		})

		It("Should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})

	When("New CIDR is invalid", func() {
		BeforeEach(func() {
			globalnetCIDR = "0.10.20.300/24"
			globalCIDRs = []string{"10.10.10.0/24", "10.10.30.0/24"}
		})

		It("Should return error", func() {
			Expect(retError).To(HaveOccurred())
		})
	})
})

var _ = Describe("AllocateGlobalCIDR: Success", func() {
	globalnetInfo := globalnet.Info{CidrRange: "169.254.0.0/16", ClusterSize: 8192}
	globalnetInfo.CidrInfo = make(map[string]*globalnet.GlobalNetwork)

	When("No GlobalCIDRs are already allocated", func() {
		result, err := globalnet.AllocateGlobalCIDR(&globalnetInfo)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate next CIDR", func() {
			Expect(result).To(Equal("169.254.0.0/19"))
		})
	})
	When("There is one allocated GlobalCIDR", func() {
		globalNetwork1 := globalnet.GlobalNetwork{
			ClusterID:   "cluster2",
			GlobalCIDRs: []string{"169.254.0.0/19"},
		}
		globalnetInfo.CidrInfo[globalNetwork1.ClusterID] = &globalNetwork1
		result, err := globalnet.AllocateGlobalCIDR(&globalnetInfo)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate next CIDR", func() {
			Expect(result).To(Equal("169.254.32.0/19"))
		})
	})
	When("There is an unallocated block available at beginning", func() {
		globalNetwork1 := globalnet.GlobalNetwork{
			ClusterID:   "cluster2",
			GlobalCIDRs: []string{"169.254.32.0/19"},
		}
		globalnetInfo.CidrInfo[globalNetwork1.ClusterID] = &globalNetwork1
		result, err := globalnet.AllocateGlobalCIDR(&globalnetInfo)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate block at beginning", func() {
			Expect(result).To(Equal("169.254.0.0/19"))
		})
	})
	When("Unallocated block between two allocated blocks", func() {
		globalNetwork1 := globalnet.GlobalNetwork{
			ClusterID:   "cluster1",
			GlobalCIDRs: []string{"169.254.0.0/19"},
		}
		globalNetwork2 := globalnet.GlobalNetwork{
			ClusterID:   "cluster2",
			GlobalCIDRs: []string{"169.254.64.0/19"},
		}
		globalnetInfo.CidrInfo[globalNetwork1.ClusterID] = &globalNetwork1
		globalnetInfo.CidrInfo[globalNetwork2.ClusterID] = &globalNetwork2
		result, err := globalnet.AllocateGlobalCIDR(&globalnetInfo)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate the unallocated block", func() {
			Expect(result).To(Equal("169.254.32.0/19"))
		})
	})
	When("Two CIDRs are allocated at beginning", func() {
		globalNetwork1 := globalnet.GlobalNetwork{
			ClusterID:   "cluster1",
			GlobalCIDRs: []string{"169.254.0.0/19"},
		}
		globalNetwork2 := globalnet.GlobalNetwork{
			ClusterID:   "cluster2",
			GlobalCIDRs: []string{"169.254.32.0/19"},
		}
		globalnetInfo.CidrInfo[globalNetwork1.ClusterID] = &globalNetwork1
		globalnetInfo.CidrInfo[globalNetwork2.ClusterID] = &globalNetwork2
		result, err := globalnet.AllocateGlobalCIDR(&globalnetInfo)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate next available block", func() {
			Expect(result).To(Equal("169.254.64.0/19"))
		})
	})
})

var _ = Describe("AllocateGlobalCIDR: Fail", func() {
	globalnetInfo := globalnet.Info{CidrRange: "169.254.0.0/16", ClusterSize: 32768}
	globalnetInfo.CidrInfo = make(map[string]*globalnet.GlobalNetwork)

	When("All CIDRs are already allocated", func() {
		globalNetwork1 := globalnet.GlobalNetwork{
			ClusterID:   "cluster2",
			GlobalCIDRs: []string{"169.254.0.0/17"},
		}
		globalNetwork2 := globalnet.GlobalNetwork{
			ClusterID:   "cluster3",
			GlobalCIDRs: []string{"169.254.128.0/17"},
		}
		globalnetInfo.CidrInfo[globalNetwork1.ClusterID] = &globalNetwork1
		globalnetInfo.CidrInfo[globalNetwork2.ClusterID] = &globalNetwork2
		result, err := globalnet.AllocateGlobalCIDR(&globalnetInfo)
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
		It("Should not allocate any CIDR", func() {
			Expect(result).To(Equal(""))
		})
	})

	When("Not enough space for new cluster", func() {
		globalNetwork1 := globalnet.GlobalNetwork{
			ClusterID:   "cluster2",
			GlobalCIDRs: []string{"169.254.0.0/18"},
		}
		globalNetwork2 := globalnet.GlobalNetwork{
			ClusterID:   "cluster3",
			GlobalCIDRs: []string{"169.254.128.0/17"},
		}
		globalnetInfo.CidrInfo[globalNetwork1.ClusterID] = &globalNetwork1
		globalnetInfo.CidrInfo[globalNetwork2.ClusterID] = &globalNetwork2
		result, err := globalnet.AllocateGlobalCIDR(&globalnetInfo)
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
		It("Should not allocate any CIDR", func() {
			Expect(result).To(Equal(""))
		})
	})
})

var _ = Describe("IsValidCidr", func() {
	When("Unspecified CIDR", func() {
		err := globalnet.IsValidCIDR("")
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
		err = globalnet.IsValidCIDR("1.2")
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("Loopback CIDR", func() {
		err := globalnet.IsValidCIDR("127.0.0.0/16")
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("Link-Local CIDR", func() {
		err := globalnet.IsValidCIDR("169.254.0.0/16")
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("Link-Local Multicast CIDR", func() {
		err := globalnet.IsValidCIDR("224.0.0.0/24")
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})
})
