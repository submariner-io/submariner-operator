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
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/finalizer"
	operatorv1alpha1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/resource"
	"github.com/submariner-io/submariner-operator/pkg/names"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var requeueAfter = reconcile.Result{RequeueAfter: time.Millisecond * 100}

type component struct {
	resource          client.Object
	uninstallResource client.Object
	checkInstalled    func() bool
}

func (c *component) isInstalled() bool {
	return c.checkInstalled == nil || c.checkInstalled()
}

func (r *Reconciler) runComponentCleanup(ctx context.Context, instance *operatorv1alpha1.Submariner) (reconcile.Result, error) {
	// This has the side effect of setting the CIDRs in the Submariner instance.
	_, err := r.discoverNetwork(instance)
	if err != nil {
		return reconcile.Result{}, err
	}

	components := []*component{
		{
			resource:          newDaemonSet(names.GatewayComponent, instance.Namespace),
			uninstallResource: newGatewayDaemonSet(instance, names.AppendUninstall(names.GatewayComponent)),
		},
		{
			resource:          newDaemonSet(names.RouteAgentComponent, instance.Namespace),
			uninstallResource: newRouteAgentDaemonSet(instance, names.AppendUninstall(names.RouteAgentComponent)),
		},
		{
			resource:          newDaemonSet(names.GlobalnetComponent, instance.Namespace),
			uninstallResource: newGlobalnetDaemonSet(instance, names.AppendUninstall(names.GlobalnetComponent)),
			checkInstalled: func() bool {
				return instance.Spec.GlobalCIDR != ""
			},
		},
	}

	// First, delete the regular DaemonSets/Deployments.
	for _, c := range components {
		if !c.isInstalled() {
			continue
		}

		err := r.ensureDeleted(ctx, c.resource)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Next, ensure all their pods are cleaned up.
	requeue, err := r.ensurePodsDeleted(ctx, components)
	if err != nil {
		return reconcile.Result{}, err
	}

	if requeue {
		return requeueAfter, nil
	}

	err = r.createUninstallResources(ctx, components)
	if err != nil {
		return reconcile.Result{}, err
	}

	requeue, err = r.ensureUninstallResourcesComplete(ctx, components)
	if err != nil {
		return reconcile.Result{}, err
	}

	if requeue {
		return requeueAfter, nil
	}

	err = finalizer.Remove(ctx, resource.ForControllerClient(r.config.Client, instance.Namespace, &operatorv1alpha1.Submariner{}),
		instance, SubmarinerFinalizer)

	return reconcile.Result{}, err // nolint:wrapcheck // No need to wrap
}

func (r *Reconciler) ensurePodsDeleted(ctx context.Context, components []*component) (bool, error) {
	for _, c := range components {
		if !c.isInstalled() {
			continue
		}

		pods, err := findPodsBySelector(ctx, r.config.Client, c.resource.GetNamespace(), &metav1.LabelSelector{
			MatchLabels: map[string]string{appLabel: c.resource.GetName()},
		})
		if err != nil {
			return false, err
		}

		numPods := len(pods)
		if numPods > 0 {
			log.Info(fmt.Sprintf("%T still has pods - requeueing: ", c.resource), "name", c.resource.GetName(),
				"namespace", c.resource.GetNamespace(), "numPods", numPods)
			return true, nil
		}
	}

	return false, nil
}

func (r *Reconciler) ensureUninstallResourcesComplete(ctx context.Context, components []*component) (bool, error) {
	for _, c := range components {
		if !c.isInstalled() {
			continue
		}

		var requeue bool
		var err error

		switch d := c.uninstallResource.(type) {
		case *appsv1.DaemonSet:
			requeue, err = r.ensureDaemonSetReady(ctx, client.ObjectKeyFromObject(d))
		}

		if err != nil {
			return false, err
		}

		if requeue {
			// TODO - check the elapased time wrt the Submariner DeletionTimeStamp and abort cleanup if too long
			return true, nil
		}
	}

	return false, nil
}

func (r *Reconciler) createUninstallResources(ctx context.Context, components []*component) error {
	for _, c := range components {
		if !c.isInstalled() {
			continue
		}

		var err error

		switch d := c.uninstallResource.(type) {
		case *appsv1.DaemonSet:
			err = r.createUninstallDaemonSetFrom(ctx, d)
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Reconciler) ensureDeleted(ctx context.Context, obj client.Object) error {
	err := r.config.Client.Delete(ctx, obj)
	if apierrors.IsNotFound(err) {
		return nil
	}

	if err == nil {
		log.Info(fmt.Sprintf("Deleted %T:", obj), "name", obj.GetName(), "namespace", obj.GetNamespace())
	}

	return errors.Wrapf(err, "error deleting %#v", obj)
}

func (r *Reconciler) createUninstallDaemonSetFrom(ctx context.Context, daemonSet *appsv1.DaemonSet) error {
	convertPodSpecContainersToUninstall(&daemonSet.Spec.Template.Spec)

	err := r.config.Client.Create(ctx, daemonSet)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return errors.Wrapf(err, "error creating %#v", daemonSet)
		}
	} else {
		log.Info("Created DaemonSet:", "name", daemonSet.Name, "namespace", daemonSet.Namespace)
	}

	return nil
}

func (r *Reconciler) ensureDaemonSetReady(ctx context.Context, key client.ObjectKey) (bool, error) {
	daemonSet := &appsv1.DaemonSet{}

	err := r.config.Client.Get(ctx, key, daemonSet)
	if err != nil {
		return false, errors.Wrapf(err, "error getting %#v", daemonSet)
	}

	if daemonSet.Status.ObservedGeneration == 0 || daemonSet.Status.ObservedGeneration < daemonSet.Generation {
		log.Info("DaemonSet generation not yet observed - requeueing:", "name", daemonSet.Name,
			"namespace", daemonSet.Namespace, "Generation", daemonSet.Generation, "ObservedGeneration", daemonSet.Status.ObservedGeneration)
		return true, nil
	}

	if daemonSet.Status.DesiredNumberScheduled == 0 {
		log.Info("DaemonSet has no available nodes:", "name", daemonSet.Name, "namespace", daemonSet.Namespace)
	} else if daemonSet.Status.DesiredNumberScheduled != daemonSet.Status.NumberReady {
		log.Info("DaemonSet not ready yet - requeueing:", "name", daemonSet.Name, "namespace", daemonSet.Namespace,
			"DesiredNumberScheduled", daemonSet.Status.DesiredNumberScheduled, "NumberReady", daemonSet.Status.NumberReady)
		return true, nil
	} else {
		log.Info("DaemonSet is ready:", "name", daemonSet.Name, "namespace", daemonSet.Namespace)
	}

	return false, r.ensureDeleted(ctx, daemonSet)
}

func convertPodSpecContainersToUninstall(podSpec *corev1.PodSpec) {
	// We're going to use the PodSpec to run a one-time task by using an init container to run the task.
	// See http://blog.itaysk.com/2017/12/26/the-single-use-daemonset-pattern-and-prepulling-images-in-kubernetes
	// for more details.
	podSpec.InitContainers = podSpec.Containers
	podSpec.InitContainers[0].Env = append(podSpec.InitContainers[0].Env, corev1.EnvVar{Name: "UNINSTALL", Value: "true"})

	podSpec.Containers = []corev1.Container{
		{
			Name:  "pause",
			Image: "gcr.io/google_containers/pause",
		},
	}
}

func newDaemonSet(name, namespace string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
