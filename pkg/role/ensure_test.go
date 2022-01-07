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

package role_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/role"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

const (
	roleYAML = `
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: test-role
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - '*'
`
)

var _ = Describe("EnsureFromYAML", func() {
	const namespace = "test-namespace"

	var client *fakeclientset.Clientset

	BeforeEach(func() {
		client = fakeclientset.NewSimpleClientset()
	})

	assertRole := func() {
		r, err := client.RbacV1().Roles(namespace).Get(context.TODO(), "test-role", metav1.GetOptions{})
		Expect(err).To(Succeed())
		Expect(r.Rules).To(HaveLen(1))
		Expect(r.Rules[0].APIGroups).To(Equal([]string{""}))
		Expect(r.Rules[0].Verbs).To(Equal([]string{"*"}))
		Expect(r.Rules[0].Resources).To(Equal([]string{"pods"}))
	}

	When("the Role doesn't exist", func() {
		It("should create it", func() {
			created, err := role.EnsureFromYAML(client, namespace, roleYAML)
			Expect(created).To(BeTrue())
			Expect(err).To(Succeed())
			assertRole()
		})
	})

	When("the Role already exists", func() {
		It("should not update it", func() {
			_, err := role.Ensure(client, namespace, &rbacv1.Role{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Role",
					APIVersion: rbacv1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-role",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"*"},
						APIGroups: []string{""},
						Resources: []string{"pods"},
					},
				},
			})
			Expect(err).To(Succeed())
			assertRole()

			created, err := role.EnsureFromYAML(client, namespace, roleYAML)
			Expect(created).To(BeFalse())
			Expect(err).To(Succeed())
		})
	})
})
