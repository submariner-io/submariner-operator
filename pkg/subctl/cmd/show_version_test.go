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

package cmd

import (
	"strings"
	"testing"
)

var imageTests = []struct {
	image      string
	repository string
	version    string
}{
	{"localhost:5000/submariner-operator:local", "localhost:5000", "local"},
	{"some-other-registry.com:1235/submariner-org/submariner-operator:v0.5.0", "some-other-registry.com:1235/submariner-org", "v0.5.0"},
	{"submariner-org/submariner-operator:v0.4.0", "submariner-org", "v0.4.0"},
	{"quay.io/submariner/submariner-operator:local", "quay.io/submariner", "local"},
}

func TestParseOperatorImage(t *testing.T) {
	for _, tt := range imageTests {
		version, repository := parseOperatorImage(tt.image)
		if strings.Compare(repository, tt.repository) != 0 {
			t.Fatalf("Operator repository for image %q shold be %q, but value parsed was %q\n", tt.image, tt.repository, repository)
		}
		if strings.Compare(version, tt.version) != 0 {
			t.Fatalf("Operator version for image %q should be %q, but value parsed was %q\n", tt.image, tt.version, version)
		}
	}
}
