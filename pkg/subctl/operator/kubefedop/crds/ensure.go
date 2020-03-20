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

package crds

import (
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/rest"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/common/embeddedyamls"
	"github.com/submariner-io/submariner-operator/pkg/utils"
)

//go:generate go run generators/yamls2go.go

//Ensure functions updates or installs the operator CRDs in the cluster
func Ensure(restConfig *rest.Config) (bool, error) {
	clientSet, err := clientset.NewForConfig(restConfig)
	if err != nil {
		return false, err
	}

	updatedCrds := false
	crds, err := getKubeFedCRDs()
	if err != nil {
		return false, err
	}

	for _, crd := range crds {
		updated, err := utils.CreateOrUpdateCRD(clientSet, crd)
		if err != nil {
			return false, err
		}
		updatedCrds = updatedCrds || updated
	}
	return updatedCrds, nil

}

var (
	kubeFedCRDs = [...]string{
		embeddedyamls.Kubefed_clusterpropagatedversions_core_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_dnsendpoints_multiclusterdns_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_domains_multiclusterdns_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_federatedservicestatuses_core_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_federatedtypeconfigs_core_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_ingressdnsrecords_multiclusterdns_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_kubefedclusters_core_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_kubefedconfigs_core_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_kubefeds_operator_kubefed_io_crd_yaml,
		embeddedyamls.Kubefed_propagatedversions_core_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_replicaschedulingpreferences_scheduling_kubefed_k8s_io_crd_yaml,
		embeddedyamls.Kubefed_servicednsrecords_multiclusterdns_kubefed_k8s_io_crd_yaml,
	}
)

func getKubeFedCRDs() ([]*apiextensionsv1beta1.CustomResourceDefinition, error) {
	var crds = []*apiextensionsv1beta1.CustomResourceDefinition{}
	for _, crdName := range kubeFedCRDs {
		crd := &apiextensionsv1beta1.CustomResourceDefinition{}

		if err := embeddedyamls.GetObject(crdName, crd); err != nil {
			return nil, err
		}

		crds = append(crds, crd)
	}
	return crds, nil
}
