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

	When("the id contains non-alphanumeric characters other than dashes", func() {
		It("should not be valid", func() {
			Expect(cluster.IsValidID("abcdéfg")).To(Not(Succeed()))
			Expect(cluster.IsValidID("abcde.g")).To(Not(Succeed()))
		})
		It("should convert invalid characters to dashes", func() {
			Expect(cluster.SanitizeClusterID("abcdéfg")).To(Equal("abcd-fg"))
			Expect(cluster.SanitizeClusterID("abcde.g")).To(Equal("abcde-g"))
		})
	})

	When("the id starts or end with a dash", func() {
		It("should not be valid", func() {
			Expect(cluster.IsValidID("-abcdef")).To(Not(Succeed()))
			Expect(cluster.IsValidID("abcdef-")).To(Not(Succeed()))
		})
		It("should replace dash with 0", func() {
			Expect(cluster.SanitizeClusterID("-abcdef")).To(Equal("0abcdef"))
			Expect(cluster.SanitizeClusterID("abcdef-")).To(Equal("abcdef0"))
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
		It("should return empty string", func() {
			Expect(cluster.SanitizeClusterID("")).To(Equal(""))
		})
	})
})

func TestCluster(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Cluster suite")
}
