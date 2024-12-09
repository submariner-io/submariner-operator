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

package network_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func init() {
	utilruntime.Must(v1alpha1.AddToScheme(scheme.Scheme))
}

func TestNetworkDiscovery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network Discovery")
}

func fakePod(component string, command []string, env []v1.EnvVar) *v1.Pod {
	return fakePodWithName(component, component, command, env)
}

func fakePodWithArg(component string, command, args []string, env ...v1.EnvVar) *v1.Pod {
	pod := fakePodWithName(component, component, command, env)
	pod.Spec.Containers[0].Args = args

	return pod
}

func fakePodWithName(name, component string, command []string, env []v1.EnvVar) *v1.Pod {
	return fakePodWithNamespace("default", name, component, command, env)
}

func fakePodWithNamespace(namespace, name, component string, command []string, env []v1.EnvVar) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: v1meta.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    map[string]string{"component": component, "name": component, "app": component, "k8s-app": component},
		},

		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Command: command,
					Env:     env,
				},
			},
		},
	}
}

func fakeKubeAPIServerPod() *v1.Pod {
	return fakePod("kube-apiserver", []string{"kube-apiserver", "--service-cluster-ip-range=" + testServiceCIDR}, []v1.EnvVar{})
}

func fakeKubeControllerManagerPod() *v1.Pod {
	return fakePod("kube-controller-manager", []string{
		"kube-controller-manager", "--cluster-cidr=" + testPodCIDR,
		"--service-cluster-ip-range=" + testServiceCIDR,
	}, []v1.EnvVar{})
}

func fakeKubeProxyPod() *v1.Pod {
	return fakePod("kube-proxy", []string{"kube-proxy", "--cluster-cidr=" + testPodCIDR}, []v1.EnvVar{})
}

func fakeService(namespace, name, component string) *v1.Service {
	return &v1.Service{
		ObjectMeta: v1meta.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    map[string]string{"component": component, "name": component},
		},
		Spec: v1.ServiceSpec{},
	}
}

func fakeNode(name, podCIDR string) *v1.Node {
	return &v1.Node{
		ObjectMeta: v1meta.ObjectMeta{
			Name: name,
		},
		Spec: v1.NodeSpec{
			PodCIDR: podCIDR,
		},
	}
}

func testDiscoverNetworkSuccess(ctx context.Context, objects ...controllerClient.Object) *network.ClusterNetwork {
	clusterNet, err := testDiscoverNetwork(ctx, objects...)
	Expect(err).NotTo(HaveOccurred())

	return clusterNet
}

func testDiscoverNetwork(ctx context.Context, objects ...controllerClient.Object) (*network.ClusterNetwork, error) {
	client := newTestClient(objects...)
	return network.Discover(ctx, client, "")
}
