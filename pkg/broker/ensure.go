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

package broker

import (
	"context"
	"errors"
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/gateway"
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"
	"github.com/submariner-io/submariner-operator/pkg/subctl/components"
	"github.com/submariner-io/submariner-operator/pkg/utils"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
)

func Ensure(config *rest.Config, componentArr []string, crds bool, namespace string) error {
	if crds {
		crdCreator, err := crdutils.NewFromRestConfig(config)
		if err != nil {
			return fmt.Errorf("error accessing the target cluster: %s", err)
		}

		for i := range componentArr {
			switch componentArr[i] {
			case components.Connectivity:
				err = gateway.Ensure(crdCreator)
				if err != nil {
					return fmt.Errorf("error setting up the connectivity requirements: %s", err)
				}
			case components.ServiceDiscovery:
				_, err = lighthouse.Ensure(crdCreator, lighthouse.BrokerCluster)
				if err != nil {
					return fmt.Errorf("error setting up the service discovery requirements: %s", err)
				}
			case components.Globalnet:
				// Globalnet needs the Lighthouse CRDs too
				_, err = lighthouse.Ensure(crdCreator, lighthouse.BrokerCluster)
				if err != nil {
					return fmt.Errorf("error setting up the globalnet requirements: %s", err)
				}
			}
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the core kubernetes clientset: %s", err)
	}

	// Create the namespace
	_, err = CreateNewBrokerNamespace(clientset, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker namespace %s", err)
	}

	// Create administrator SA, Role, and bind them
	if err := createBrokerAdministratorRoleAndSA(clientset, namespace); err != nil {
		return err
	}

	// Create cluster Role, and a default account for backwards compatibility, also bind it
	if err := createBrokerClusterRoleAndDefaultSA(clientset, namespace); err != nil {
		return err
	}
	_, err = WaitForClientToken(clientset, SubmarinerBrokerAdminSA, namespace)
	return err
}

func createBrokerClusterRoleAndDefaultSA(clientset *kubernetes.Clientset, namespace string) error {
	// Create the a default SA for cluster access (backwards compatibility with documentation)
	_, err := CreateNewBrokerSA(clientset, submarinerBrokerClusterDefaultSA, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the default broker service account: %s", err)
	}

	// Create the broker cluster role, which will also be used by any new enrolled cluster
	_, err = CreateOrUpdateClusterBrokerRole(clientset, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating broker role: %s", err)
	}

	// Create the role binding
	_, err = CreateNewBrokerRoleBinding(clientset, submarinerBrokerClusterDefaultSA, submarinerBrokerClusterRole, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker rolebinding: %s", err)
	}
	return nil
}

// CreateSAForCluster creates a new SA, and binds it to the submariner cluster role
func CreateSAForCluster(clientset *kubernetes.Clientset, clusterID, namespace string) (*v1.Secret, error) {
	saName := fmt.Sprintf(submarinerBrokerClusterSAFmt, clusterID)
	_, err := CreateNewBrokerSA(clientset, saName, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("error creating cluster sa: %s", err)
	}

	_, err = CreateNewBrokerRoleBinding(clientset, saName, submarinerBrokerClusterRole, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("error binding sa to cluster role: %s", err)
	}

	clientToken, err := WaitForClientToken(clientset, saName, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("error getting cluster sa token: %s", err)
	}
	return clientToken, nil
}

func createBrokerAdministratorRoleAndSA(clientset *kubernetes.Clientset, namespace string) error {
	// Create the SA we need for the managing the broker (from subctl, etc..)
	_, err := CreateNewBrokerSA(clientset, SubmarinerBrokerAdminSA, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker admin service account: %s", err)
	}

	// Create the broker admin role
	_, err = CreateOrUpdateBrokerAdminRole(clientset, namespace)
	if err != nil {
		return fmt.Errorf("error creating subctl role: %s", err)
	}

	// Create the role binding
	_, err = CreateNewBrokerRoleBinding(clientset, SubmarinerBrokerAdminSA, submarinerBrokerAdminRole, namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker rolebinding: %s", err)
	}

	return nil
}

func WaitForClientToken(clientset *kubernetes.Clientset, submarinerBrokerSA, namespace string) (secret *v1.Secret, err error) {
	// wait for the client token to be ready, while implementing
	// exponential backoff pattern, it will wait a total of:
	// sum(n=0..9, 1.2^n * 5) seconds, = 130 seconds

	backoff := wait.Backoff{
		Steps:    10,
		Duration: 5 * time.Second,
		Factor:   1.2,
		Jitter:   1,
	}

	var lastErr error
	err = wait.ExponentialBackoff(backoff, func() (bool, error) {
		secret, lastErr = GetClientTokenSecret(clientset, namespace, submarinerBrokerSA)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if errors.Is(err, wait.ErrWaitTimeout) {
		return nil, lastErr
	}

	return secret, err
}

func CreateNewBrokerNamespace(clientset *kubernetes.Clientset, namespace string) (brokernamespace *v1.Namespace, err error) {
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}

	return clientset.CoreV1().Namespaces().Create(context.TODO(), ns, metav1.CreateOptions{})
}

func CreateOrUpdateClusterBrokerRole(clientset *kubernetes.Clientset, namespace string) (created bool, err error) {
	return utils.CreateOrUpdateRole(context.TODO(), clientset, namespace, NewBrokerClusterRole())
}

func CreateOrUpdateBrokerAdminRole(clientset *kubernetes.Clientset, namespace string) (created bool, err error) {
	return utils.CreateOrUpdateRole(context.TODO(), clientset, namespace, NewBrokerAdminRole())
}

func CreateNewBrokerRoleBinding(clientset *kubernetes.Clientset, serviceAccount, role, namespace string) (
	brokerRoleBinding *rbac.RoleBinding, err error) {
	return clientset.RbacV1().RoleBindings(namespace).Create(
		context.TODO(), NewBrokerRoleBinding(serviceAccount, role, namespace), metav1.CreateOptions{})
}

func CreateNewBrokerSA(clientset *kubernetes.Clientset, submarinerBrokerSA, namespace string) (brokerSA *v1.ServiceAccount, err error) {
	return clientset.CoreV1().ServiceAccounts(namespace).Create(
		context.TODO(), NewBrokerSA(submarinerBrokerSA), metav1.CreateOptions{})
}
