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

package deployment_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/deployment"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Ensure", func() {
	replicas := int32(2)
	testDeployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "test-namespace",
			Name:      "test-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
		},
	}

	var client *fakeclientset.Clientset

	BeforeEach(func() {
		client = fakeclientset.NewSimpleClientset()
	})

	assertDeployment := func() {
		d, err := client.AppsV1().Deployments(testDeployment.Namespace).Get(context.TODO(), testDeployment.Name, metav1.GetOptions{})
		Expect(err).To(Succeed())
		Expect(d.Spec.Replicas).To(Equal(&replicas))
	}

	When("the Deployment doesn't exist", func() {
		It("should create it", func() {
			created, err := deployment.Ensure(client, testDeployment.Namespace, testDeployment)
			Expect(created).To(BeTrue())
			Expect(err).To(Succeed())
			assertDeployment()
		})
	})

	When("the Deployment already exists", func() {
		It("should not update it", func() {
			_, err := client.AppsV1().Deployments(testDeployment.Namespace).Create(context.TODO(), testDeployment, metav1.CreateOptions{})
			Expect(err).To(Succeed())

			created, err := deployment.Ensure(client, testDeployment.Namespace, testDeployment)
			Expect(created).To(BeFalse())
			Expect(err).To(Succeed())
		})
	})
})
