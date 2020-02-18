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
)

var _ = Describe("IsOverlappingCidr", func() {
	When("There are no base CIDRs", func() {
		overlapping, err := IsOverlappingCIDR([]string{}, "10.10.10.0/24")
		It("Should return false", func() {
			Expect(overlapping).To(BeFalse())
		})
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("At least one Base CIDR is superset of new CIDR", func() {
		overlapping, err := IsOverlappingCIDR([]string{"10.10.0.0/16", "10.20.30.0/24"}, "10.10.10.0/24")
		It("Should return true", func() {
			Expect(overlapping).To(BeTrue())
		})
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("At least one Base CIDR is subset of new CIDR", func() {
		overlapping, err := IsOverlappingCIDR([]string{"10.10.10.0/24", "10.10.20.0/24"}, "10.10.30.0/16")
		It("Should return true", func() {
			Expect(overlapping).To(BeTrue())
		})
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("At least one Base CIDR is subset of new CIDR", func() {
		overlapping, err := IsOverlappingCIDR([]string{"10.10.0.0/16", "10.20.30.0/24"}, "10.10.10.0/24")
		It("Should return true", func() {
			Expect(overlapping).To(BeTrue())
		})
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("New CIDR partially overlaps with at least one Base CIDR", func() {
		overlapping, err := IsOverlappingCIDR([]string{"10.10.10.0/24"}, "10.10.10.128/22")
		It("Should return true", func() {
			Expect(overlapping).To(BeTrue())
		})
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("New CIDR lies between any two Base CIDR", func() {
		overlapping, err := IsOverlappingCIDR([]string{"10.10.10.0/24", "10.10.30.0/24"}, "10.10.20.128/24")
		It("Should return false", func() {
			Expect(overlapping).To(BeFalse())
		})
		It("Should not return error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
	})

	When("At least one Base CIDR is invalid", func() {
		_, err := IsOverlappingCIDR([]string{"10.10.10.0/33", "10.10.30.0/24"}, "10.10.20.128/24")
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

	When("New CIDR is invalid", func() {
		_, err := IsOverlappingCIDR([]string{"10.10.10.0/24", "10.10.30.0/24"}, "10.10.20.300/24")
		It("Should return error", func() {
			Expect(err).To(HaveOccurred())
		})
	})

})
