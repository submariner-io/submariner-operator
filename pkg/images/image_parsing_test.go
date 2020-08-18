package images

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
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

var _ = Describe("image parsing", func() {
	When("Parsing image", func() {
		It("Should parse version and repository", func() {
			_, rep := ParseOperatorImage("localhost:5000/submariner-operator:local")
			Expect(rep).To(Equal("localhost:5000"))

			for _, tt := range imageTests {
				version, repository := ParseOperatorImage(tt.image)
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
