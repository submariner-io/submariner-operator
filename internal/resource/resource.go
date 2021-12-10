/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

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
package resource

import (
	"context"
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/internal/constants"
	subOperatorClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/subctl/cmd/utils"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/submarinercr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1opts "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

func getSubmarinerWithError(config *rest.Config) (*v1alpha1.Submariner, error) {
	submarinerClient, err := subOperatorClientset.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	submariner, err := submarinerClient.SubmarinerV1alpha1().Submariners(constants.OperatorNamespace).
		Get(context.TODO(), submarinercr.SubmarinerName, v1opts.GetOptions{})
	if err != nil {
		return nil, err
	}

	return submariner, nil
}

func GetSubmariner(config *rest.Config) *v1alpha1.Submariner {
	submariner, err := getSubmarinerWithError(config)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		utils.ExitOnError("Error obtaining the Submariner resource", err)
	}

	return submariner
}

