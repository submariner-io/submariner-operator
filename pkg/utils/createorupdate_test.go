/*
© 2020 Red Hat, Inc. and others.

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

package utils

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	extendedfakeclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
)

var _ = Describe("CreateOrUpdateClusterRole", func() {
	var (
		clusterRole *rbacv1.ClusterRole
		client      *fakeclientset.Clientset
	)

	BeforeEach(func() {
		clusterRole = &rbacv1.ClusterRole{}
		// TODO skitt add our own object
		err := embeddedyamls.GetObject(embeddedyamls.Cluster_role_yaml, clusterRole)
		Expect(err).ShouldNot(HaveOccurred())
		client = fakeclientset.NewSimpleClientset()
	})

	When("When called", func() {
		It("Should add the ClusterRole properly", func() {
			created, err := CreateOrUpdateClusterRole(client, clusterRole)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdClusterRole, err := client.RbacV1().ClusterRoles().Get(clusterRole.Name, v1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdClusterRole.ObjectMeta.Name).Should(Equal("submariner-operator:globalnet"))
		})
	})

	When("When called twice", func() {
		It("Should add the ClusterRole properly, and return false on second call", func() {
			created, err := CreateOrUpdateClusterRole(client, clusterRole)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateClusterRole(client, clusterRole)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("CreateOrUpdateClusterRoleBinding", func() {
	var (
		clusterRoleBinding *rbacv1.ClusterRoleBinding
		client             *fakeclientset.Clientset
	)

	BeforeEach(func() {
		clusterRoleBinding = &rbacv1.ClusterRoleBinding{}
		// TODO skitt add our own object
		err := embeddedyamls.GetObject(embeddedyamls.Cluster_role_binding_yaml, clusterRoleBinding)
		Expect(err).ShouldNot(HaveOccurred())
		client = fakeclientset.NewSimpleClientset()
	})

	When("When called", func() {
		It("Should add the ClusterRoleBinding properly", func() {
			created, err := CreateOrUpdateClusterRoleBinding(client, clusterRoleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdClusterRoleBinding, err := client.RbacV1().ClusterRoleBindings().Get(clusterRoleBinding.Name, v1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdClusterRoleBinding.ObjectMeta.Name).Should(Equal("submariner-operator:globalnet"))
		})
	})

	When("When called twice", func() {
		It("Should add the ClusterRoleBinding properly, and return false on second call", func() {
			created, err := CreateOrUpdateClusterRoleBinding(client, clusterRoleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateClusterRoleBinding(client, clusterRoleBinding)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

var _ = Describe("CreateOrUpdateCRD", func() {
	var (
		crd    *apiextensionsv1beta1.CustomResourceDefinition
		client *extendedfakeclientset.Clientset
	)

	BeforeEach(func() {
		crd = &apiextensionsv1beta1.CustomResourceDefinition{}
		err := embeddedyamls.GetObject(embeddedyamls.Crds_submariner_io_submariners_crd_yaml, crd)
		Expect(err).ShouldNot(HaveOccurred())
		client = extendedfakeclientset.NewSimpleClientset()
	})

	When("When called", func() {
		It("Should add the CRD properly", func() {
			created, err := CreateOrUpdateCRD(client, crd)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdCrd, err := client.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.Name, v1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdCrd.Spec.Names.Kind).Should(Equal("Submariner"))
		})
	})

	When("When called twice", func() {
		It("Should add the CRD properly, and return false on second call", func() {
			created, err := CreateOrUpdateCRD(client, crd)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateCRD(client, crd)
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
	})

	When("When called", func() {
		It("Should add the Deployment properly", func() {
			created, err := CreateOrUpdateDeployment(client, namespace, deployment)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdDeployment, err := client.AppsV1().Deployments(namespace).Get(deployment.Name, v1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdDeployment.ObjectMeta.Name).Should(Equal(name))
		})
	})

	When("When called twice", func() {
		It("Should add the Deployment properly, and return false on second call", func() {
			created, err := CreateOrUpdateDeployment(client, namespace, deployment)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateDeployment(client, namespace, deployment)
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
	)

	BeforeEach(func() {
		role = &rbacv1.Role{}
		// TODO skitt add our own object
		err := embeddedyamls.GetObject(embeddedyamls.Role_yaml, role)
		Expect(err).ShouldNot(HaveOccurred())
		client = fakeclientset.NewSimpleClientset()
	})

	When("When called", func() {
		It("Should add the Role properly", func() {
			created, err := CreateOrUpdateRole(client, namespace, role)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdRole, err := client.RbacV1().Roles(namespace).Get(role.Name, v1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdRole.ObjectMeta.Name).Should(Equal("submariner-operator"))
		})
	})

	When("When called twice", func() {
		It("Should add the Role properly, and return false on second call", func() {
			created, err := CreateOrUpdateRole(client, namespace, role)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateRole(client, namespace, role)
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
	)

	BeforeEach(func() {
		roleBinding = &rbacv1.RoleBinding{}
		// TODO skitt add our own object
		err := embeddedyamls.GetObject(embeddedyamls.Role_binding_yaml, roleBinding)
		Expect(err).ShouldNot(HaveOccurred())
		client = fakeclientset.NewSimpleClientset()
	})

	When("When called", func() {
		It("Should add the RoleBinding properly", func() {
			created, err := CreateOrUpdateRoleBinding(client, namespace, roleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())

			createdRoleBinding, err := client.RbacV1().RoleBindings(namespace).Get(roleBinding.Name, v1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(createdRoleBinding.ObjectMeta.Name).Should(Equal("submariner-operator"))
		})
	})

	When("When called twice", func() {
		It("Should add the RoleBinding properly, and return false on second call", func() {
			created, err := CreateOrUpdateRoleBinding(client, namespace, roleBinding)
			Expect(created).To(BeTrue())
			Expect(err).ToNot(HaveOccurred())
			created, err = CreateOrUpdateRoleBinding(client, namespace, roleBinding)
			Expect(created).To(BeFalse())
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func TestCreateOrUpdate(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create or update handling")
}
