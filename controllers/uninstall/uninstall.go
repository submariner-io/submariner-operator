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

package uninstall

import (
	"context"
	"fmt"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ComponentReadyTimeout = time.Minute * 2
	ContainerEnvVar       = "SUBMARINER_UNINSTALL"
)

var minComponentUninstallVersion = semver.New("0.12.0")

type Component struct {
	Resource          client.Object
	UninstallResource client.Object
	CheckInstalled    func() bool
}

type Info struct {
	Client     client.Client
	Components []*Component
	StartTime  time.Time
	Log        logr.Logger
}

func (c *Component) isInstalled() bool {
	return c.CheckInstalled == nil || c.CheckInstalled()
}

func (i *Info) Run(ctx context.Context) (bool, error) {
	timedOut := time.Since(i.StartTime) >= ComponentReadyTimeout
	if timedOut {
		i.Log.Info("Timed out waiting for components to complete - aborting")

		i.cleanup(ctx)

		return false, nil
	}

	// First, delete the regular DaemonSets/Deployments.
	for _, c := range i.Components {
		if !c.isInstalled() {
			continue
		}

		err := i.ensureDeleted(ctx, c.Resource)
		if err != nil {
			return false, err
		}
	}

	// Next, ensure all their pods are cleaned up.
	podsIncomplete, err := i.ensurePodsDeleted(ctx)
	if err != nil {
		return false, err
	}

	err = i.createUninstallResources(ctx)
	if err != nil {
		return false, err
	}

	uninstallIncomplete, err := i.ensureUninstallResourcesComplete(ctx)
	if err != nil {
		return false, err
	}

	if podsIncomplete || uninstallIncomplete {
		return true, nil
	}

	i.cleanup(ctx)

	return false, nil
}

func (i *Info) ensureDeleted(ctx context.Context, obj client.Object) error {
	err := i.Client.Delete(ctx, obj)
	if apierrors.IsNotFound(err) {
		return nil
	}

	if err == nil {
		i.Log.Info(fmt.Sprintf("Deleted %T:", obj), "name", obj.GetName(), "namespace", obj.GetNamespace())
	}

	return errors.Wrapf(err, "error deleting %#v", obj)
}

func (i *Info) ensurePodsDeleted(ctx context.Context) (bool, error) {
	requeue := false

	for _, c := range i.Components {
		if !c.isInstalled() {
			continue
		}

		pods, err := findPodsBySelector(ctx, i.Client, c.Resource.GetNamespace(), &metav1.LabelSelector{
			MatchLabels: map[string]string{"app": c.Resource.GetName()},
		})
		if err != nil {
			return false, err
		}

		numPods := len(pods)
		if numPods > 0 {
			i.Log.Info(fmt.Sprintf("%T still has pods - requeueing: ", c.Resource), "name", c.Resource.GetName(),
				"namespace", c.Resource.GetNamespace(), "numPods", numPods)

			requeue = true
		}
	}

	return requeue, nil
}

func (i *Info) createUninstallResources(ctx context.Context) error {
	for _, c := range i.Components {
		if !c.isInstalled() {
			continue
		}

		var err error

		switch d := c.UninstallResource.(type) {
		case *appsv1.DaemonSet:
			err = i.createUninstallDaemonSetFrom(ctx, d)
		case *appsv1.Deployment:
			err = i.createUninstallDeploymentFrom(ctx, d)
		default:
			panic(fmt.Sprintf("Unknown type: %T", d))
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (i *Info) createUninstallDeploymentFrom(ctx context.Context, deployment *appsv1.Deployment) error {
	convertPodSpecContainersToUninstall(&deployment.Spec.Template.Spec)

	err := i.Client.Create(ctx, deployment)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "error creating %#v", deployment)
		}
	} else {
		i.Log.Info("Created Deployment:", "name", deployment.Name, "namespace", deployment.Namespace)
	}

	return nil
}

func (i *Info) createUninstallDaemonSetFrom(ctx context.Context, daemonSet *appsv1.DaemonSet) error {
	convertPodSpecContainersToUninstall(&daemonSet.Spec.Template.Spec)

	err := i.Client.Create(ctx, daemonSet)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "error creating %#v", daemonSet)
		}
	} else {
		i.Log.Info("Created DaemonSet:", "name", daemonSet.Name, "namespace", daemonSet.Namespace,
			"Image", daemonSet.Spec.Template.Spec.InitContainers[0].Image)
	}

	return nil
}

