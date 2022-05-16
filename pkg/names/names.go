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

package names

import "fmt"

/* Component names and other constants. */
const (
	NetworkPluginSyncerComponent = "submariner-networkplugin-syncer"
	RouteAgentComponent          = "submariner-routeagent"
	GatewayComponent             = "submariner-gateway"
	GlobalnetComponent           = "submariner-globalnet"
	ServiceDiscoveryComponent    = "submariner-lighthouse-agent"
	LighthouseCoreDNSComponent   = "lighthouse-coredns"
	OperatorComponent            = "submariner-operator"
	ServiceDiscoveryCrName       = "service-discovery"
	SubmarinerCrName             = "submariner"
)

/* These values are used by downstream distributions to override the component default image name. */
var (
	NetworkPluginSyncerImage = "submariner-networkplugin-syncer"
	RouteAgentImage          = "submariner-route-agent"
	GatewayImage             = "submariner-gateway"
	GlobalnetImage           = "submariner-globalnet"
	ServiceDiscoveryImage    = "lighthouse-agent"
	LighthouseCoreDNSImage   = "lighthouse-coredns"
	OperatorImage            = "submariner-operator"
)

/* Deprecated: These values are used by downstream distributions to patch the image names by adding a prefix/suffix. */
var (
	ImagePrefix  = ""
	ImagePostfix = ""
)

var ValidImageNames = []string{
	NetworkPluginSyncerImage, RouteAgentImage, GatewayImage, GlobalnetImage,
	ServiceDiscoveryImage, LighthouseCoreDNSImage, OperatorImage,
}

func AppendUninstall(name string) string {
	return name + "-uninstall"
}

func ForClusterSA(clusterID string) string {
	return fmt.Sprintf("cluster-%s", clusterID)
}
