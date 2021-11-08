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

package submariner

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
)

func updateDaemonSetStatus(clnt client.Reader, ctx context.Context, daemonSet *appsv1.DaemonSet, status *v1alpha1.DaemonSetStatus,
	namespace string) error {
	if daemonSet != nil {
		if status == nil {
			status = &v1alpha1.DaemonSetStatus{}
		}
		status.Status = &daemonSet.Status
		if status.LastResourceVersion != daemonSet.ObjectMeta.ResourceVersion {
			// The daemonset has changed, check its containers
			mismatchedContainerImages, nonReadyContainerStates, err :=
				checkDaemonSetContainers(clnt, ctx, daemonSet, namespace)
			if err != nil {
				return err
			}
			status.MismatchedContainerImages = mismatchedContainerImages
			status.NonReadyContainerStates = nonReadyContainerStates
			status.LastResourceVersion = daemonSet.ObjectMeta.ResourceVersion
		}
	}
	return nil
}

func checkDaemonSetContainers(clnt client.Reader, ctx context.Context, daemonSet *appsv1.DaemonSet,
	namespace string) (bool, *[]corev1.ContainerState, error) {
	containerStatuses, err := retrieveDaemonSetContainerStatuses(clnt, ctx, daemonSet, namespace)
	if err != nil {
		return false, nil, err
	}
	var containerImageManifest *string = nil
	var mismatchedContainerImages = false
	var nonReadyContainerStates = []corev1.ContainerState{}
	for i := range *containerStatuses {
		if containerImageManifest == nil {
			containerImageManifest = &((*containerStatuses)[i].ImageID)
		} else if *containerImageManifest != (*containerStatuses)[i].ImageID {
			// Container mismatch
			mismatchedContainerImages = true
		}
		if !*(*containerStatuses)[i].Started {
			// Not (yet) ready
			nonReadyContainerStates = append(nonReadyContainerStates, (*containerStatuses)[i].State)
		}
	}
	return mismatchedContainerImages, &nonReadyContainerStates, nil
}

func retrieveDaemonSetContainerStatuses(clnt client.Reader, ctx context.Context, daemonSet *appsv1.DaemonSet,
	namespace string) (*[]corev1.ContainerStatus, error) {
	pods := &corev1.PodList{}
	selector, err := metav1.LabelSelectorAsSelector(daemonSet.Spec.Selector)
	if err != nil {
		return nil, err
	}
	err = clnt.List(ctx, pods, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, err
	}
	containerStatuses := []corev1.ContainerStatus{}
	for i := range pods.Items {
		containerStatuses = append(containerStatuses, pods.Items[i].Status.ContainerStatuses...)
	}
	return &containerStatuses, nil
}
