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

package gather

import (
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/client"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

type Info struct {
	RestConfig           *rest.Config
	Status               reporter.Interface
	Submariner           *v1alpha1.Submariner
	ServiceDiscovery     *v1alpha1.ServiceDiscovery
	ClusterName          string
	DirName              string
	IncludeSensitiveData bool
	Summary              *Summary
	ClientProducer       client.Producer
}

type Summary struct {
	Resources []ResourceInfo
	PodLogs   []LogInfo
}

type version struct {
	Subctl    string
	Subm      string
	K8sServer string
}

type clusterConfig struct {
	CNIPlugin        string
	CloudProvider    v1alpha1.CloudProvider
	TotalNode        int
	GatewayNode      map[string]types.UID
	GWNodeNumber     int
	MasterNode       map[string]types.UID
	MasterNodeNumber int
}

type nodeConfig struct {
	Name        string
	Info        v1.NodeSystemInfo
	InternalIPs string
	ExternalIPs string
}

type LogInfo struct {
	PodName      string
	Namespace    string
	NodeName     string
	RestartCount int32
	PodState     v1.PodPhase
	LogFileName  []string
}

type ResourceInfo struct {
	Name      string
	Namespace string
	Type      string
	FileName  string
}

type data struct {
	ClusterName   string
	Versions      version
	ClusterConfig clusterConfig
	NodeConfig    []nodeConfig
	PodLogs       []LogInfo
	ResourceInfo  []ResourceInfo
}
