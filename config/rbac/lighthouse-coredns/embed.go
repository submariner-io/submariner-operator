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

// Package lighthousecoredns embeds the Lighthouse CoreDNS RBAC YAML files
package lighthousecoredns

import _ "embed"

var (
	//go:embed cluster_role_binding.yaml
	ClusterRoleBinding []byte

	//go:embed cluster_role.yaml
	ClusterRole []byte

	//go:embed ocp_cluster_role_binding.yaml
	OCPClusterRoleBinding []byte

	//go:embed ocp_cluster_role.yaml
	OCPClusterRole []byte

	//go:embed role_binding.yaml
	RoleBinding []byte

	//go:embed role.yaml
	Role []byte

	//go:embed service_account.yaml
	ServiceAccount []byte
)
