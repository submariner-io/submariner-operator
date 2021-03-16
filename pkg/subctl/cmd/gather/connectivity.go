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

package gather

import (
	"github.com/pkg/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	gatewayPodLabel    = "app=submariner-gateway"
	routeagentPodLabel = "app=submariner-routeagent"
)

func GatherGatewayPodLogs(clientSet kubernetes.Interface, params GatherParams) error {
	pods, err := findPods(clientSet, v1.ListOptions{LabelSelector: gatewayPodLabel})

	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		params.PodName = pod.Name
		err := getPodLogs(clientSet, params)
		if err != nil {
			return errors.WithMessagef(err, "error getting logs for pod %q", pod.Name)
		}
	}
	return nil
}

func GatherRouteagentPodLogs(clientSet kubernetes.Interface, params GatherParams) error {
	pods, err := findPods(clientSet, v1.ListOptions{LabelSelector: routeagentPodLabel})

	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		params.PodName = pod.Name
		err := getPodLogs(clientSet, params)
		if err != nil {
			return errors.WithMessagef(err, "error getting logs for pod %q", pod.Name)
		}
	}
	return nil
}
