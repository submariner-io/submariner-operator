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

package crds

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("getKubeFedCRDs", func() {
	When("When called", func() {
		It("Should parse the embedded yaml properly", func() {
			_, err := getKubeFedCRDs()
			Expect(err).ShouldNot(HaveOccurred())
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
		crds, err := getKubeFedCRDs()
		Expect(err).ShouldNot(HaveOccurred())
		crd = crds[0]
		client = fake.NewSimpleClientset()
	})
	When("When called", func() {
		It("Should add the CRD properly", func() {
			created, err := updateOrCreateCRD(client, crd)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdCrd, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, v1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdCrd.Spec.Names.Kind).Should(Equal("ClusterPropagatedVersion"))
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
