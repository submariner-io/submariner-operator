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
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("removeLighthouseSection", func() {
	const (
		section = `#lighthouse-start AUTO-GENERATED SECTION. DO NOT EDIT
clusterset.local:53 {
    forward . 10.96.0.10
}
#lighthouse-end
`
		corefile = `.:53 {
    errors
    health
    kubernetes cluster.local in-addr.arpa ip6.arpa {
        pods insecure
    }
    forward . /etc/resolv.conf
    cache 30
}

example.org:53 {
    forward . 1.1.1.1
}
`
	)

	When("there is no section", func() {
		It("should return the Corefile unchanged", func() {
			Expect(removeLighthouseSection(corefile)).To(Equal(corefile))
		})
	})

	When("there is a section", func() {
		It("should return the Corefile without it", func() {
			Expect(removeLighthouseSection(section + corefile)).To(Equal(corefile))
		})
	})

	When("the markers are indented", func() {
		It("should return the Corefile without the section", func() {
			indented := "    " + strings.ReplaceAll(strings.TrimSuffix(section, "\n"), "\n", "\n    ") + "\n"
			Expect(removeLighthouseSection(indented + corefile)).To(Equal(corefile))
		})
	})

	When("a line only mentions a marker", func() {
		It("should return the Corefile unchanged", func() {
			mention := "# the #lighthouse-start block below is managed by Submariner\n"
			Expect(removeLighthouseSection(mention + corefile)).To(Equal(mention + corefile))
		})
	})

	When("a longer word begins with the start marker text", func() {
		It("should return the Corefile unchanged", func() {
			nearMiss := "#lighthouse-started is not a marker\n"
			Expect(removeLighthouseSection(nearMiss + corefile)).To(Equal(nearMiss + corefile))
		})
	})

	When("the end marker is missing and a longer word begins with its text", func() {
		It("should return an error", func() {
			opened := strings.Replace(section, "#lighthouse-end\n", "", 1)

			_, err := removeLighthouseSection(opened + "#lighthouse-endless zone follows\n" + corefile)
			Expect(err).To(MatchError(errUnpairedLighthouseMarkers))
		})
	})

	When("a start marker has no end marker", func() {
		It("should return an error", func() {
			_, err := removeLighthouseSection(strings.Replace(section, "#lighthouse-end\n", "", 1) + corefile)
			Expect(err).To(MatchError(errUnpairedLighthouseMarkers))
		})
	})

	When("an end marker has no start marker", func() {
		It("should return an error", func() {
			_, err := removeLighthouseSection("#lighthouse-end\n" + corefile)
			Expect(err).To(MatchError(errUnpairedLighthouseMarkers))
		})
	})

	When("a section is not terminated before the next one starts", func() {
		It("should return an error", func() {
			_, err := removeLighthouseSection(strings.Replace(section, "#lighthouse-end\n", "", 1) + section + corefile)
			Expect(err).To(MatchError(errUnpairedLighthouseMarkers))
		})
	})
})
