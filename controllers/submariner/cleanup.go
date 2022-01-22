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

func (r *Reconciler) runComponentCleanup(ctx context.Context, instance *operatorv1alpha1.Submariner) (reconcile.Result, error) {
	// First, delete the regular DaemonSets/Deployments and ensure all their pods are cleaned up.
	objsToDelete := []client.Object{newDaemonSet(names.GatewayComponent, instance.Namespace)}
	for _, obj := range objsToDelete {
		err := r.ensureDeleted(ctx, obj)
		if err != nil {
			return reconcile.Result{}, err
		}

		pods, err := findPodsBySelector(ctx, r.config.Client, obj.GetNamespace(), &metav1.LabelSelector{
			MatchLabels: map[string]string{appLabel: obj.GetName()},
		})
		if err != nil {
			return reconcile.Result{}, err
		}

		numPods := len(pods)
		if numPods > 0 {
			log.Info(fmt.Sprintf("%T still has pods - requeueing: ", obj), "name", obj.GetName(), "namespace", obj.GetNamespace(),
				"numPods", numPods)
			return requeueAfter, nil
		}
	}

	// This has the side effect of setting the CIDRs in the Submariner instance.
	_, err := r.discoverNetwork(instance)
	if err != nil {
		return requeueAfter, err
	}

	uninstallDaemonSets := []*appsv1.DaemonSet{
		newGatewayDaemonSet(instance, names.AppendUninstall(names.GatewayComponent)),
	}

	// Next, create the corresponding uninstall DaemonSets/Deployments and ensure each completes (ie reports ready).
	for _, daemonSet := range uninstallDaemonSets {
		requeue, err := r.ensureUninstallDaemonSetReady(ctx, daemonSet)
		if err != nil {
			return reconcile.Result{}, err
		}

		if requeue {
			// TODO - check the elapased time wrt the Submariner DeletionTimeStamp and abort cleanup if too long
			return requeueAfter, nil
		}
	}

	// TODO - ensure all uninstall DS's are deleted

	err = finalizer.Remove(ctx, resource.ForControllerClient(r.config.Client, instance.Namespace, &operatorv1alpha1.Submariner{}),
		instance, SubmarinerFinalizer)

	return reconcile.Result{}, err // nolint:wrapcheck // No need to wrap
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

func (r *Reconciler) ensureUninstallDaemonSetReady(ctx context.Context, daemonSet *appsv1.DaemonSet) (bool, error) {
	convertPodSpecContainersToUninstall(&daemonSet.Spec.Template.Spec)

	err := r.config.Client.Create(ctx, daemonSet)
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return false, errors.Wrapf(err, "error creating %#v", daemonSet)
		}
	} else {
		log.Info("Created DaemonSet:", "name", daemonSet.Name, "namespace", daemonSet.Namespace)
	}

	err = r.config.Client.Get(ctx, client.ObjectKeyFromObject(daemonSet), daemonSet)
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
	// We're going to use the PodSpec run a one-time task by using an init container to run the task.
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
