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

package network_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testComponent1 = "test-component1"
	testComponent2 = "test-component2"
	testFirstPod   = "first-pod"
	testSecondPod  = "second-pod"
	testThirdPod   = "third-pod"
)

var _ = Describe("findPod", func() {
	var client client.Client

	BeforeEach(func() {
		client = fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			fakePodWithName(testFirstPod, testComponent1, nil, nil),
			fakePodWithName(testSecondPod, testComponent2, nil, nil),
			fakePodWithName(testThirdPod, testComponent2, nil, nil)).Build()
	})

	When("There are no pods to be found", func() {
		It("Should return nil", func() {
			pod, err := network.FindPod(context.TODO(), client, "component=not-to-be-found")
			Expect(err).NotTo(HaveOccurred())
			Expect(pod).To(BeNil())
		})
	})

	When("A pod is found", func() {
		It("Should return the pod", func() {
			pod, err := network.FindPod(context.TODO(), client, componentLabel(testComponent1))
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Name).To(Equal(testFirstPod))
		})
	})

	When("Multiple pods are found", func() {
		It("Should return the first pod", func() {
			pod, err := network.FindPod(context.TODO(), client, componentLabel(testComponent2))
			Expect(err).NotTo(HaveOccurred())
			Expect(pod.Name).To(Equal(testSecondPod))
		})
	})
})

const (
	testParameter1 = "--parameter1"
	testValue1     = "value1"
	testComponent3 = "test-component3"
)

var _ = Describe("findPodCommandParameter", func() {
	var client client.Client

	BeforeEach(func() {
		client = fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(
			fakePodWithName(testFirstPod, testComponent1, nil, nil),
			fakePodWithName(testSecondPod, testComponent2,
				[]string{"component1", testParameter1 + "=" + testValue1}, nil),
			fakePodWithName(testThirdPod, testComponent3,
				[]string{"sh", "-c", "component1 " + testParameter1 + "=" + testValue1}, nil),
		).Build()
	})

	When("There are no pods to be found", func() {
		It("Should return empty string", func() {
			param, err := network.FindPodCommandParameter(context.TODO(), client, componentLabel("not-to-be-found"), testParameter1)
			Expect(err).ToNot(HaveOccurred())
			Expect(param).To(BeEmpty())
		})
	})

	When("A pod is found, but does not contain the parameter", func() {
		It("Should return an empty string", func() {
			param, err := network.FindPodCommandParameter(context.TODO(), client, componentLabel(testComponent1), "unknown-parameter")
			Expect(err).ToNot(HaveOccurred())
			Expect(param).To(BeEmpty())
		})
	})

	When("A pod is found, and contains the parameter", func() {
		It("Should return the parameter value", func() {
			param, err := network.FindPodCommandParameter(context.TODO(), client, componentLabel(testComponent2), testParameter1)
			Expect(err).ToNot(HaveOccurred())
			Expect(param).To(Equal(testValue1))
		})
	})

	When("A pod is found, and the parameter is wrapped in a sh call", func() {
		It("Should return the parameter value", func() {
			param, err := network.FindPodCommandParameter(context.TODO(), client, componentLabel(testComponent3), testParameter1)
			Expect(err).ToNot(HaveOccurred())
			Expect(param).To(Equal(testValue1))
		})
	})
})

func componentLabel(component string) string {
	return "component=" + component
}
