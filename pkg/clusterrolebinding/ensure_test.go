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

package clusterrolebinding_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/clusterrolebinding"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

const (
	clusterRoleBindingYAML = `
kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: test-clusterrolebinding
subjects:
  - kind: ServiceAccount
    name: test-sa
roleRef:
  kind: ClusterRole
  name: test-clusterrole
  apiGroup: rbac.authorization.k8s.io
`
)

var _ = Describe("EnsureFromYAML", func() {
	const namespace = "test-namespace"

	var client *fakeclientset.Clientset

	BeforeEach(func() {
		client = fakeclientset.NewSimpleClientset()
	})

	assertClusterRoleBinding := func() {
		r, err := client.RbacV1().ClusterRoleBindings().Get(context.TODO(), "test-clusterrolebinding", metav1.GetOptions{})
		Expect(err).To(Succeed())
		Expect(r.RoleRef.APIGroup).To(Equal("rbac.authorization.k8s.io"))
		Expect(r.RoleRef.Name).To(Equal("test-clusterrole"))
		Expect(r.RoleRef.Kind).To(Equal("ClusterRole"))
		Expect(r.Subjects).To(HaveLen(1))
		Expect(r.Subjects[0].Kind).To(Equal("ServiceAccount"))
		Expect(r.Subjects[0].Name).To(Equal("test-sa"))
	}

	When("the ClusterRoleBinding doesn't exist", func() {
		It("should create it", func() {
			created, err := clusterrolebinding.EnsureFromYAML(client, namespace, clusterRoleBindingYAML)
			Expect(created).To(BeTrue())
			Expect(err).To(Succeed())
			assertClusterRoleBinding()
		})
	})

	When("the ClusterRoleBinding already exists", func() {
		It("should not update it", func() {
			_, err := clusterrolebinding.Ensure(client, &rbacv1.ClusterRoleBinding{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRoleBinding",
					APIVersion: rbacv1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterrolebinding",
				},
				Subjects: []rbacv1.Subject{
					{
						Kind: "ServiceAccount",
						Name: "test-sa",
					},
				},
				RoleRef: rbacv1.RoleRef{
					APIGroup: "rbac.authorization.k8s.io",
					Kind:     "ClusterRole",
					Name:     "test-clusterrole",
				},
			})
			Expect(err).To(Succeed())
			assertClusterRoleBinding()

			created, err := clusterrolebinding.EnsureFromYAML(client, namespace, clusterRoleBindingYAML)
			Expect(created).To(BeFalse())
			Expect(err).To(Succeed())
		})
	})
})
