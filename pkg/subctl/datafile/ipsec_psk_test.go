/*
© 2019 Red Hat, Inc. and others.

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

package datafile

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("ipsec_psk handling", func() {
	When("generateRandonPSK is called", func() {
		It("should return the amount of entropy requested", func() {
			psk, err := generateRandomPSK(ipsecSecretLength)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(psk).To(HaveLen(ipsecSecretLength))
		})
	})

	When("NewBrokerPSKSecret is called", func() {
		It("should return a secret with a psk data inside", func() {
			secret, err := newIPSECPSKSecret()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(secret.Name).To(Equal("submariner-ipsec-psk"))
			Expect(secret.Data).To(HaveKey("psk"))
			Expect(secret.Data["psk"]).To(HaveLen(ipsecSecretLength))
		})
	})

})
