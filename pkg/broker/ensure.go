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

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the core kubernetes clientset: %s", err)
	}

	// Create the namespace
	_, err = CreateNewBrokerNamespace(clientset)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker namespace %s", err)
	}

	// Create the SA we need for the broker
	_, err = CreateNewBrokerSA(clientset, "submariner-k8s-broker-client")
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the default broker service account: %s", err)
	}

	// Create the role
	_, err = CreateNewBrokerRole(clientset, "submariner-k8s-broker-client")
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating broker role: %s", err)
	}

	// Create the role binding
	_, err = CreateNewBrokerRoleBinding(clientset, "submariner-k8s-broker-client", "submariner-k8s-broker-client")
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker rolebinding: %s", err)
	}

	return WaitForClientToken(clientset, "submariner-k8s-broker-client")
}

func WaitForClientToken(clientset *kubernetes.Clientset, submarinerBrokerSA string) error {

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
	err := wait.ExponentialBackoff(backoff, func() (bool, error) {
		_, lastErr = GetClientTokenSecret(clientset, SubmarinerBrokerNamespace, submarinerBrokerSA)
		if lastErr != nil {
			return false, nil
		}
		return true, nil
	})
	if err == wait.ErrWaitTimeout {
		return lastErr
	}

	return err
}

func CreateNewBrokerNamespace(clientset *kubernetes.Clientset) (brokernamespace *v1.Namespace, err error) {
	brokernamespace, err = clientset.CoreV1().Namespaces().Create(NewBrokerNamespace())
	return brokernamespace, err
}

func CreateNewBrokerRole(clientset *kubernetes.Clientset, submarinerBrokerRole string) (brokerrole *rbac.Role, err error) {
	brokerrole, err = clientset.RbacV1().Roles(SubmarinerBrokerNamespace).Create(NewBrokerRole(submarinerBrokerRole))
	return brokerrole, err
}

func CreateNewBrokerRoleBinding(clientset *kubernetes.Clientset, submarinerBrokerRole string, submarinerBrokerSA string) (brokerrolebinding *rbac.RoleBinding, err error) {
	brokerrolebinding, err = clientset.RbacV1().RoleBindings(SubmarinerBrokerNamespace).Create(NewBrokerRoleBinding(submarinerBrokerRole, submarinerBrokerSA))
	return brokerrolebinding, err
}

func CreateNewBrokerSA(clientset *kubernetes.Clientset, submarinerBrokerSA string) (brokerSA *v1.ServiceAccount, err error) {
	brokerSA, err = clientset.CoreV1().ServiceAccounts(SubmarinerBrokerNamespace).Create(NewBrokerSA(submarinerBrokerSA))
	return brokerSA, err
}
