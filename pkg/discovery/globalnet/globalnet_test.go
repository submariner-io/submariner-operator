/*
© 2020 Red Hat, Inc. and others.

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

package globalnet

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
)

var _ = Describe("IsOverlappingCidr", func() {
	When("There are no base CIDRs", func() {
		overlapping, err := isOverlappingCIDR([]string{}, "10.10.10.0/24")
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should return false", func() {
			Expect(overlapping).To(BeFalse())
		})
	})

	When("At least one Base CIDR is superset of new CIDR", func() {
		overlapping, err := isOverlappingCIDR([]string{"10.10.0.0/16", "10.20.30.0/24"}, "10.10.10.0/24")
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should return true", func() {
			Expect(overlapping).To(BeTrue())
		})
	})

	When("At least one Base CIDR is subset of new CIDR", func() {
		overlapping, err := isOverlappingCIDR([]string{"10.10.10.0/24", "10.10.20.0/24"}, "10.10.30.0/16")
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should return true", func() {
			Expect(overlapping).To(BeTrue())
		})
	})

	When("New CIDR partially overlaps with at least one Base CIDR", func() {
		overlapping, err := isOverlappingCIDR([]string{"10.10.10.0/24"}, "10.10.10.128/22")
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should return true", func() {
			Expect(overlapping).To(BeTrue())
		})
	})

	When("New CIDR lies between any two Base CIDR", func() {
		overlapping, err := isOverlappingCIDR([]string{"10.10.10.0/24", "10.10.30.0/24"}, "10.10.20.128/24")
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should return false", func() {
			Expect(overlapping).To(BeFalse())
		})
	})

	When("At least one Base CIDR is invalid", func() {
		_, err := isOverlappingCIDR([]string{"10.10.10.0/33", "10.10.30.0/24"}, "10.10.20.128/24")
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("New CIDR is invalid", func() {
		_, err := isOverlappingCIDR([]string{"10.10.10.0/24", "10.10.30.0/24"}, "10.10.20.300/24")
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

})

var _ = Describe("AllocateGlobalCIDR: Success", func() {
	subctlData := datafile.SubctlData{GlobalnetCidrRange: "169.254.0.0/16", GlobalnetClusterSize: 8192}

	When("No GlobalCIDRs are already allocated", func() {
		var globalNetworks = make(map[string]*GlobalNetwork)
		result, err := AllocateGlobalCIDR(globalNetworks, &subctlData)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate next CIDR", func() {
			Expect(result).To(Equal("169.254.0.0/19"))
		})
	})
	When("There is one allocated GlobalCIDR", func() {
		var globalNetworks = make(map[string]*GlobalNetwork)
		globalNetwork1 := GlobalNetwork{
			ClusterId:   "cluster2",
			GlobalCIDRs: []string{"169.254.0.0/19"},
		}
		globalNetworks[globalNetwork1.ClusterId] = &globalNetwork1
		result, err := AllocateGlobalCIDR(globalNetworks, &subctlData)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate next CIDR", func() {
			Expect(result).To(Equal("169.254.32.0/19"))
		})
	})
	When("There is an unallocated block available at beginning", func() {
		var globalNetworks = make(map[string]*GlobalNetwork)
		globalNetwork1 := GlobalNetwork{
			ClusterId:   "cluster2",
			GlobalCIDRs: []string{"169.254.32.0/19"},
		}
		globalNetworks[globalNetwork1.ClusterId] = &globalNetwork1
		result, err := AllocateGlobalCIDR(globalNetworks, &subctlData)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate block at beginning", func() {
			Expect(result).To(Equal("169.254.0.0/19"))
		})
	})
	When("Unallocated block between two allocated blocks", func() {
		var globalNetworks = make(map[string]*GlobalNetwork)
		globalNetwork1 := GlobalNetwork{
			ClusterId:   "cluster1",
			GlobalCIDRs: []string{"169.254.0.0/19"},
		}
		globalNetwork2 := GlobalNetwork{
			ClusterId:   "cluster2",
			GlobalCIDRs: []string{"169.254.64.0/19"},
		}
		globalNetworks[globalNetwork1.ClusterId] = &globalNetwork1
		globalNetworks[globalNetwork2.ClusterId] = &globalNetwork2
		result, err := AllocateGlobalCIDR(globalNetworks, &subctlData)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate the unallocated block", func() {
			Expect(result).To(Equal("169.254.32.0/19"))
		})
	})
	When("Two CIDRs are allocated at beginning", func() {
		var globalNetworks = make(map[string]*GlobalNetwork)
		globalNetwork1 := GlobalNetwork{
			ClusterId:   "cluster1",
			GlobalCIDRs: []string{"169.254.0.0/19"},
		}
		globalNetwork2 := GlobalNetwork{
			ClusterId:   "cluster2",
			GlobalCIDRs: []string{"169.254.32.0/19"},
		}
		globalNetworks[globalNetwork1.ClusterId] = &globalNetwork1
		globalNetworks[globalNetwork2.ClusterId] = &globalNetwork2
		result, err := AllocateGlobalCIDR(globalNetworks, &subctlData)
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("Should allocate next available block", func() {
			Expect(result).To(Equal("169.254.64.0/19"))
		})
	})
})

var _ = Describe("AllocateGlobalCIDR: Fail", func() {
	subctlData := datafile.SubctlData{GlobalnetCidrRange: "169.254.0.0/16", GlobalnetClusterSize: 32768}

	When("All CIDRs are already allocated", func() {
		var globalNetworks = make(map[string]*GlobalNetwork)
		globalNetwork1 := GlobalNetwork{
			ClusterId:   "cluster2",
			GlobalCIDRs: []string{"169.254.0.0/17"},
		}
		globalNetwork2 := GlobalNetwork{
			ClusterId:   "cluster3",
			GlobalCIDRs: []string{"169.254.128.0/17"},
		}
		globalNetworks[globalNetwork1.ClusterId] = &globalNetwork1
		globalNetworks[globalNetwork2.ClusterId] = &globalNetwork2
		result, err := AllocateGlobalCIDR(globalNetworks, &subctlData)
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
		It("Should not allocate any CIDR", func() {
			Expect(result).To(Equal(""))
		})
	})

	When("Not enough space for new cluster", func() {
		var globalNetworks = make(map[string]*GlobalNetwork)
		globalNetwork1 := GlobalNetwork{
			ClusterId:   "cluster2",
			GlobalCIDRs: []string{"169.254.0.0/18"},
		}
		globalNetwork2 := GlobalNetwork{
			ClusterId:   "cluster3",
			GlobalCIDRs: []string{"169.254.128.0/17"},
		}
		globalNetworks[globalNetwork1.ClusterId] = &globalNetwork1
		globalNetworks[globalNetwork2.ClusterId] = &globalNetwork2
		result, err := AllocateGlobalCIDR(globalNetworks, &subctlData)
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
		It("Should not allocate any CIDR", func() {
			Expect(result).To(Equal(""))
		})
	})
})
