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

	submariner "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/engine"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/retry"
)

func Ensure(config *rest.Config, namespace string, submarinerSpec submariner.SubmarinerSpec) error {

	err := engine.Ensure(config)
	if err != nil {
		return fmt.Errorf("error setting up the engine requirements: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the core kubernetes clientset: %s", err)
	}

	// Create the namespace
	_, err = clientset.CoreV1().Namespaces().Create(NewSubmarinerNamespace())
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the Submariner namespace %s", err)
	}

	submariner := &submariner.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name: "submariner",
		},
		Spec: submarinerSpec,
	}

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	_, err = updateOrCreateSubmariner(submarinerClient, namespace, submariner)

	if err != nil {
		return err
	}

	fmt.Printf("* Submariner is up and running\n")

	return nil
}

func updateOrCreateSubmariner(clientSet submarinerclientset.Interface, namespace string, submariner *submariner.Submariner) (bool, error) {
	_, err := clientSet.SubmarinerV1alpha1().Submariners(namespace).Create(submariner)
	if err == nil {
		return true, nil
	} else if errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingCfg, err := clientSet.SubmarinerV1alpha1().Submariners(namespace).Get(submariner.Name, v1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get pre-existing cfg %s : %s", submariner.Name, err)
			}
			submariner.ResourceVersion = existingCfg.ResourceVersion
			_, err = clientSet.SubmarinerV1alpha1().Submariners(namespace).Update(submariner)
			if err != nil {
				return fmt.Errorf("failed to update pre-existing cfg  %s : %s", submariner.Name, err)
			}
			return nil
		})
		return false, retryErr
	}
	return false, err
}
