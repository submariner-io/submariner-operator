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
package servicediscovery_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Service discovery controller", func() {
	Context("Reconciliation", testReconciliation)
})

func testReconciliation() {
	t := newTestDriver()

	When("the lighthouse DNS service IP is updated", func() {
		updatedClusterIP := "10.10.10.11"

		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, newDNSConfig(clusterIP), newDNSService(updatedClusterIP))
		})

		It("should update the the DNS config", func() {
			t.assertReconcileSuccess()

			Expect(t.assertDNSConfig().Spec).To(Equal(newDNSConfig(updatedClusterIP).Spec))
		})
	})

	When("the lighthouse DNS service IP is not updated", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, newDNSConfig(clusterIP), newDNSService(clusterIP))
		})

		It("should not update the the DNS config", func() {
			t.assertReconcileSuccess()

			Expect(t.assertDNSConfig().Spec).To(Equal(newDNSConfig(clusterIP).Spec))
		})
	})

	When("the lighthouse clusterIP is not configured", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, newDNSService(clusterIP))
			t.createConfigMap(newConfigMap(""))
		})

		It("should update the coreDNS config map", func() {
			t.assertReconcileSuccess()

			Expect(t.assertCoreDNSConfigMap().Data["Corefile"]).To(ContainSubstring(clusterlocalConfig + clusterIP + "\n}"))
			Expect(t.assertCoreDNSConfigMap().Data["Corefile"]).To(ContainSubstring(superClusterlocalConfig + clusterIP + "\n}"))
		})
	})

	When("the lighthouse clusterIP is already configured", func() {
		updatedClusterIP := "10.10.10.11"

		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, newDNSService(updatedClusterIP))
			t.createConfigMap(newConfigMap(clusterlocalConfig + clusterIP + "\n}" + superClusterlocalConfig + clusterIP + "\n}"))
		})

		It("should update the coreDNS config map", func() {
			t.assertReconcileSuccess()

			Expect(t.assertCoreDNSConfigMap().Data["Corefile"]).To(ContainSubstring(clusterlocalConfig + updatedClusterIP))
			Expect(t.assertCoreDNSConfigMap().Data["Corefile"]).To(ContainSubstring(superClusterlocalConfig + updatedClusterIP))
		})
	})
}
