/*
© 2019 Red Hat, Inc. and others.

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
	"fmt"
	"os"
	"regexp"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func discoverGenericNetwork(clientSet kubernetes.Interface) (*ClusterNetwork, error) {
	clusterNetwork := &ClusterNetwork{
		NetworkPlugin: "generic",
	}

	podIPRange, err := findPodIPRange(clientSet)
	if err != nil {
		return nil, err
	}

	if podIPRange != "" {
		clusterNetwork.PodCIDRs = []string{podIPRange}
	}

	clusterIPRange, err := findClusterIPRange(clientSet)
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

func findClusterIPRange(clientSet kubernetes.Interface) (string, error) {
	clusterIPRange, err := findClusterIPRangeFromApiserver(clientSet)
	if err != nil || clusterIPRange != "" {
		return clusterIPRange, err
	}

	clusterIPRange, err = findClusterIPRangeFromServiceCreation(clientSet)
	if err != nil || clusterIPRange != "" {
		return clusterIPRange, err
	}

	return "", nil
}

func findClusterIPRangeFromApiserver(clientSet kubernetes.Interface) (string, error) {
	return findPodCommandParameter(clientSet, "component=kube-apiserver", "--service-cluster-ip-range")
}

func findClusterIPRangeFromServiceCreation(clientSet kubernetes.Interface) (string, error) {
	// find service cidr based on https://stackoverflow.com/questions/44190607/how-do-you-find-the-cluster-service-cidr-of-a-kubernetes-cluster
	invalidSvcSpec := &v1.Service{
		ObjectMeta: v1meta.ObjectMeta{
			Name: "invalid-svc",
		},
		Spec: v1.ServiceSpec{
			ClusterIP: "1.1.1.1",
			Ports: []v1.ServicePort{
				{
					Port: 443,
					TargetPort: intstr.IntOrString{
						IntVal: 443,
					},
				},
			},
		},
	}

	ns := os.Getenv("WATCH_NAMESPACE")
	// WATCH_NAMESPACE env should be set to operator's namespace, if running in operator
	if ns == "" {
		// otherwise, it should be called from subctl command, so use "default" namespace
		ns = "default"
	}

	// create service to the namespace
	_, err := clientSet.CoreV1().Services(ns).Create(invalidSvcSpec)

	// creating invalid service didn't fail as expected
	if err == nil {
		return "", fmt.Errorf("could not determine the service IP range via service creation - " +
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

func findPodIPRange(clientSet kubernetes.Interface) (string, error) {
	podIPRange, err := findPodIPRangeKubeController(clientSet)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	podIPRange, err = findPodIPRangeKubeProxy(clientSet)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	podIPRange, err = findPodIPRangeFromNodeSpec(clientSet)
	if err != nil || podIPRange != "" {
		return podIPRange, err
	}

	return "", nil
}

func findPodIPRangeKubeController(clientSet kubernetes.Interface) (string, error) {
	return findPodCommandParameter(clientSet, "component=kube-controller-manager", "--cluster-cidr")
}

func findPodIPRangeKubeProxy(clientSet kubernetes.Interface) (string, error) {
	return findPodCommandParameter(clientSet, "component=kube-proxy", "--cluster-cidr")
}

func findPodIPRangeFromNodeSpec(clientSet kubernetes.Interface) (string, error) {
	nodes, err := clientSet.CoreV1().Nodes().List(v1meta.ListOptions{})

	if err != nil {
		return "", errors.WithMessagef(err, "error listing nodes")
	}

	return parseToPodCidr(nodes.Items)
}

func parseToPodCidr(nodes []v1.Node) (string, error) {
	for _, node := range nodes {
		if node.Spec.PodCIDR != "" {
			return node.Spec.PodCIDR, nil
		}
	}

	return "", nil
}
