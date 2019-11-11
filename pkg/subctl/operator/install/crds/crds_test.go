package crds

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

var _ = Describe("updateOrCreateCRD", func() {

	var (
		crd    *apiextensionsv1beta1.CustomResourceDefinition
		client *fake.Clientset
	)
	BeforeEach(func() {
		var err error
		crd, err = getSubmarinerCRD()
		Expect(err).ShouldNot(HaveOccurred())
		client = fake.NewSimpleClientset()
	})
	When("When called", func() {
		It("Should add the CRD properly", func() {
			created, err := updateOrCreateCRD(client, crd)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdCrd, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, v1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdCrd.Spec.Names.Kind).Should(Equal("Submariner"))
		})
	})

	When("When called twice", func() {
		It("Should add the CRD properly, and return false on second call", func() {
			created, err := updateOrCreateCRD(client, crd)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = updateOrCreateCRD(client, crd)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func TestOperatorCRDs(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Operator CRDs handling")
}
