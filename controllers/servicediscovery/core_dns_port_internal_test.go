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

package servicediscovery

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("findCoreDNSListeningPort", func() {
	When("no port is found", func() {
		It("should return the default port", func() {
			Expect(findCoreDNSListeningPort("")).To(Equal(coreDNSDefaultPort))
		})
	})

	When("a non-standard port is found and the core config was already modified", func() {
		It("should return the port", func() {
			coreFileContents := `
                #lighthouse-start AUTO-GENERATED SECTION. DO NOT EDIT
                clusterset.local:123 {
                  forward . 10.43.188.46
                }
                #lighthouse-end
                .:456 {
                  errors
                  health
                  kubernetes cluster.local in-addr.arpa ip6.arpa {
                    pods insecure
                    upstream
                    fallthrough in-addr.arpa ip6.arpa
                  }
                  prometheus :9153
                  forward . /etc/resolv.conf {
                    policy sequential
                  }
                  cache 30
                  reload
               }`
			Expect(findCoreDNSListeningPort(coreFileContents)).To(Equal("456"))
		})
	})

	When("non-standard port is found and Coreconfig was not modified (OCP coreconfig)", func() {
		coreFileContents := `
            .:5353 {
              bufsize 1232
              errors
              health {
                lameduck 20s
              }
              ready
              kubernetes cluster.local in-addr.arpa ip6.arpa {
                pods insecure
                upstream
                fallthrough in-addr.arpa ip6.arpa
              }
              prometheus 127.0.0.1:9153
              forward . /etc/resolv.conf {
                policy sequential
              }
              cache 900 {
                denial 9984 30
              }
              reload
            }`

		Expect(findCoreDNSListeningPort(coreFileContents)).To(Equal("5353"))
	})

	When("standard port is found and Coreconfig was not modified (kind Coreconfig)", func() {
		coreFileContents := `
            .:53 {
              errors
              health {
                lameduck 5s
              }
              ready
              kubernetes cluster1.local in-addr.arpa ip6.arpa {
                pods insecure
                fallthrough in-addr.arpa ip6.arpa
                ttl 30
              }
              prometheus :9153
              forward . /etc/resolv.conf {
                max_concurrent 1000
              }
              cache 30
              loop
              reload
              loadbalance
            }`

		Expect(findCoreDNSListeningPort(coreFileContents)).To(Equal("53"))
	})
})
