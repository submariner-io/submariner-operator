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
	"context"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	subctlversion "github.com/submariner-io/submariner-operator/pkg/version"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

// Embed the file content as string.
//go:embed layout.gohtml
var layout string

func ClusterSummary(info Info) {
	dataGathered := getClusterInfo(info)
	file := createFile(info.DirName)
	writeToHTML(file, dataGathered)
}

func getClusterInfo(info Info) data {
	versions := getVersions(info)
	config := getClusterConfig(info)
	nConfig, err := getNodeConfig(info)
	if err != nil {
		fmt.Println(err)
	}

	d := data{
		ClusterName:   info.ClusterName,
		Versions:      versions,
		ClusterConfig: config,
		NodeConfig:    nConfig,
		PodLogs:       info.Summary.PodLogs,
		ResourceInfo:  info.Summary.Resources,
	}
	return d
}

func getClusterConfig(info Info) clusterConfig {
	gwNodes, err := getGWNodes(info)
	if err != nil {
		fmt.Println(err)
	}
	mNodes, err := getMasterNodes(info)
	if err != nil {
		fmt.Println(err)
	}
	allNodes, err := listNodes(info, metav1.ListOptions{})
	if err != nil {
		fmt.Println(err)
	}
	config := clusterConfig{
		TotalNode:        len(allNodes.Items),
		GatewayNode:      gwNodes,
		GWNodeNumber:     len(gwNodes),
		MasterNode:       mNodes,
		MasterNodeNumber: len(mNodes),
	}

	config.CNIPlugin = "Not found"
	config.CloudProvider = "N/A" // Broker clusters won't have Submariner to gather information from
	if info.Submariner != nil {
		config.CNIPlugin = info.Submariner.Status.NetworkPlugin
		fmt.Println(info.Submariner.Status)
		config.CloudProvider = info.Submariner.Status.DeploymentInfo.CloudProvider
	}
	return config
}

func getVersions(info Info) version {
	Versions := version{
		Subctl: subctlversion.Version,
	}

	k8sServerVersion, err := info.ClientSet.Discovery().ServerVersion()
	if err != nil {
		fmt.Println("error in getting k8s server version", err)
		Versions.K8sServer = err.Error()
	}
	Versions.K8sServer = k8sServerVersion.String()

	Versions.Subm = "Not installed"
	if info.Submariner != nil {
		Versions.Subm = info.Submariner.Spec.Version
	}
	return Versions
}

func getSpecificNode(info Info, selector string) (map[string]types.UID, error) {
	nodes, err := listNodes(info, metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		return nil, err
	}
	node := make(map[string]types.UID, len(nodes.Items))
	for _, n := range nodes.Items {
		node[n.GetName()] = n.GetUID()
	}
	return node, nil
}

func getGWNodes(info Info) (map[string]types.UID, error) {
	selector := labels.SelectorFromSet(labels.Set(map[string]string{"submariner.io/gateway": "true"}))
	nodes, err := getSpecificNode(info, selector.String())
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func getMasterNodes(info Info) (map[string]types.UID, error) {
	selector := "node-role.kubernetes.io/master="
	nodes, err := getSpecificNode(info, selector)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func getNodeConfig(info Info) ([]nodeConfig, error) {
	nodes, err := listNodes(info, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	nodeConfigs := make([]nodeConfig, len(nodes.Items))
	for _, allNode := range nodes.Items {
		nodeInfo := v1.NodeSystemInfo{
			KernelVersion:           allNode.Status.NodeInfo.KernelVersion,
			OSImage:                 allNode.Status.NodeInfo.OSImage,
			ContainerRuntimeVersion: allNode.Status.NodeInfo.ContainerRuntimeVersion,
			KubeletVersion:          allNode.Status.NodeInfo.KubeletVersion,
			KubeProxyVersion:        allNode.Status.NodeInfo.KubeProxyVersion,
			OperatingSystem:         allNode.Status.NodeInfo.OperatingSystem,
			Architecture:            allNode.Status.NodeInfo.Architecture,
		}
		name := allNode.GetName()
		config := nodeConfig{
			Name: name,
			Info: nodeInfo,
		}
		nodeConfigs = append(nodeConfigs, config)
	}
	return nodeConfigs, nil
}

func listNodes(info Info, listOptions metav1.ListOptions) (*v1.NodeList, error) {
	nodes, err := info.ClientSet.CoreV1().Nodes().List(context.TODO(), listOptions)
	if err != nil {
		return nil, err
	}
	return nodes, nil
}

func createFile(dirname string) io.Writer {
	fileName := filepath.Join(dirname, "summary.html")
	f, err := os.OpenFile(fileName, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		fmt.Printf("Error creating file %s\n", fileName)
	}
	return f
}

func writeToHTML(fileWriter io.Writer, cData data) {
	t := template.Must(template.New("layout.html").Parse(layout))
	err := t.Execute(fileWriter, cData)
	if err != nil {
		fmt.Println(err)
	}
}
