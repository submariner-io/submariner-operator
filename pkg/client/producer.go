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

package client

import (
	operatorClientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	submarinerClientset "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
	apiextClient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type Producer interface {
	ForCRD() apiextClient.Interface
	ForKubernetes() kubernetes.Interface
	ForDynamic() dynamic.Interface
	ForOperator() operatorClientset.Interface
	ForSubmariner() submarinerClientset.Interface
}
