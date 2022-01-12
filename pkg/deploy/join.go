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

package deploy

type WithJoinOptions struct {
	PreferredServer               bool
	ForceUDPEncaps                bool
	NatTraversal                  bool
	IgnoreRequirements            bool
	GlobalnetEnabled              bool
	IpsecDebug                    bool
	SubmarinerDebug               bool
	OperatorDebug                 bool
	LoadBalancerEnabled           bool
	HealthCheckEnable             bool
	NattPort                      int
	IkePort                       int
	GlobalnetClusterSize          uint
	HealthCheckInterval           uint64
	HealthCheckMaxPacketLossCount uint64
	ClusterID                     string
	ServiceCIDR                   string
	ClusterCIDR                   string
	GlobalnetCIDR                 string
	Repository                    string
	ImageVersion                  string
	ColorCodes                    string
	CableDriver                   string
	CorednsCustomConfigMap        string
	CustomDomains                 []string
	ImageOverrideArr              []string
}
