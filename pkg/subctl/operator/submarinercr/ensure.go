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
	"github.com/submariner-io/admiral/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	submariner "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
)

const (
	SubmarinerName = "submariner"
)

func Ensure(config *rest.Config, namespace string, submarinerSpec submariner.SubmarinerSpec) error {
	submarinerCR := &submariner.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name: SubmarinerName,
		},
		Spec: submarinerSpec,
	}

	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}

	client := dynClient.Resource(schema.GroupVersionResource{
		Group:    submariner.SchemeGroupVersion.Group,
		Version:  submariner.SchemeGroupVersion.Version,
		Resource: "submariners"}).Namespace(namespace)
	propagationPolicy := metav1.DeletePropagationForeground

	return util.CreateAnew(client, submarinerCR, &metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
}
