/*
© 2019 Red Hat, Inc. and others.

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
	"fmt"
	"time"

	v1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"

	"github.com/submariner-io/submariner-operator/pkg/engine"
	"github.com/submariner-io/submariner-operator/pkg/lighthouse"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Ensure(config *rest.Config) error {
	err := engine.Ensure(config)
	if err != nil {
		return fmt.Errorf("error setting up the engine requirements: %s", err)
	}

	err = lighthouse.Ensure(config)
	if err != nil {
		return fmt.Errorf("error setting up the lighthouse requirements: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the core kubernetes clientset: %s", err)
	}

	// Create the namespace
	_, err = CreateNewBrokerNamespace(clientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker namespace %s", err)
	}

	// Create administrator SA, Role, and bind them
	if err = createBrokerAdministratorRoleAndSA(clientset); err != nil {
		return err
	}

	// Create cluster Role, and a default account for backwards compatibility, also bind it
	if err = createBrokerClusterRoleAndDefaultSA(clientset); err != nil {
		return err
	}
	_, err = WaitForClientToken(clientset, SubmarinerBrokerAdminSA)
	return err
}

func createBrokerClusterRoleAndDefaultSA(clientset *kubernetes.Clientset) error {

	// Create the a default SA for cluster access (backwards compatibility with documentation)
	_, err := CreateNewBrokerSA(clientset, submarinerBrokerClusterDefaultSA)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the default broker service account: %s", err)
	}

	// Create the broker cluster role, which will also be used by any new enrolled cluster
	_, err = CreateNewClusterBrokerRole(clientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating broker role: %s", err)
	}

	// Create the role binding
	_, err = CreateNewBrokerRoleBinding(clientset, submarinerBrokerClusterDefaultSA, submarinerBrokerClusterRole)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker rolebinding: %s", err)
	}
	return nil

}

// CreateSAForCluster creates a new SA, and binds it to the submariner cluster role
func CreateSAForCluster(clientset *kubernetes.Clientset, clusterID string) (*v1.Secret, error) {
	saName := fmt.Sprintf(submarinerBrokerClusterSAFmt, clusterID)
	_, err := CreateNewBrokerSA(clientset, saName)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("error creating cluster sa: %s", err)
	}

	_, err = CreateNewBrokerRoleBinding(clientset, saName, submarinerBrokerClusterRole)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("error binding sa to cluster role: %s", err)
	}

	clientToken, err := WaitForClientToken(clientset, saName)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("error getting cluster sa token: %s", err)
	}
	return clientToken, nil

}

func createBrokerAdministratorRoleAndSA(clientset *kubernetes.Clientset) error {
	// Create the SA we need for the managing the broker (from subctl, etc..)
	_, err := CreateNewBrokerSA(clientset, SubmarinerBrokerAdminSA)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker admin service account: %s", err)
	}

	// Create the broker admin role
	_, err = CreateNewBrokerAdminRole(clientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating subctl role: %s", err)
	}

	// Create the role binding
	_, err = CreateNewBrokerRoleBinding(clientset, SubmarinerBrokerAdminSA, submarinerBrokerAdminRole)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker rolebinding: %s", err)
	}

	return nil
}

func WaitForClientToken(clientset *kubernetes.Clientset, submarinerBrokerSA string) (secret *v1.Secret, err error) {

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
		secret, lastErr = GetClientTokenSecret(clientset, SubmarinerBrokerNamespace, submarinerBrokerSA)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err == wait.ErrWaitTimeout {
		return nil, lastErr
	}

	return secret, err
}

func CreateNewBrokerNamespace(clientset *kubernetes.Clientset) (brokernamespace *v1.Namespace, err error) {
	return clientset.CoreV1().Namespaces().Create(NewBrokerNamespace())
}

func CreateNewClusterBrokerRole(clientset *kubernetes.Clientset) (brokerrole *rbac.Role, err error) {
	return clientset.RbacV1().Roles(SubmarinerBrokerNamespace).Create(NewBrokerClusterRole())
}

func CreateNewBrokerAdminRole(clientset *kubernetes.Clientset) (brokerAdminRole *rbac.Role, err error) {
	return clientset.RbacV1().Roles(SubmarinerBrokerNamespace).Create(NewBrokerAdminRole())
}

func CreateNewBrokerRoleBinding(clientset *kubernetes.Clientset, serviceAccount, role string) (brokerRoleBinding *rbac.RoleBinding, err error) {
	return clientset.RbacV1().RoleBindings(SubmarinerBrokerNamespace).Create(
		NewBrokerRoleBinding(serviceAccount, role),
	)
}

func CreateNewBrokerSA(clientset *kubernetes.Clientset, submarinerBrokerSA string) (brokerSA *v1.ServiceAccount, err error) {
	return clientset.CoreV1().ServiceAccounts(SubmarinerBrokerNamespace).Create(NewBrokerSA(submarinerBrokerSA))
}
