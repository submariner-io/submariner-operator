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
	"testing"

	. "github.com/onsi/gomega"
)

func TestMergeLighthouseIntoCorefile(t *testing.T) {
	g := NewWithT(t)

	existing := `# User config
example.com:53 {
    forward . 1.2.3.4
}
clusterset.local:53 {
    forward . 10.0.0.1
}
`

	merged := mergeLighthouseIntoCorefile(existing, []string{"clusterset.local"}, "10.96.1.2")
	g.Expect(merged).To(ContainSubstring("#lighthouse-start"))
	g.Expect(merged).To(ContainSubstring("forward . 10.96.1.2"))
	g.Expect(merged).To(ContainSubstring("example.com:53"))
	g.Expect(merged).NotTo(ContainSubstring("10.0.0.1"))
	g.Expect(merged).NotTo(ContainSubstring("forward . 10.0.0.1"))

	updated := mergeLighthouseIntoCorefile(merged, []string{"clusterset.local"}, "10.96.9.9")
	g.Expect(updated).To(ContainSubstring("forward . 10.96.9.9"))
	g.Expect(updated).NotTo(ContainSubstring("10.96.1.2"))
	g.Expect(updated).To(ContainSubstring("example.com:53"))

	cleaned := mergeLighthouseIntoCorefile(updated, []string{"clusterset.local"}, "")
	g.Expect(cleaned).NotTo(ContainSubstring("lighthouse-start"))
	g.Expect(cleaned).NotTo(ContainSubstring("clusterset.local"))
	g.Expect(cleaned).To(ContainSubstring("example.com:53"))
}
