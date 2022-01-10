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

package clusterrole_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/clusterrole"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

const (
	clusterRoleYAML = `
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: test-clusterrole
rules:
  - apiGroups:
      - ""
    resources:
      - pods
    verbs:
      - get
`
)

var _ = Describe("EnsureFromYAML", func() {
	var client *fakeclientset.Clientset

	BeforeEach(func() {
		client = fakeclientset.NewSimpleClientset()
	})

	assertClusterRole := func() {
		r, err := client.RbacV1().ClusterRoles().Get(context.TODO(), "test-clusterrole", metav1.GetOptions{})
		Expect(err).To(Succeed())
		Expect(r.Rules).To(HaveLen(1))
		Expect(r.Rules[0].APIGroups).To(Equal([]string{""}))
		Expect(r.Rules[0].Verbs).To(Equal([]string{"get"}))
		Expect(r.Rules[0].Resources).To(Equal([]string{"pods"}))
	}

	When("the ClusterRole doesn't exist", func() {
		It("should create it", func() {
			created, err := clusterrole.EnsureFromYAML(client, clusterRoleYAML)
			Expect(created).To(BeTrue())
			Expect(err).To(Succeed())
			assertClusterRole()
		})
	})

	When("the ClusterRole already exists", func() {
		It("should not update it", func() {
			_, err := clusterrole.Ensure(client, &rbacv1.ClusterRole{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ClusterRole",
					APIVersion: rbacv1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-clusterrole",
				},
				Rules: []rbacv1.PolicyRule{
					{
						Verbs:     []string{"get"},
						APIGroups: []string{""},
						Resources: []string{"pods"},
					},
				},
			})
			Expect(err).To(Succeed())
			assertClusterRole()

			created, err := clusterrole.EnsureFromYAML(client, clusterRoleYAML)
			Expect(created).To(BeFalse())
			Expect(err).To(Succeed())
		})
	})
})
