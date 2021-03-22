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
package resource

import (
	"github.com/submariner-io/shipyard/test/e2e/framework"
	v1 "k8s.io/api/core/v1"
)

func addNodeSelectorTerm(nodeSelTerms []v1.NodeSelectorTerm, label string,
	op v1.NodeSelectorOperator, values []string) []v1.NodeSelectorTerm {
	return append(nodeSelTerms, v1.NodeSelectorTerm{MatchExpressions: []v1.NodeSelectorRequirement{
		{
			Key:      label,
			Operator: op,
			Values:   values,
		},
	}})
}

func nodeAffinity(scheduling framework.NetworkPodScheduling) *v1.Affinity {
	var nodeSelTerms []v1.NodeSelectorTerm

	switch scheduling {
	case framework.GatewayNode:
		nodeSelTerms = addNodeSelectorTerm(nodeSelTerms, framework.GatewayLabel,
			v1.NodeSelectorOpIn, []string{"true"})

	case framework.NonGatewayNode:
		nodeSelTerms = addNodeSelectorTerm(nodeSelTerms, framework.GatewayLabel,
			v1.NodeSelectorOpDoesNotExist, nil)
		nodeSelTerms = addNodeSelectorTerm(nodeSelTerms, framework.GatewayLabel,
			v1.NodeSelectorOpNotIn, []string{"true"})
	}

	return &v1.Affinity{
		NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: nodeSelTerms,
			},
		},
	}
}
