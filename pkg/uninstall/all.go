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

package uninstall

import (
	"context"
	"strings"
	"time"

	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	"github.com/submariner-io/submariner-operator/controllers/uninstall"
	"github.com/submariner-io/submariner-operator/internal/constants"
	"github.com/submariner-io/submariner-operator/pkg/client"
	"github.com/submariner-io/submariner-operator/pkg/names"
	"github.com/submariner-io/submariner-operator/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/servicediscoverycr"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
)

func All(clients client.Producer, clusterName, submarinerNamespace string, status reporter.Interface) error {
	_, err := clients.ForKubernetes().CoreV1().Namespaces().Get(context.TODO(), submarinerNamespace, metav1.GetOptions{})
	if err != nil {
		return status.Error(err, "Error retrieving the Submariner namespace %q", submarinerNamespace)
	}

	found, brokerNS, err := ensureSubmarinerDeleted(clients, clusterName, submarinerNamespace, status)
	if err != nil {
		return err
	}

	if !found {
		brokerNS, err = ensureServiceDiscoveryDeleted(clients, clusterName, submarinerNamespace, status)
		if err != nil {
			return err
		}
	}

	err = deleteClusterRolesAndBindings(clients, clusterName, status)
	if err != nil {
		return err
	}

	status.Start("Deleting the Submariner namespace %q on cluster %q", submarinerNamespace, clusterName)
	defer status.End()

	err = clients.ForKubernetes().CoreV1().Namespaces().Delete(context.TODO(), submarinerNamespace, metav1.DeleteOptions{})
	if err != nil {
		return status.Error(err, "Error deleting the namespace")
	}

	if brokerNS != "" {
		_, err := clients.ForKubernetes().CoreV1().Namespaces().Get(context.TODO(), brokerNS, metav1.GetOptions{})
		if err == nil {
			status.Start("Deleting the broker namespace %q on cluster %q", brokerNS, clusterName)

			err = clients.ForKubernetes().CoreV1().Namespaces().Delete(context.TODO(), brokerNS, metav1.DeleteOptions{})
			if err != nil && !apierrors.IsNotFound(err) {
				return status.Error(err, "Error deleting the namespace")
			}
		}
	}

	err = deleteCRDs(clients, clusterName, status)
	if err != nil {
		return err
	}

	return unlabelGatewayNodes(clients, clusterName, status)
}

func unlabelGatewayNodes(clients client.Producer, clusterName string, status reporter.Interface) error {
	status.Start("Unlabeling gateway nodes on cluster %q", clusterName)
	defer status.End()

	list, err := clients.ForKubernetes().CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{constants.SubmarinerGatewayLabel: "true"}).String(),
	})
	if err != nil {
		return status.Error(err, "Error listing Nodes")
	}

	// nolint:wrapcheck // Let the caller wrap errors
	nodeInterface := &resource.InterfaceFuncs{
		GetFunc: func(ctx context.Context, name string, options metav1.GetOptions) (runtime.Object, error) {
			return clients.ForKubernetes().CoreV1().Nodes().Get(ctx, name, options)
		},
		UpdateFunc: func(ctx context.Context, obj runtime.Object, options metav1.UpdateOptions) (runtime.Object, error) {
			return clients.ForKubernetes().CoreV1().Nodes().Update(ctx, obj.(*corev1.Node), options)
		},
	}

	for i := range list.Items {
		err = util.Update(context.TODO(), nodeInterface, &list.Items[i], func(existing runtime.Object) (runtime.Object, error) {
			delete(existing.(*corev1.Node).Labels, constants.SubmarinerGatewayLabel)
			return existing, nil
		})
		if err != nil {
			return status.Error(err, "Error updating Node %q", list.Items[i].Name)
		}
	}

	return nil
}

func deleteCRDs(clients client.Producer, clusterName string, status reporter.Interface) error {
	status.Start("Deleting the Submariner custom resource definitions on cluster %q", clusterName)
	defer status.End()

	list, err := clients.ForCRD().ApiextensionsV1().CustomResourceDefinitions().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return status.Error(err, "Error listing CustomResourceDefinitions")
	}

	for i := range list.Items {
		if !strings.HasSuffix(list.Items[i].Name, ".submariner.io") {
			continue
		}

		err = clients.ForCRD().ApiextensionsV1().CustomResourceDefinitions().Delete(context.TODO(), list.Items[i].Name,
			metav1.DeleteOptions{})
		if err != nil {
			return status.Error(err, "Error deleting CustomResourceDefinition %q", list.Items[i].Name)
		}

		status.Success("Deleted the %q custom resource definition", list.Items[i].Name)
	}

	return nil
}

