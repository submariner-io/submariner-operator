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

package cluster_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/internal/cluster"
)

var _ = Describe("TestClusterIDs", func() {
	Describe("IsValidID", testIsValidID)
	Describe("SanitizeID", testSanitizeID)
})

func testIsValidID() {
	When("the id only contains alphabetic characters", func() {
		It("should be valid", func() {
			Expect(cluster.IsValidID("abcdef")).To(Succeed())
		})
	})

	When("the id only contains numeric characters", func() {
		It("should be valid", func() {
			Expect(cluster.IsValidID("012345")).To(Succeed())
		})
	})

	When("the id only contains alphanumeric characters", func() {
		It("should be valid", func() {
			Expect(cluster.IsValidID("a0b1c2d3ef")).To(Succeed())
			Expect(cluster.IsValidID("0a1b2c3def")).To(Succeed())
			Expect(cluster.IsValidID("a0b1c2d3e4f5")).To(Succeed())
			Expect(cluster.IsValidID("0a1b2c3d4e5f6")).To(Succeed())
		})
	})

	When("the id only contains alphanumeric characters and dashes", func() {
		It("should be valid", func() {
			Expect(cluster.IsValidID("a0b-1c2d3ef")).To(Succeed())
			Expect(cluster.IsValidID("0a1b2c3-def")).To(Succeed())
			Expect(cluster.IsValidID("a0b1c2d3---e4f5")).To(Succeed())
		})
	})

	When("the id contains uppercase alphabetic characters", func() {
		It("should not be valid", func() {
			Expect(cluster.IsValidID("A1b2c3d4e5f6")).To(Not(Succeed()))
			Expect(cluster.IsValidID("1A2b3c4d5e6f")).To(Not(Succeed()))
			Expect(cluster.IsValidID("a1-B2-c3-D4")).To(Not(Succeed()))
		})
	})

	When("the id contains non-alphanumeric characters other than dashes", func() {
		It("should not be valid", func() {
			Expect(cluster.IsValidID("abcdéfg")).To(Not(Succeed()))
			Expect(cluster.IsValidID("abcde.g")).To(Not(Succeed()))
		})
	})

	When("the id starts or end with a dash", func() {
		It("should not be valid", func() {
			Expect(cluster.IsValidID("-abcdef")).To(Not(Succeed()))
			Expect(cluster.IsValidID("abcdef-")).To(Not(Succeed()))
		})
	})

	When("the id is longer than 63 characters", func() {
		It("should not be valid", func() {
			Expect(cluster.IsValidID("012345678901234567890123456789012345678901234567890123456789012")).To(Succeed())
			Expect(cluster.IsValidID("0123456789012345678901234567890123456789012345678901234567890123")).To(Not(Succeed()))
			Expect(cluster.IsValidID("0123456789012345678901234567890123456789012345678901234567890123456789")).To(Not(Succeed()))
		})
	})

	When("the id is empty", func() {
		It("should not be valid", func() {
			Expect(cluster.IsValidID("")).To(Not(Succeed()))
		})
	})
}

func testSanitizeID() {
	When("the id only contains alphabetic characters", func() {
		It("should return same id", func() {
			Expect(cluster.SanitizeID("abcdef")).To(Equal("abcdef"))
		})
	})

	When("the id only contains numeric characters", func() {
		It("should return same id", func() {
			Expect(cluster.IsValidID("012345")).To(Succeed())
			Expect(cluster.SanitizeID("012345")).To(Equal("012345"))
		})
	})

	When("the id only contains alphanumeric characters", func() {
		It("should return same id", func() {
			Expect(cluster.SanitizeID("a0b1c2d3ef")).To(Equal("a0b1c2d3ef"))
			Expect(cluster.SanitizeID("0a1b2c3def")).To(Equal("0a1b2c3def"))
			Expect(cluster.SanitizeID("a0b1c2d3e4f5")).To(Equal("a0b1c2d3e4f5"))
			Expect(cluster.SanitizeID("0a1b2c3d4e5f6")).To(Equal("0a1b2c3d4e5f6"))
		})
	})

	When("the id only contains alphanumeric characters and dashes", func() {
		It("should return same id valid", func() {
			Expect(cluster.SanitizeID("a0b-1c2d3ef")).To(Equal("a0b-1c2d3ef"))
			Expect(cluster.SanitizeID("0a1b2c3-def")).To(Equal("0a1b2c3-def"))
			Expect(cluster.SanitizeID("a0b1c2d3---e4f5")).To(Equal("a0b1c2d3---e4f5"))
		})
	})

	When("the id contains uppercase alphabetic characters", func() {
		It("should convert lowercase characters to uppercase", func() {
			Expect(cluster.SanitizeID("A1b2c3d4e5f6")).To(Equal("a1b2c3d4e5f6"))
			Expect(cluster.SanitizeID("1A2b3c4d5e6f")).To(Equal("1a2b3c4d5e6f"))
			Expect(cluster.SanitizeID("a1-B2-c3-D4")).To(Equal("a1-b2-c3-d4"))
		})
	})

	When("the id contains non-alphanumeric characters other than dashes", func() {
		It("should convert invalid characters to dashes", func() {
			Expect(cluster.SanitizeID("abcdéfg")).To(Equal("abcd-fg"))
			Expect(cluster.SanitizeID("abcde.g")).To(Equal("abcde-g"))
		})
	})

	When("the id starts or end with a dash", func() {
		It("should replace dash with 0", func() {
			Expect(cluster.SanitizeID("-abcdef")).To(Equal("0abcdef"))
			Expect(cluster.SanitizeID("abcdef-")).To(Equal("abcdef0"))
		})
	})

	When("the id starts or end with non alphanumeric character", func() {
		It("should replace non alphanumeric character with 0", func() {
			Expect(cluster.SanitizeID("éabcdef")).To(Equal("0abcdef"))
			Expect(cluster.SanitizeID("abcdefé")).To(Equal("abcdef0"))
		})
	})

	When("the id is longer than 63 characters", func() {
		It("should return same id", func() {
			testID := "0123456789012345678901234567890123456789012345678901234567890123"
			Expect(cluster.SanitizeID(testID)).To(Equal(testID))

			testID = "0123456789012345678901234567890123456789012345678901234567890123456789"
			Expect(cluster.SanitizeID(testID)).To(Equal(testID))
		})
	})

	When("the id is empty", func() {
		It("should return empty string", func() {
			Expect(cluster.SanitizeID("")).To(Equal(""))
		})
	})
}

func TestCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster suite")
}