func (i *Info) ensureUninstallResourcesComplete(ctx context.Context) (bool, error) {
	anyRequeue := false

	for _, c := range i.Components {
		if !c.isInstalled() {
			continue
		}

		var requeue bool
		var err error

		switch d := c.UninstallResource.(type) {
		case *appsv1.DaemonSet:
			requeue, err = i.ensureDaemonSetReady(ctx, client.ObjectKeyFromObject(d))
		case *appsv1.Deployment:
			requeue, err = i.ensureDeploymentReady(ctx, client.ObjectKeyFromObject(d))
		default:
			panic(fmt.Sprintf("Unknown type: %T", d))
		}

		if err != nil {
			return false, err
		}

		anyRequeue = anyRequeue || requeue
	}

	return anyRequeue, nil
}

func (i *Info) ensureDaemonSetReady(ctx context.Context, key client.ObjectKey) (bool, error) {
	daemonSet := &appsv1.DaemonSet{}

	err := i.Client.Get(ctx, key, daemonSet)
	if err != nil {
		return false, errors.Wrapf(err, "error getting %#v", daemonSet)
	}

	if daemonSet.Status.ObservedGeneration == 0 || daemonSet.Status.ObservedGeneration < daemonSet.Generation {
		i.Log.Info("DaemonSet generation not yet observed - requeueing:", "name", daemonSet.Name,
			"namespace", daemonSet.Namespace, "Generation", daemonSet.Generation, "ObservedGeneration", daemonSet.Status.ObservedGeneration)
		return true, nil
	}

	if daemonSet.Status.DesiredNumberScheduled == 0 {
		i.Log.Info("DaemonSet has no available nodes:", "name", daemonSet.Name, "namespace", daemonSet.Namespace)
	} else if daemonSet.Status.DesiredNumberScheduled != daemonSet.Status.NumberReady {
		i.Log.Info("DaemonSet not ready yet:", "name", daemonSet.Name, "namespace", daemonSet.Namespace,
			"DesiredNumberScheduled", daemonSet.Status.DesiredNumberScheduled, "NumberReady", daemonSet.Status.NumberReady)
		return true, nil
	} else {
		i.Log.Info("DaemonSet is ready:", "name", daemonSet.Name, "namespace", daemonSet.Namespace)
	}

	return false, nil
}

func (i *Info) ensureDeploymentReady(ctx context.Context, key client.ObjectKey) (bool, error) {
	deployment := &appsv1.Deployment{}

	err := i.Client.Get(ctx, key, deployment)
	if err != nil {
		return false, errors.Wrapf(err, "error getting %#v", deployment)
	}

	var replicas int32 = 1
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	if deployment.Status.AvailableReplicas != replicas {
		i.Log.Info("Deployment not ready yet:", "name", deployment.Name, "namespace", deployment.Namespace,
			"AvailableReplicas", deployment.Status.AvailableReplicas, "DesiredReplicas", replicas)
		return true, nil
	}

	i.Log.Info("Deployment is ready:", "name", deployment.Name, "namespace", deployment.Namespace)

	return false, nil
}

func (i *Info) cleanup(ctx context.Context) {
	for _, c := range i.Components {
		err := i.ensureDeleted(ctx, c.UninstallResource)
		if err != nil {
			i.Log.Error(err, "Unable to delete uninstall resource", "name", c.UninstallResource.GetName(),
				"namespace", c.UninstallResource.GetNamespace())
		}
	}
}

func convertPodSpecContainersToUninstall(podSpec *corev1.PodSpec) {
	// We're going to use the PodSpec to run a one-time task by using an init container to run the task.
	// See http://blog.itaysk.com/2017/12/26/the-single-use-daemonset-pattern-and-prepulling-images-in-kubernetes
	// for more details.
	podSpec.InitContainers = podSpec.Containers
	podSpec.InitContainers[0].Env = append(podSpec.InitContainers[0].Env, corev1.EnvVar{Name: ContainerEnvVar, Value: "true"})

	podSpec.Containers = []corev1.Container{
		{
			Name:  "pause",
			Image: "gcr.io/google_containers/pause",
		},
	}
}

func findPodsBySelector(ctx context.Context, clnt client.Reader, namespace string,
	labelSelector *metav1.LabelSelector) ([]corev1.Pod, error) {
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return nil, errors.Wrap(err, "error creating label selector")
	}

	pods := &corev1.PodList{}

	err = clnt.List(ctx, pods, client.InNamespace(namespace), client.MatchingLabelsSelector{Selector: selector})
	if err != nil {
		return nil, errors.Wrap(err, "error listing DaemonSet pods")
	}

	return pods.Items, nil
}

func IsSupportedForVersion(version string) bool {
	semVersion, _ := semver.NewVersion(version)
	return !(semVersion != nil && !semVersion.LessThan(*minComponentUninstallVersion))
}
