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

package cleanup

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/shipyard/test/e2e/framework"
	operatorv1alpha1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/uninstall"
	operatorclient "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	operatorv1alpha1client "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned/typed/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

var _ = Describe("Service Discovery cleanup", func() {
	When("the ServiceDiscovery resource is deleted", testServiceDiscoveryCleanup)
})

func testServiceDiscoveryCleanup() {
	var (
		serviceDiscoveryInterface operatorv1alpha1client.ServiceDiscoveryInterface
		submarinerInterface       operatorv1alpha1client.SubmarinerInterface
		serviceDiscovery          *operatorv1alpha1.ServiceDiscovery
		submariner                *operatorv1alpha1.Submariner
		lhAgentAgentPodMonitor    *podMonitor
		stopCh                    chan struct{}
	)

	BeforeEach(func() {
		stopCh = make(chan struct{})

		operatorClient, err := operatorclient.NewForConfig(framework.RestConfigs[framework.ClusterA])
		Expect(err).To(Succeed())

		serviceDiscoveryInterface = operatorClient.SubmarinerV1alpha1().ServiceDiscoveries(framework.TestContext.SubmarinerNamespace)
		submarinerInterface = operatorClient.SubmarinerV1alpha1().Submariners(framework.TestContext.SubmarinerNamespace)

		installed := false

		submariner, err = submarinerInterface.Get(context.TODO(), submarinercr.SubmarinerName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			submariner = nil

			serviceDiscovery, err = serviceDiscoveryInterface.Get(context.TODO(), names.ServiceDiscoveryCrName, metav1.GetOptions{})
			if !apierrors.IsNotFound(err) {
				Expect(err).To(Succeed())
				installed = true
			}
		} else {
			Expect(err).To(Succeed())
			installed = submariner.Spec.ServiceDiscoveryEnabled
		}

		if !installed {
			framework.Skipf("ServiceDiscovery is not installed, skipping the test...")
			return
		}

		lhAgentAgentPodMonitor = startPodMonitor(names.AppendUninstall(names.ServiceDiscoveryComponent), stopCh)
	})

	AfterEach(func() {
		close(stopCh)

		if submariner != nil {
			Expect(util.Update(context.TODO(), resourceInterfaceForSubmariner(submarinerInterface), submariner,
				func(existing runtime.Object) (runtime.Object, error) {
					existing.(*operatorv1alpha1.Submariner).Spec.ServiceDiscoveryEnabled = true
					return existing, nil
				})).To(Succeed())
		} else if serviceDiscovery != nil {
			serviceDiscovery.ObjectMeta = metav1.ObjectMeta{
				Name:        serviceDiscovery.Name,
				Labels:      serviceDiscovery.Labels,
				Annotations: serviceDiscovery.Annotations,
			}

			_, err := serviceDiscoveryInterface.Create(context.TODO(), serviceDiscovery, metav1.CreateOptions{})
			if err != nil {
				framework.Errorf("Error re-creating ServiceDiscovery: %v", err)
			} else {
				By("Re-created ServiceDiscovery resource")
			}
		}
	})

	It("should run an uninstall Deployment and cleanup coredns config", func() {
		By(fmt.Sprintf("Deleting ServiceDiscovery resource in %q", framework.TestContext.ClusterIDs[framework.ClusterA]))

		if submariner != nil {
			Expect(util.Update(context.TODO(), resourceInterfaceForSubmariner(submarinerInterface), submariner,
				func(existing runtime.Object) (runtime.Object, error) {
					existing.(*operatorv1alpha1.Submariner).Spec.ServiceDiscoveryEnabled = false
					return existing, nil
				})).To(Succeed())
		} else {
			Expect(serviceDiscoveryInterface.Delete(context.TODO(), names.ServiceDiscoveryCrName, metav1.DeleteOptions{})).To(Succeed())
		}

		Eventually(func() bool {
			_, err := serviceDiscoveryInterface.Get(context.TODO(), names.ServiceDiscoveryCrName, metav1.GetOptions{})
			return apierrors.IsNotFound(err)
		}, uninstall.ComponentReadyTimeout*2).Should(BeTrue(), "ServiceDiscovery resource not deleted")

		By("ServiceDiscovery resource deleted")

		lhAgentAgentPodMonitor.assertUninstallPodsCompleted()

		_, err := framework.KubeClients[framework.ClusterA].AppsV1().Deployments(framework.TestContext.SubmarinerNamespace).Get(context.TODO(),
			names.AppendUninstall(names.ServiceDiscoveryComponent), metav1.GetOptions{})
		Expect(apierrors.IsNotFound(err)).To(BeTrue(), fmt.Sprintf("Unexpected Deployment %q found",
			names.AppendUninstall(names.ServiceDiscoveryComponent)))

		Expect(getCorednsConfigMap().Data["Corefile"]).ToNot(ContainSubstring("lighthouse"))
	})
}

func getCorednsConfigMap() *corev1.ConfigMap {
	configMap, err := framework.KubeClients[framework.ClusterA].CoreV1().ConfigMaps("kube-system").Get(context.TODO(),
		"coredns", metav1.GetOptions{})
	Expect(err).To(Succeed())

	return configMap
}

func resourceInterfaceForSubmariner(client operatorv1alpha1client.SubmarinerInterface) resource.Interface {
	return &resource.InterfaceFuncs{
		GetFunc: func(ctx context.Context, name string, options metav1.GetOptions) (runtime.Object, error) {
			return client.Get(ctx, name, options)
		},
		CreateFunc: func(ctx context.Context, obj runtime.Object, options metav1.CreateOptions) (runtime.Object, error) {
			return client.Create(ctx, obj.(*operatorv1alpha1.Submariner), options)
		},
		UpdateFunc: func(ctx context.Context, obj runtime.Object, options metav1.UpdateOptions) (runtime.Object, error) {
			return client.Update(ctx, obj.(*operatorv1alpha1.Submariner), options)
		},
		DeleteFunc: func(ctx context.Context, name string, options metav1.DeleteOptions) error {
			return client.Delete(ctx, name, options)
		},
	}
}
