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

package crd_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/crd"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extendedfakeclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	crdYAML = `
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: submariners.submariner.io
spec:
  group: submariner.io
  names:
    kind: Submariner
`
)

var _ = Describe("Updater", func() {
	var (
		client  *extendedfakeclientset.Clientset
		updater crd.Updater
	)

	BeforeEach(func() {
		client = extendedfakeclientset.NewSimpleClientset()
		updater = crd.UpdaterFromClientSet(client)
	})

	assertCRDExists := func(name string) {
		crd, err := updater.Get(context.TODO(), name, metav1.GetOptions{})
		Expect(err).To(Succeed())
		Expect(crd.Spec.Names.Kind).Should(Equal("Submariner"))
	}

	Context("on CreateOrUpdate", func() {
		crd := &apiextensions.CustomResourceDefinition{
			TypeMeta: metav1.TypeMeta{
				Kind:       "CustomResourceDefinition",
				APIVersion: apiextensions.SchemeGroupVersion.String(),
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "submariners.submariner.io",
			},
			Spec: apiextensions.CustomResourceDefinitionSpec{
				Group: "submariner.io",
				Names: apiextensions.CustomResourceDefinitionNames{
					Kind: "Submariner",
				},
			},
		}

		When("the CRD doesn't exist", func() {
			It("should create it", func() {
				created, err := updater.CreateOrUpdate(context.TODO(), crdYAML)
				Expect(created).To(BeTrue())
				Expect(err).To(Succeed())
				assertCRDExists(crd.Name)
			})
		})

		When("the CRD already exists", func() {
			It("should not update it", func() {
				_, err := updater.Create(context.TODO(), crd, metav1.CreateOptions{})
				Expect(err).To(Succeed())
				assertCRDExists(crd.Name)

				created, err := updater.CreateOrUpdate(context.TODO(), crdYAML)
				Expect(created).To(BeFalse())
				Expect(err).To(Succeed())

				actualActions := client.Actions()
				for i := range actualActions {
					Expect(actualActions[i].GetVerb()).ToNot(Equal("update"))
				}
			})
		})
	})
})
