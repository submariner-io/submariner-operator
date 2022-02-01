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

package rolebinding_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/rolebinding"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

const (
	roleBindingYAML = `
kind: RoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: test-rolebinding
subjects:
  - kind: ServiceAccount
    name: test-sa
roleRef:
  kind: Role
  name: test-role
  apiGroup: rbac.authorization.k8s.io
`
)

var _ = Describe("EnsureFromYAML", func() {
	const namespace = "test-namespace"

	var client *fakeclientset.Clientset

	BeforeEach(func() {
		client = fakeclientset.NewSimpleClientset()
	})

	assertRoleBinding := func() {
		r, err := client.RbacV1().RoleBindings(namespace).Get(context.TODO(), "test-rolebinding", metav1.GetOptions{})
		Expect(err).To(Succeed())
		Expect(r.RoleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
		Expect(r.RoleRef.Name).To(Equal("test-role"))
		Expect(r.RoleRef.Kind).To(Equal("Role"))
		Expect(r.Subjects).To(HaveLen(1))
		Expect(r.Subjects[0].Kind).To(Equal("ServiceAccount"))
		Expect(r.Subjects[0].Name).To(Equal("test-sa"))
	}

	When("the RoleBinding doesn't exist", func() {
		It("should create it", func() {
			created, err := rolebinding.EnsureFromYAML(client, namespace, roleBindingYAML)
			Expect(created).To(BeTrue())
			Expect(err).To(Succeed())
			assertRoleBinding()
		})
	})

	When("the RoleBinding already exists", func() {
		It("should not update it", func() {
			_, err := rolebinding.Ensure(client, namespace, &rbacv1.RoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "RoleBinding",
					APIVersion: rbacv1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-rolebinding",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind: "ServiceAccount",
						Name: "test-sa",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "Role",
					Name:     "test-role",
				},
			})
			Expect(err).To(Succeed())
			assertRoleBinding()

			created, err := rolebinding.EnsureFromYAML(client, namespace, roleBindingYAML)
			Expect(created).To(BeFalse())
			Expect(err).To(Succeed())
		})
	})
})
