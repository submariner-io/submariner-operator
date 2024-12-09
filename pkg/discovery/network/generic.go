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

package network

import (
	"context"
	"fmt"
	"os"
	"regexp"

	"github.com/pkg/errors"
	"github.com/submariner-io/submariner/pkg/cni"
	corev1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

//nolint:nilnil // Intentional as the purpose is to discover.
func discoverGenericNetwork(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	clusterNetwork, err := discoverNetwork(ctx, client)
	if err != nil {
		return nil, err
	}

	if clusterNetwork != nil {
		clusterNetwork.NetworkPlugin = cni.Generic
		return clusterNetwork, nil
	}

	return nil, nil
}

//nolint:nilnil // Intentional as the purpose is to discover.
func discoverNetwork(ctx context.Context, client controllerClient.Client) (*ClusterNetwork, error) {
	clusterNetwork := &ClusterNetwork{}

	podIPRange, err := findPodIPRange(ctx, client)
	if err != nil {
		return nil, err
	}

	if podIPRange != "" {
		clusterNetwork.PodCIDRs = []string{podIPRange}
	}

	clusterIPRange, err := findClusterIPRange(ctx, client)
	if err != nil {
		return nil, err
	}

	if clusterIPRange != "" {
		clusterNetwork.ServiceCIDRs = []string{clusterIPRange}
	}

	if len(clusterNetwork.PodCIDRs) > 0 || len(clusterNetwork.ServiceCIDRs) > 0 {
		return clusterNetwork, nil
	}

	return nil, nil
}

func findClusterIPRange(ctx context.Context, client controllerClient.Client) (string, error) {
	clusterIPRange, err := findClusterIPRangeFromApiserver(ctx, client)
	if err != nil || clusterIPRange != "" {
		return clusterIPRange, err
	}

	clusterIPRange, err = findClusterIPRangeFromKubeController(ctx, client)
	if err != nil || clusterIPRange != "" {
		return clusterIPRange, err
	}

	clusterIPRange, err = findClusterIPRangeFromServiceCreation(ctx, client)
	if err != nil || clusterIPRange != "" {
		return clusterIPRange, err
	}

	return "", nil
}

func findClusterIPRangeFromApiserver(ctx context.Context, client controllerClient.Client) (string, error) {
	return FindPodCommandParameter(ctx, client, "component=kube-apiserver", "--service-cluster-ip-range")
}

func findClusterIPRangeFromKubeController(ctx context.Context, client controllerClient.Client) (string, error) {
	return FindPodCommandParameter(ctx, client, "component=kube-controller-manager", "--service-cluster-ip-range")
}

func findClusterIPRangeFromServiceCreation(ctx context.Context, client controllerClient.Client) (string, error) {
	ns := os.Getenv("WATCH_NAMESPACE")
	// WATCH_NAMESPACE env should be set to operator's namespace, if running in operator
	if ns == "" {
		// otherwise, it should be called from subctl command, so use "default" namespace
		ns = "default"
	}

	// find service cidr based on https://stackoverflow.com/questions/44190607/how-do-you-find-the-cluster-service-cidr-of-a-kubernetes-cluster
	invalidSvcSpec := &corev1.Service{
		ObjectMeta: v1meta.ObjectMeta{
			Name:      "invalid-svc",
			Namespace: ns,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "1.1.1.1",
			Ports: []corev1.ServicePort{
				{
					Port: 443,
					TargetPort: intstr.IntOrString{
						IntVal: 443,
					},
				},
			},
		},
	}

	// create service to the namespace
	err := client.Create(ctx, invalidSvcSpec)

	// creating invalid service didn't fail as expected
	if err == nil {
		return "", errors.New("could not determine the service IP range via service creation - " +
			"expected a specific error but none was returned")
	}

	return parseServiceCIDRFrom(err.Error())
}

func parseServiceCIDRFrom(msg string) (string, error) {
	// expected msg is below:
	//   "The Service \"invalid-svc\" is invalid: spec.clusterIPs: Invalid value: []string{\"1.1.1.1\"}:
	//   failed to allocated ip:1.1.1.1 with error:provided IP is not in the valid range.
	//   The range of valid IPs is 10.45.0.0/16"
	// expected matched string is below:
	//   10.45.0.0/16
	re := regexp.MustCompile(".*valid IPs is (.*)$")

	match := re.FindStringSubmatch(msg)
	if match == nil {
		return "", fmt.Errorf("could not determine the service IP range via service creation - the expected error "+
			"was not returned. The actual error was %q", msg)
	}

	// returns first matching string
	return match[1], nil
}

func findPodIPRange(ctx context.Context, client controllerClient.Client) (string, error) {
	podIPRange, err := findPodIPRangeFromKubeController(ctx, client)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	podIPRange, err = findPodIPRangeFromKubeProxy(ctx, client)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	podIPRange, err = findPodIPRangeFromNodeSpec(ctx, client)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	return "", nil
}

func findPodIPRangeFromKubeController(ctx context.Context, client controllerClient.Client) (string, error) {
	return FindPodCommandParameter(ctx, client, "component=kube-controller-manager", "--cluster-cidr")
}

func findPodIPRangeFromKubeProxy(ctx context.Context, client controllerClient.Client) (string, error) {
	return FindPodCommandParameter(ctx, client, "k8s-app=kube-proxy", "--cluster-cidr")
}

func findPodIPRangeFromNodeSpec(ctx context.Context, client controllerClient.Client) (string, error) {
	nodes := &corev1.NodeList{}

	err := client.List(ctx, nodes)
	if err != nil {
		return "", errors.WithMessagef(err, "error listing nodes")
	}

	return parseToPodCidr(nodes.Items)
}

func parseToPodCidr(nodes []corev1.Node) (string, error) {
	// In K8s, each node is typically assigned a unique PodCIDR range for the pods that run on that node.
	// Each node's PodCIDR is used to allocate IP addresses to the pods scheduled on that node. Only if
	// the cluster is a single node deployment, we should rely on the node.Spec.PodCIDR as podCIDR of the cluster.
	if len(nodes) == 1 {
		return nodes[0].Spec.PodCIDR, nil
	}

	return "", nil
}
