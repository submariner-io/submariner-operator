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

// nolint:dupl // The test cases are similar but not duplicated.
package utils

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	extendedfakeclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("CreateOrUpdateClusterRole", func() {
	var (
		clusterRole *rbacv1.ClusterRole
		client      *fakeclientset.Clientset
		ctx         context.Context
	)

	BeforeEach(func() {
		clusterRole = &rbacv1.ClusterRole{}
		// TODO skitt add our own object
		err := embeddedyamls.GetObject(embeddedyamls.Config_rbac_submariner_globalnet_cluster_role_yaml, clusterRole)
		Expect(err).ShouldNot(HaveOccurred())
		client = fakeclientset.NewSimpleClientset()
		ctx = context.TODO()
	})

	When("called", func() {
		It("Should add the ClusterRole properly", func() {
			created, err := CreateOrUpdateClusterRole(ctx, client, clusterRole)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdClusterRole, err := client.RbacV1().ClusterRoles().Get(ctx, clusterRole.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdClusterRole.ObjectMeta.Name).Should(Equal("submariner-globalnet"))
		})
	})

	When("called twice", func() {
		It("Should add the ClusterRole properly, and return false on second call", func() {
			created, err := CreateOrUpdateClusterRole(ctx, client, clusterRole)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateClusterRole(ctx, client, clusterRole)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

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
			created, err := CreateOrUpdateClusterRoleBinding(ctx, client, clusterRoleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdClusterRoleBinding, err := client.RbacV1().ClusterRoleBindings().Get(ctx, clusterRoleBinding.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdClusterRoleBinding.ObjectMeta.Name).Should(Equal("submariner-globalnet"))
		})
	})

	When("called twice", func() {
		It("Should add the ClusterRoleBinding properly, and return false on second call", func() {
			created, err := CreateOrUpdateClusterRoleBinding(ctx, client, clusterRoleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateClusterRoleBinding(ctx, client, clusterRoleBinding)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("CreateOrUpdateCRD", func() {
	var (
		crd    *apiextensions.CustomResourceDefinition
		client *extendedfakeclientset.Clientset
		ctx    context.Context
	)

	BeforeEach(func() {
		crd = &apiextensions.CustomResourceDefinition{}
		err := embeddedyamls.GetObject(embeddedyamls.Deploy_crds_submariner_io_submariners_yaml, crd)
		Expect(err).ShouldNot(HaveOccurred())
		client = extendedfakeclientset.NewSimpleClientset()
		ctx = context.TODO()
	})

	When("called", func() {
		It("Should add the CRD properly", func() {
			created, err := CreateOrUpdateCRD(ctx, crdutils.NewFromClientSet(client), crd)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdCrd, err := client.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crd.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdCrd.Spec.Names.Kind).Should(Equal("Submariner"))
		})
	})

	When("called twice", func() {
		It("Should add the CRD properly, and return false on second call", func() {
			crdUpdater := crdutils.NewFromClientSet(client)
			created, err := CreateOrUpdateCRD(ctx, crdUpdater, crd)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateCRD(ctx, crdUpdater, crd)
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
			created, err := CreateOrUpdateDeployment(ctx, client, namespace, deployment)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdDeployment, err := client.AppsV1().Deployments(namespace).Get(ctx, deployment.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdDeployment.ObjectMeta.Name).Should(Equal(name))
		})
	})

	When("called twice", func() {
		It("Should add the Deployment properly, and return false on second call", func() {
			created, err := CreateOrUpdateDeployment(ctx, client, namespace, deployment)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateDeployment(ctx, client, namespace, deployment)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("CreateOrUpdateRole", func() {
	var (
		namespace = "test-namespace"
		role      *rbacv1.Role
		client    *fakeclientset.Clientset
		ctx       context.Context
	)

	BeforeEach(func() {
		role = &rbacv1.Role{}
		// TODO skitt add our own object
		err := embeddedyamls.GetObject(embeddedyamls.Config_rbac_submariner_operator_role_yaml, role)
		Expect(err).ShouldNot(HaveOccurred())
		client = fakeclientset.NewSimpleClientset()
		ctx = context.TODO()
	})

	When("called", func() {
		It("Should add the Role properly", func() {
			created, err := CreateOrUpdateRole(ctx, client, namespace, role)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdRole, err := client.RbacV1().Roles(namespace).Get(ctx, role.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdRole.ObjectMeta.Name).Should(Equal("submariner-operator"))
		})
	})

	When("called twice", func() {
		It("Should add the Role properly, and return false on second call", func() {
			created, err := CreateOrUpdateRole(ctx, client, namespace, role)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateRole(ctx, client, namespace, role)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("CreateOrUpdateRoleBinding", func() {
	var (
		namespace   = "test-namespace"
		roleBinding *rbacv1.RoleBinding
		client      *fakeclientset.Clientset
		ctx         context.Context
	)

	BeforeEach(func() {
		roleBinding = &rbacv1.RoleBinding{}
		// TODO skitt add our own object
		err := embeddedyamls.GetObject(embeddedyamls.Config_rbac_submariner_operator_role_binding_yaml, roleBinding)
		Expect(err).ShouldNot(HaveOccurred())
		client = fakeclientset.NewSimpleClientset()
		ctx = context.TODO()
	})

	When("called", func() {
		It("Should add the RoleBinding properly", func() {
			created, err := CreateOrUpdateRoleBinding(ctx, client, namespace, roleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdRoleBinding, err := client.RbacV1().RoleBindings(namespace).Get(ctx, roleBinding.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdRoleBinding.ObjectMeta.Name).Should(Equal("submariner-operator"))
		})
	})

	When("called twice", func() {
		It("Should add the RoleBinding properly, and return false on second call", func() {
			created, err := CreateOrUpdateRoleBinding(ctx, client, namespace, roleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateRoleBinding(ctx, client, namespace, roleBinding)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func TestCreateOrUpdate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create or update handling")
}
