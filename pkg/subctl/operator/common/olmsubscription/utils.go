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

package olmsubscription

import (
	"fmt"
	"time"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmclientv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1alpha1"
	extendedclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

func SubscriptionCRDExists(clientSet extendedclientset.Interface) (bool, error) {
	existingCrd, err := clientSet.ApiextensionsV1beta1().CustomResourceDefinitions().Get("subscriptions.operators.coreos.com", metav1.GetOptions{})
	if err != nil {
		return false, err
	}
	if existingCrd != nil {
		return true, nil
	}
	return false, nil
}

func WaitForReady(clientSet olmclientv1alpha1.OperatorsV1alpha1Interface, namespace string, subscription string, interval, timeout time.Duration) error {

	subscriptions := clientSet.Subscriptions(namespace)

	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		sub, err := subscriptions.Get(subscription, metav1.GetOptions{})
		if err != nil && !errors.IsNotFound(err) {
			return false, fmt.Errorf("error waiting for subscription: %s", err)
		}

		if sub.Status.State == olmv1alpha1.SubscriptionStateAtLatest {
			return true, nil
		}
		return false, nil
	})
}
