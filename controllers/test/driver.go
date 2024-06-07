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

package test

import (
	"context"

	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/syncer/test"
	admtest "github.com/submariner-io/admiral/pkg/test"
	"github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/uninstall"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Driver struct {
	InitScopedClientObjs  []client.Object
	ScopedClient          client.Client
	InitGeneralClientObjs []client.Object
	GeneralClient         client.Client
	InterceptorFuncs      interceptor.Funcs
	Controller            reconcile.Reconciler
	Namespace             string
	ResourceName          string
}

func (d *Driver) BeforeEach() {
	d.ScopedClient = nil
	d.InitScopedClientObjs = []client.Object{}
	d.GeneralClient = nil
	d.InitGeneralClientObjs = []client.Object{}
	d.Controller = nil
}

func (d *Driver) JustBeforeEach() {
	if d.ScopedClient == nil {
		d.ScopedClient = d.NewScopedClient()
	}

	if d.GeneralClient == nil {
		d.GeneralClient = d.NewGeneralClient()
	}
}

func (d *Driver) NewScopedClient() client.Client {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(d.InitScopedClientObjs...).
		WithStatusSubresource(&v1alpha1.Submariner{}).WithInterceptorFuncs(d.InterceptorFuncs).
		WithRESTMapper(test.GetRESTMapperFor(&corev1.Secret{})).Build()
}

func (d *Driver) NewGeneralClient() client.Client {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(d.InitGeneralClientObjs...).
		WithStatusSubresource(&v1alpha1.Submariner{}).WithInterceptorFuncs(d.InterceptorFuncs).Build()
}

func (d *Driver) DoReconcile(ctx context.Context) (reconcile.Result, error) {
	return d.Controller.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: d.Namespace,
		Name:      d.ResourceName,
	}})
}

func (d *Driver) AssertReconcileSuccess(ctx context.Context) {
	r, err := d.DoReconcile(ctx)
	Expect(err).To(Succeed())
	Expect(r.Requeue).To(BeFalse())
	Expect(r.RequeueAfter).To(BeNumerically("==", 0))
}

func (d *Driver) AssertReconcileRequeue(ctx context.Context) {
	r, err := d.DoReconcile(ctx)
	Expect(err).To(Succeed())
	Expect(r.RequeueAfter).To(BeNumerically(">", 0), "Expected requeue after")
	Expect(r.Requeue).To(BeFalse())
}

func (d *Driver) AssertReconcileError(ctx context.Context) {
	_, err := d.DoReconcile(ctx)
	Expect(err).ToNot(Succeed())
}

func (d *Driver) GetDaemonSet(ctx context.Context, name string) (*appsv1.DaemonSet, error) {
	foundDaemonSet := &appsv1.DaemonSet{}
	err := d.ScopedClient.Get(ctx, types.NamespacedName{Name: name, Namespace: d.Namespace}, foundDaemonSet)

	return foundDaemonSet, err
}

func (d *Driver) AssertDaemonSet(ctx context.Context, name string) *appsv1.DaemonSet {
	daemonSet, err := d.GetDaemonSet(ctx, name)
	Expect(err).To(Succeed())

	Expect(daemonSet.ObjectMeta.Labels).To(HaveKeyWithValue("app", name))
	Expect(daemonSet.Spec.Selector.MatchLabels).To(HaveKeyWithValue("app", name))

	for k, v := range daemonSet.Spec.Selector.MatchLabels {
		Expect(daemonSet.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))
	}

	return daemonSet
}

func (d *Driver) AssertNoDaemonSet(ctx context.Context, name string) {
	_, err := d.GetDaemonSet(ctx, name)
	Expect(errors.IsNotFound(err)).To(BeTrue(), "IsNotFound error")
	Expect(err).To(HaveOccurred())
}

func (d *Driver) GetDeployment(ctx context.Context, name string) (*appsv1.Deployment, error) {
	found := &appsv1.Deployment{}
	err := d.ScopedClient.Get(ctx, types.NamespacedName{Name: name, Namespace: d.Namespace}, found)

	return found, err
}

func (d *Driver) AssertDeployment(ctx context.Context, name string) *appsv1.Deployment {
	deployment, err := d.GetDeployment(ctx, name)
	Expect(err).To(Succeed())

	Expect(deployment.ObjectMeta.Labels).To(HaveKeyWithValue("app", name))
	Expect(deployment.Spec.Selector.MatchLabels).To(HaveKeyWithValue("app", name))
	Expect(deployment.Spec.Replicas).ToNot(BeNil())
	Expect(int(*deployment.Spec.Replicas)).To(Equal(1))

	return deployment
}

