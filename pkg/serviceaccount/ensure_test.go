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

package serviceaccount_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/serviceaccount"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

const (
	roleYAML = `
apiVersion: v1
kind: ServiceAccount
metadata:
  name: test-sa
`
)

var _ = Describe("EnsureFromYAML", func() {
	const namespace = "test-namespace"

	var client *fakeclientset.Clientset

	BeforeEach(func() {
		client = fakeclientset.NewSimpleClientset()
	})

	assertServiceAccount := func() {
		_, err := client.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), "test-sa", metav1.GetOptions{})
		Expect(err).To(Succeed())
	}

	When("the ServiceAccount doesn't exist", func() {
		It("should create it", func() {
			created, err := serviceaccount.EnsureFromYAML(client, namespace, roleYAML)
			Expect(created).To(BeTrue())
			Expect(err).To(Succeed())
			assertServiceAccount()
		})
	})

	When("the ServiceAccount already exists", func() {
		It("should not update it", func() {
			_, err := serviceaccount.Ensure(client, namespace, &corev1.ServiceAccount{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ServiceAccount",
					APIVersion: corev1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-sa",
				},
			})
			Expect(err).To(Succeed())
			assertServiceAccount()

			created, err := serviceaccount.EnsureFromYAML(client, namespace, roleYAML)
			Expect(created).To(BeFalse())
			Expect(err).To(Succeed())
		})
	})
})
