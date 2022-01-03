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
	operatorclient "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	submarinerclient "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

type DefaultProducer struct {
	CRDClient        apiextclient.Interface
	KubeClient       kubernetes.Interface
	DynamicClient    dynamic.Interface
	OperatorClient   operatorclient.Interface
	SubmarinerClient submarinerclient.Interface
}

func (p *DefaultProducer) ForCRD() apiextclient.Interface {
	return p.CRDClient
}

func (p *DefaultProducer) ForKubernetes() kubernetes.Interface {
	return p.KubeClient
}

func (p *DefaultProducer) ForDynamic() dynamic.Interface {
	return p.DynamicClient
}

func (p *DefaultProducer) ForSubmariner() submarinerclient.Interface {
	return p.SubmarinerClient
}

func (p *DefaultProducer) ForOperator() operatorclient.Interface {
	return p.OperatorClient
}
