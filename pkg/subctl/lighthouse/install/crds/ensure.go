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

package crds

import (
	"fmt"
	"os/exec"

	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
)

//go:generate go run generators/yamls2go.go

// Copied over from operator/install/crds/ensure.go

//Ensure functions updates or installs the multiclusterservives CRDs in the cluster
func Ensure(restConfig *rest.Config, kubeConfig string, kubeContext string) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	crd, err := getMcsCRD()
	if err != nil {
		return false, err
	}

	// Attempt to update or create the CRD definition
	// TODO(majopela): In the future we may want to report when we have updated the existing
	//                 CRD definition with new versions
	_, err = utils.CreateOrUpdateCRD(clientSet, crd)
	if err != nil {
		return false, err
	}
	args := []string{"enable", "MulticlusterService"}
	if kubeConfig != "" {
		args = append(args, "--kubeconfig", kubeConfig)
	}
	if kubeContext != "" {
		args = append(args, "--host-cluster-context", kubeContext)
	}
	args = append(args, "--kubefed-namespace", "kubefed-operator")
	out, err := exec.Command("kubefedctl", args...).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("error federating MulticlusterService CRD: %s\n%s", err, out)
	}
	return true, nil
}

func getMcsCRD() (*apiextensionsv1beta1.CustomResourceDefinition, error) {
	crd := &apiextensionsv1beta1.CustomResourceDefinition{}

	if err := embeddedyamls.GetObject(embeddedyamls.Lighthouse_crds_multiclusterservices_crd_yaml, crd); err != nil {
		return nil, err
	}

	return crd, nil
}
