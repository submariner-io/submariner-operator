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
	"github.com/submariner-io/submariner-operator/pkg/engine"
	crdutils "github.com/submariner-io/submariner-operator/pkg/utils/crds"
)

const (
	SubmarinerName = "submariner"
)

func Ensure(config *rest.Config, namespace string, submarinerSpec submariner.SubmarinerSpec) error {
	crdUpdater, err := crdutils.NewFromRestConfig(config)
	if err != nil {
		return fmt.Errorf("error connecting to the target cluster: %s", err)
	}
	err = engine.Ensure(crdUpdater)
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

	_, err = updateOrCreateSubmariner(submarinerClient, namespace, submarinerCR)

	if err != nil {
		return err
	}

	return nil
}

func updateOrCreateSubmariner(clientSet submarinerclientset.Interface, namespace string, submarinerCR *submariner.Submariner) (bool,
	error) {
	_, err := clientSet.SubmarinerV1alpha1().Submariners(namespace).Create(submarinerCR)
	if err == nil {
		return true, nil
	} else if errors.IsAlreadyExists(err) {
		retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			existingCfg, err := clientSet.SubmarinerV1alpha1().Submariners(namespace).Get(submarinerCR.Name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get pre-existing cfg %s : %s", submarinerCR.Name, err)
			}
			submarinerCR.ResourceVersion = existingCfg.ResourceVersion
			_, err = clientSet.SubmarinerV1alpha1().Submariners(namespace).Update(submarinerCR)
			if err != nil {
				return fmt.Errorf("failed to update pre-existing cfg  %s : %s", submarinerCR.Name, err)
			}
			return nil
		})
		return false, retryErr
	}
	return false, err
}
