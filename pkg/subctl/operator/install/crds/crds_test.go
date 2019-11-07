package crds

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("getSubmarinerCRD", func() {
	When("When called", func() {
		It("Should parse the embedded yaml properly", func() {
			crd, err := getSubmarinerCRD()
			Expect(err).ShouldNot(HaveOccurred())
			Expect(crd.Spec.Names.Kind).Should(Equal("Submariner"))
			Expect(crd.Spec.Versions[0].Name).Should(Equal("v1alpha1"))
		})
	})

})

func TestOperatorCRDs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator CRDs handling")
}
