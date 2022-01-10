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

package utils_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("CreateOrUpdateClusterRoleBinding", func() {
	var (
		clusterRoleBinding *rbacv1.ClusterRoleBinding
		client             *fakeclientset.Clientset
		ctx                context.Context
	)

	BeforeEach(func() {
		clusterRoleBinding = &rbacv1.ClusterRoleBinding{}
		// TODO skitt add our own object
		err := embeddedyamls.GetObject(embeddedyamls.Config_rbac_submariner_globalnet_cluster_role_binding_yaml, clusterRoleBinding)
		Expect(err).ShouldNot(HaveOccurred())
		client = fakeclientset.NewSimpleClientset()
		ctx = context.TODO()
	})

	When("called", func() {
		It("Should add the ClusterRoleBinding properly", func() {
			created, err := utils.CreateOrUpdateClusterRoleBinding(ctx, client, clusterRoleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdClusterRoleBinding, err := client.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdClusterRoleBinding.ObjectMeta.Name).Should(Equal("submariner-globalnet"))
		})
	})

	When("called twice", func() {
		It("Should add the ClusterRoleBinding properly, and return false on second call", func() {
			created, err := utils.CreateOrUpdateClusterRoleBinding(ctx, client, clusterRoleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = utils.CreateOrUpdateClusterRoleBinding(ctx, client, clusterRoleBinding)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("CreateOrUpdateDeployment", func() {
	var (
		namespace  = "test-namespace"
		name       = "test-deployment"
		deployment *appsv1.Deployment
		client     *fakeclientset.Clientset
		ctx        context.Context
	)

	BeforeEach(func() {
		replicas := int32(1)
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: &replicas,
				Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"name": name}},
			},
		}
		client = fakeclientset.NewSimpleClientset()
		ctx = context.TODO()
	})

	When("called", func() {
		It("Should add the Deployment properly", func() {
			created, err := utils.CreateOrUpdateDeployment(ctx, client, namespace, deployment)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdDeployment, err := client.AppsV1().Deployments(namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdDeployment.ObjectMeta.Name).Should(Equal(name))
		})
	})

	When("called twice", func() {
		It("Should add the Deployment properly, and return false on second call", func() {
			created, err := utils.CreateOrUpdateDeployment(ctx, client, namespace, deployment)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = utils.CreateOrUpdateDeployment(ctx, client, namespace, deployment)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func TestCreateOrUpdate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create or update handling")
}
