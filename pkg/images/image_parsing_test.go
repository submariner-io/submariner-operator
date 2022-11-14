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

package images_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apis "github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/images"
)

var imageTests = []struct {
	image      string
	repository string
	version    string
}{
	{"localhost:5000/submariner-operator:local", "localhost:5000", "local"},
	{"some-other-registry.com:1235/submariner-org/submariner-operator:0.5.0", "some-other-registry.com:1235/submariner-org", "0.5.0"},
	{"submariner-org/submariner-operator:0.4.0", "submariner-org", "0.4.0"},
	{"quay.io/submariner/submariner-operator:local", "quay.io/submariner", "local"},
	{"any.reg/subm-tech-preview/submariner-custom-operator:0.8.0", "any.reg/subm-tech-preview", "0.8.0"},
	{"submariner-operator:0.8.1", "", "0.8.1"},
	{"submariner-operator", "", apis.DefaultSubmarinerOperatorVersion},
}

var _ = Describe("image parsing", func() {
	When("Parsing image", func() {
		It("Should parse version and repository", func() {
			_, rep := images.ParseOperatorImage("localhost:5000/submariner-operator:local")
			Expect(rep).To(Equal("localhost:5000"))

			for _, tt := range imageTests {
				version, repository := images.ParseOperatorImage(tt.image)
				Expect(repository).To(Equal(tt.repository))
				Expect(version).To(Equal(tt.version))
			}
		})
	})
})

func TestParseOperatorImage(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "image parsing")
}
