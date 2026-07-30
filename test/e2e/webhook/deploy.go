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

package webhook

import (
	"context"
	"time"

	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	operatorv1alpha1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	webhookyaml "github.com/submariner-io/submariner-operator/config/webhook"
	"github.com/submariner-io/submariner-operator/pkg/webhook/deploy"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func Deploy() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer cancel()

	brokerCluster, brokerNamespace := findBrokerCluster(ctx)
	framework.By("Deploying webhook on broker cluster " + framework.TestContext.ClusterIDs[brokerCluster])

	crClient, err := client.New(framework.RestConfigs[brokerCluster], client.Options{})
	Expect(err).NotTo(HaveOccurred())

	// Deploy self-signed issuer
	framework.By("Deploying self-signed issuer")

	issuer := &unstructured.Unstructured{}
	Expect(yaml.Unmarshal(webhookyaml.SelfSignedIssuer, issuer)).To(Succeed())
	Expect(deploy.Ensure(ctx, crClient, issuer)).To(Succeed())

	// Deploy webhook components
	framework.By("Deploying webhook components")
	Expect(deploy.Webhook(ctx, crClient, issuer.GetName(), "", brokerNamespace, reporter.Stdout())).To(Succeed())

	// Wait for webhook deployment to be ready
	framework.By("Waiting for webhook deployment to be ready")
	framework.AwaitUntil(ctx, "find ready webhook deployment", func(ctx context.Context) (*appsv1.Deployment, error) {
		deployment := &appsv1.Deployment{}
		err := crClient.Get(ctx, client.ObjectKey{
			Namespace: deploy.OperatorNamespace,
			Name:      "submariner-operator-webhook",
		}, deployment)

		return deployment, err
	}, func(result *appsv1.Deployment) (bool, string, error) {
		if result.Status.ReadyReplicas > 0 && result.Status.AvailableReplicas > 0 {
			return true, "", nil
		}

		return false, "Deployment not ready yet", nil
	})
}

func findBrokerCluster(ctx context.Context) (framework.ClusterIndex, string) {
	for i, c := range framework.DynClients {
		list, err := c.Resource(operatorv1alpha1.GroupVersion.WithResource("brokers")).Namespace(metav1.NamespaceAll).
			List(ctx, metav1.ListOptions{})
		Expect(err).NotTo(HaveOccurred())

		for j := range list.Items {
			return framework.ClusterIndex(i), list.Items[j].GetNamespace()
		}
	}

	framework.Fail("Broker cluster not found")

	return 0, ""
}
