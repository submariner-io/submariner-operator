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

package submariner_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	submarinerController "github.com/submariner-io/submariner-operator/controllers/submariner"
	"github.com/submariner-io/submariner-operator/controllers/test"
	"github.com/submariner-io/submariner-operator/pkg/discovery/globalnet"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const brokerName = "test-broker"

var _ = Describe("Broker controller tests", func() {
	t := test.Driver{
		Namespace:    submarinerNamespace,
		ResourceName: brokerName,
	}

	var broker *v1alpha1.Broker

	BeforeEach(func() {
		t.BeforeEach()
		broker = &v1alpha1.Broker{
			ObjectMeta: metav1.ObjectMeta{
				Name:      brokerName,
				Namespace: submarinerNamespace,
			},
			Spec: v1alpha1.BrokerSpec{
				GlobalnetCIDRRange:          "168.254.0.0/16",
				DefaultGlobalnetClusterSize: 8192,
				GlobalnetEnabled:            true,
			},
		}

		t.InitScopedClientObjs = []client.Object{broker}
	})

	JustBeforeEach(func() {
		t.JustBeforeEach()

		t.Controller = &submarinerController.BrokerReconciler{
			Client: t.ScopedClient,
		}
	})

	It("should create the globalnet ConfigMap", func(ctx SpecContext) {
		t.AssertReconcileSuccess(ctx)

		globalnetInfo, _, err := globalnet.GetGlobalNetworks(ctx, t.ScopedClient, submarinerNamespace)
		Expect(err).To(Succeed())
		Expect(globalnetInfo.CIDR).To(Equal(broker.Spec.GlobalnetCIDRRange))
		Expect(globalnetInfo.AllocationSize).To(Equal(broker.Spec.DefaultGlobalnetClusterSize))
	})

	It("should create the CRDs", func(ctx SpecContext) {
		t.AssertReconcileSuccess(ctx)

		crd := &apiextensions.CustomResourceDefinition{}
		Expect(t.ScopedClient.Get(ctx, client.ObjectKey{Name: "clusters.submariner.io"}, crd)).To(Succeed())
		Expect(t.ScopedClient.Get(ctx, client.ObjectKey{Name: "endpoints.submariner.io"}, crd)).To(Succeed())
		Expect(t.ScopedClient.Get(ctx, client.ObjectKey{Name: "gateways.submariner.io"}, crd)).To(Succeed())
		Expect(t.ScopedClient.Get(ctx, client.ObjectKey{Name: "serviceimports.multicluster.x-k8s.io"}, crd)).To(Succeed())
	})

	When("the Broker resource doesn't exist", func() {
		BeforeEach(func() {
			t.InitScopedClientObjs = nil
		})

		It("should return success", func(ctx SpecContext) {
			t.AssertReconcileSuccess(ctx)
		})
	})
})