func deleteClusterRolesAndBindings(clients client.Producer, clusterName string, status reporter.Interface) error {
	status.Start("Deleting the Submariner cluster roles and bindings on cluster %q", clusterName)
	defer status.End()

	list, err := clients.ForKubernetes().RbacV1().ClusterRoleBindings().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return status.Error(err, "Error listing ClusterRoleBindings")
	}

	for i := range list.Items {
		if !strings.HasPrefix(list.Items[i].Name, "submariner-") {
			continue
		}

		err = clients.ForKubernetes().RbacV1().ClusterRoleBindings().Delete(context.TODO(), list.Items[i].Name, metav1.DeleteOptions{})
		if err != nil {
			return status.Error(err, "Error deleting ClusterRoleBinding %q", list.Items[i].Name)
		}

		err = clients.ForKubernetes().RbacV1().ClusterRoles().Delete(context.TODO(), list.Items[i].RoleRef.Name, metav1.DeleteOptions{})
		if err != nil {
			return status.Error(err, "Error deleting ClusterRole %q", list.Items[i].RoleRef.Name)
		}

		status.Success("Deleted the %q cluster role and binding", list.Items[i].Name)
	}

	return nil
}

func ensureSubmarinerDeleted(clients client.Producer, clusterName, namespace string, status reporter.Interface) (bool, string, error) {
	defer status.End()

	status.Start("Checking if the connectivity component is installed on cluster %q", clusterName)

	submClient := clients.ForOperator().SubmarinerV1alpha1().Submariners(namespace)

	submariner, err := submClient.Get(context.TODO(), submarinercr.SubmarinerName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		status.Success("The connectivity component is not installed on cluster %q - skipping", clusterName)
		return false, "", nil
	}

	if err != nil {
		return false, "", status.Error(err, "Error retrieving the Submariner resource")
	}

	status.Success("The connectivity component is installed on cluster %q", clusterName)

	status.Start("Deleting the Submariner resource - this may take some time")

	err = ensureDeleted(submarinercr.ResourceInterface(submClient), submariner.Name)

	return true, submariner.Spec.BrokerK8sRemoteNamespace,
		status.Error(err, "Error deleting Submariner resource %q", submariner.Name)
}

func ensureServiceDiscoveryDeleted(clients client.Producer, clusterName, namespace string, status reporter.Interface) (string, error) {
	defer status.End()

	status.Start("Checking if the service discovery component is installed on cluster %q", clusterName)

	sdClient := clients.ForOperator().SubmarinerV1alpha1().ServiceDiscoveries(namespace)

	serviceDiscovery, err := sdClient.Get(context.TODO(), names.ServiceDiscoveryCrName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		status.Success("The service discovery component is not installed on cluster %q - skipping", clusterName)
		return "", nil
	}

	if err != nil {
		return "", status.Error(err, "Error retrieving the ServiceDiscovery resource")
	}

	status.Success("The service discovery component is installed on cluster %q", clusterName)

	status.Start("Deleting the ServiceDiscovery resource - this may take some time")

	err = ensureDeleted(servicediscoverycr.ResourceInterface(sdClient), serviceDiscovery.Name)

	return serviceDiscovery.Spec.BrokerK8sRemoteNamespace,
		status.Error(err, "Error deleting ServiceDiscovery resource %q", serviceDiscovery.Name)
}

func ensureDeleted(resourceInterface resource.Interface, name string) error {
	const maxWait = uninstall.ComponentReadyTimeout + (time.Second * 30)
	const checkInterval = 2 * time.Second

	// nolint:wrapcheck // Let the caller wrap errors
	return wait.PollImmediate(checkInterval, maxWait, func() (bool, error) {
		err := resourceInterface.Delete(context.TODO(), name, metav1.DeleteOptions{})
		if apierrors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
}
