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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	v1meta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOpenShift4NetworkDiscovery(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network discovery")
}

func fakePod(component string, command []string, env []v1.EnvVar) *v1.Pod {
	return fakePodWithName(component, component, command, env)
}

func fakePodWithName(name, component string, command []string, env []v1.EnvVar) *v1.Pod {
	return fakePodWithNamespace("default", name, component, command, env)
}

func fakePodWithNamespace(namespace, name, component string, command []string, env []v1.EnvVar) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: v1meta.ObjectMeta{
			Namespace: namespace,
			Name:      name,
			Labels:    map[string]string{"component": component, "name": component},
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