func (d *Driver) AssertNoDeployment(ctx context.Context, name string) {
	_, err := d.GetDeployment(ctx, name)
	Expect(errors.IsNotFound(err)).To(BeTrue(), "IsNotFound error")
	Expect(err).To(HaveOccurred())
}

func (d *Driver) AssertNoResource(obj client.Object) {
	err := d.ScopedClient.Get(context.Background(), types.NamespacedName{Name: obj.GetName(), Namespace: d.Namespace}, obj)
	Expect(errors.IsNotFound(err)).To(BeTrue(), "IsNotFound error")
	Expect(err).To(HaveOccurred())
}

func (d *Driver) AssertUninstallInitContainer(template *corev1.PodTemplateSpec, image string) map[string]string {
	Expect(template.Spec.InitContainers).To(HaveLen(1))
	Expect(template.Spec.InitContainers[0].Image).To(Equal(image))

	envMap := EnvMapFromVars(template.Spec.InitContainers[0].Env)
	Expect(envMap).To(HaveKeyWithValue(uninstall.ContainerEnvVar, "true"))

	return envMap
}

func (d *Driver) UpdateDaemonSetToReady(ctx context.Context, daemonSet *appsv1.DaemonSet) {
	d.UpdateDaemonSetToScheduled(ctx, daemonSet)

	daemonSet.Status.NumberReady = daemonSet.Status.DesiredNumberScheduled
	Expect(d.ScopedClient.Status().Update(ctx, daemonSet)).To(Succeed())
}

func (d *Driver) UpdateDaemonSetToObserved(ctx context.Context, daemonSet *appsv1.DaemonSet) {
	daemonSet.Generation = 1
	Expect(d.ScopedClient.Update(ctx, daemonSet)).To(Succeed())

	daemonSet.Status.ObservedGeneration = daemonSet.Generation
	Expect(d.ScopedClient.Status().Update(ctx, daemonSet)).To(Succeed())
}

func (d *Driver) UpdateDaemonSetToScheduled(ctx context.Context, daemonSet *appsv1.DaemonSet) {
	daemonSet.Generation = 1
	Expect(d.ScopedClient.Update(ctx, daemonSet)).To(Succeed())

	daemonSet.Status.ObservedGeneration = daemonSet.Generation
	daemonSet.Status.DesiredNumberScheduled = 1
	Expect(d.ScopedClient.Status().Update(ctx, daemonSet)).To(Succeed())
}

func (d *Driver) UpdateDeploymentToReady(ctx context.Context, deployment *appsv1.Deployment) {
	deployment.Status.ReadyReplicas = *deployment.Spec.Replicas
	deployment.Status.AvailableReplicas = *deployment.Spec.Replicas
	Expect(d.ScopedClient.Status().Update(ctx, deployment)).To(Succeed())
}

func (d *Driver) NewDaemonSet(name string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: d.Namespace,
			Name:      name,
		},
	}
}

func (d *Driver) NewDeployment(name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: d.Namespace,
			Name:      name,
		},
	}
}

func (d *Driver) NewPodWithLabel(label, value string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: d.Namespace,
			Name:      string(uuid.NewUUID()),
			Labels: map[string]string{
				label: value,
			},
		},
	}
}

func (d *Driver) DeletePods(ctx context.Context, label, value string) {
	err := d.ScopedClient.DeleteAllOf(ctx, &corev1.Pod{}, client.InNamespace(d.Namespace),
		client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(map[string]string{label: value})})
	Expect(err).To(Succeed())
}

func (d *Driver) AwaitFinalizer(obj client.Object, finalizer string) {
	admtest.AwaitFinalizer[client.Object](resource.ForControllerClient[client.Object](d.ScopedClient, d.Namespace, obj), obj.GetName(),
		finalizer)
}

func (d *Driver) AwaitNoResource(obj client.Object) {
	admtest.AwaitNoResource[client.Object](resource.ForControllerClient[client.Object](d.ScopedClient, d.Namespace, obj), obj.GetName())
}

func EnvMapFrom(daemonSet *appsv1.DaemonSet) map[string]string {
	return EnvMapFromVars(daemonSet.Spec.Template.Spec.Containers[0].Env)
}

func EnvMapFromVars(env []corev1.EnvVar) map[string]string {
	envMap := map[string]string{}
	for _, envVar := range env {
		envMap[envVar.Name] = envVar.Value
	}

	return envMap
}
