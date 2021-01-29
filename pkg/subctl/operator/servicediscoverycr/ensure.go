/*
© 2021 Red Hat, Inc. and others.

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

package servicediscoverycr

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	submariner "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
)

const (
	ServiceDiscoveryName = "service-discovery"
)

func Ensure(config *rest.Config, namespace string, serviceDiscoverySpec submariner.ServiceDiscoverySpec) error {
	serviceDiscoveryCR := &submariner.ServiceDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Name: ServiceDiscoveryName,
		},
		Spec: serviceDiscoverySpec,
	}

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	_, err = createServiceDiscovery(submarinerClient, namespace, serviceDiscoveryCR)

	if err != nil {
		return err
	}

	return nil
}

func createServiceDiscovery(clientSet submarinerclientset.Interface, namespace string, serviceDiscoveryCR *submariner.ServiceDiscovery) (
	bool, error) {
	_, err := clientSet.SubmarinerV1alpha1().ServiceDiscoveries(namespace).Create(serviceDiscoveryCR)
	if err == nil {
		return true, nil
	} else if errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// We can’t always handle existing resources, and we want to overwrite them anyway, so delete them
			err := clientSet.SubmarinerV1alpha1().ServiceDiscoveries(namespace).Delete(serviceDiscoveryCR.Name, &metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("failed to delete pre-existing cfg %s : %s", serviceDiscoveryCR.Name, err)
			}
			_, err = clientSet.SubmarinerV1alpha1().ServiceDiscoveries(namespace).Create(serviceDiscoveryCR)
			if err != nil {
				return fmt.Errorf("failed to create cfg  %s : %s", serviceDiscoveryCR.Name, err)
			}
			return nil
		})
		return false, retryErr
	}
	return false, err
}
