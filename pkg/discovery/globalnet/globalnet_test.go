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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const namespace = "test-ns"

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
			ClusterID:  "west",
			GlobalCIDR: globalnetCIDR,
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

var _ = Describe("AllocateAndUpdateGlobalCIDRConfigMap", func() {
	var client controllerClient.Client

	BeforeEach(func() {
		client = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
		Expect(globalnet.CreateConfigMap(context.Background(), client, true, "168.254.0.0/16",
			8192, namespace)).To(Succeed())
	})

	When("the globalnet CIDR is not specified", func() {
		const expGlobalCIDR = "168.254.0.0/19"

		It("should allocate a new one", func() {
			netconfig := &globalnet.Config{
				ClusterID: "east",
			}

			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(context.Background(), client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal(expGlobalCIDR))

			globalnetInfo, _, err := globalnet.GetGlobalNetworks(context.Background(), client, namespace)
			Expect(err).To(Succeed())
			Expect(globalnetInfo.CidrInfo).To(HaveKeyWithValue(netconfig.ClusterID, &globalnet.GlobalNetwork{
				GlobalCIDRs: []string{expGlobalCIDR},
				ClusterID:   netconfig.ClusterID,
			}))

			netconfig.GlobalCIDR = ""
			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(context.Background(), client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal(expGlobalCIDR))
		})
	})

	When("the globalnet CIDR is specified", func() {
		const expGlobalCIDR = "168.254.0.0/15"

		It("should not allocate a new one", func() {
			netconfig := &globalnet.Config{
				ClusterID:  "east",
				GlobalCIDR: expGlobalCIDR,
			}

			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(context.Background(), client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal(expGlobalCIDR))

			globalnetInfo, _, err := globalnet.GetGlobalNetworks(context.Background(), client, namespace)
			Expect(err).To(Succeed())
			Expect(globalnetInfo.CidrInfo).To(HaveKeyWithValue(netconfig.ClusterID, &globalnet.GlobalNetwork{
				GlobalCIDRs: []string{expGlobalCIDR},
				ClusterID:   netconfig.ClusterID,
			}))

			netconfig.GlobalCIDR = ""
			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(context.Background(), client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal(expGlobalCIDR))
		})
	})

	When("the globalnet cluster size is specified", func() {
		It("should allocate a CIDR", func() {
			netconfig := &globalnet.Config{
				ClusterID:   "east",
				ClusterSize: 4096,
			}

			Expect(globalnet.AllocateAndUpdateGlobalCIDRConfigMap(context.Background(), client, namespace,
				netconfig, reporter.Klog())).To(Succeed())
			Expect(netconfig.GlobalCIDR).To(Equal("168.254.0.0/20"))
		})
	})
})

var _ = Describe("ValidateExistingGlobalNetworks", func() {
	var client controllerClient.Client

	BeforeEach(func() {
		client = fake.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	})

	When("the existing globalnet config is valid", func() {
		It("should succeed", func() {
			Expect(globalnet.CreateConfigMap(context.Background(), client, true, globalnet.DefaultGlobalnetCIDR,
				globalnet.DefaultGlobalnetClusterSize, namespace)).To(Succeed())

			Expect(globalnet.ValidateExistingGlobalNetworks(context.Background(), client, namespace)).To(Succeed())
		})
	})

	When("the globalnet config does not exist", func() {
		It("should succeed", func() {
			Expect(globalnet.ValidateExistingGlobalNetworks(context.Background(), client, namespace)).To(Succeed())
		})
	})

	When("the existing globalnet CIDR is invalid", func() {
		It("should return an error", func() {
			Expect(globalnet.CreateConfigMap(context.Background(), client, true, "169.254.0.0/16",
				globalnet.DefaultGlobalnetClusterSize, namespace)).To(Succeed())

			Expect(globalnet.ValidateExistingGlobalNetworks(context.Background(), client, namespace)).ToNot(Succeed())
		})
	})
})
