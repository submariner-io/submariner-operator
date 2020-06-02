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
	"time"

	olmv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
	olmclientv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/client/clientset/versioned/typed/operators/v1alpha1"
	extendedclientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/utils"
)

const subscriptionCheckInterval = 5 * time.Second
const subscriptionWaitTime = 1 * time.Minute

func Available(restConfig *rest.Config) (bool, error) {
	clientSet, err := extendedclientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	return SubscriptionCRDExists(clientSet)
}

func Ensure(restConfig *rest.Config, namespace string, operatorName string) (bool, error) {
	clientSet, err := olmclientv1alpha1.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	subscription := &olmv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      operatorName,
		},
		Spec: &olmv1alpha1.SubscriptionSpec{
			Channel:                "alpha",
			Package:                "submariner",
			CatalogSource:          "operatorhubio-catalog",
			CatalogSourceNamespace: "olm",
		},
	}

	created, err := utils.CreateOrUpdateSubscription(clientSet, namespace, subscription)
	if err != nil {
		return false, err
	}

	err = WaitForReady(clientSet, namespace, subscription.Name, subscriptionCheckInterval, subscriptionWaitTime)

	return created, err
}
