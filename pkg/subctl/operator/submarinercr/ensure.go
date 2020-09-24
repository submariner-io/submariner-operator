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

package submarinercr

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"

	submariner "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
)

const (
	SubmarinerName = "submariner"
)

func Ensure(config *rest.Config, namespace string, submarinerSpec submariner.SubmarinerSpec) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the core kubernetes clientset: %s", err)
	}

	// Create the namespace
	_, err = clientset.CoreV1().Namespaces().Create(NewSubmarinerNamespace())
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the Submariner namespace %s", err)
	}

	submarinerCR := &submariner.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name: SubmarinerName,
		},
		Spec: submarinerSpec,
	}

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	_, err = createSubmariner(submarinerClient, namespace, submarinerCR)

	if err != nil {
		return err
	}

	return nil
}

func createSubmariner(clientSet submarinerclientset.Interface, namespace string, submarinerCR *submariner.Submariner) (bool, error) {
	_, err := clientSet.SubmarinerV1alpha1().Submariners(namespace).Create(submarinerCR)
	if err == nil {
		return true, nil
	} else if errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			// We can’t always handle existing resources, and we want to overwrite them anyway, so delete them
			err := clientSet.SubmarinerV1alpha1().Submariners(namespace).Delete(submarinerCR.Name, &metav1.DeleteOptions{})
			if err != nil {
				return fmt.Errorf("failed to delete pre-existing cfg %s : %s", submarinerCR.Name, err)
			}
			_, err = clientSet.SubmarinerV1alpha1().Submariners(namespace).Create(submarinerCR)
			if err != nil {
				return fmt.Errorf("failed to create cfg  %s : %s", submarinerCR.Name, err)
			}
			return nil
		})
		return false, retryErr
	}
	return false, err
}
