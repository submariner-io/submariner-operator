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
	admtest "github.com/submariner-io/admiral/pkg/test"
	"github.com/submariner-io/submariner-operator/controllers/resource"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Driver struct {
	InitClientObjs []client.Object
	Client         client.Client
	Controller     reconcile.Reconciler
	Namespace      string
	ResourceName   string
}

func (d *Driver) BeforeEach() {
	d.Client = nil
	d.InitClientObjs = []client.Object{}
	d.Controller = nil
}

func (d *Driver) JustBeforeEach() {
	if d.Client == nil {
		d.Client = d.NewClient()
	}
}

func (d *Driver) NewClient() client.Client {
	return fake.NewClientBuilder().WithScheme(scheme.Scheme).WithObjects(d.InitClientObjs...).Build()
}

func (d *Driver) DoReconcile() (reconcile.Result, error) {
	return d.Controller.Reconcile(context.TODO(), reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: d.Namespace,
		Name:      d.ResourceName,
	}})
}

func (d *Driver) AssertReconcileSuccess() {
	r, err := d.DoReconcile()
	Expect(err).To(Succeed())
	Expect(r.Requeue).To(BeFalse())
	Expect(r.RequeueAfter).To(BeNumerically("==", 0))
}

func (d *Driver) AssertReconcileRequeue() {
	r, err := d.DoReconcile()
	Expect(err).To(Succeed())
	Expect(r.RequeueAfter).To(BeNumerically(">", 0), "Expected requeue after")
	Expect(r.Requeue).To(BeFalse())
}

func (d *Driver) AssertReconcileError() {
	_, err := d.DoReconcile()
	Expect(err).ToNot(Succeed())
}

func (d *Driver) GetDaemonSet(name string) (*appsv1.DaemonSet, error) {
	foundDaemonSet := &appsv1.DaemonSet{}
	err := d.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: d.Namespace}, foundDaemonSet)

	return foundDaemonSet, err
}

func (d *Driver) AssertDaemonSet(name string) *appsv1.DaemonSet {
	daemonSet, err := d.GetDaemonSet(name)
	Expect(err).To(Succeed())

	Expect(daemonSet.ObjectMeta.Labels).To(HaveKeyWithValue("app", name))
	Expect(daemonSet.Spec.Selector.MatchLabels).To(HaveKeyWithValue("app", name))

	for k, v := range daemonSet.Spec.Selector.MatchLabels {
		Expect(daemonSet.Spec.Template.ObjectMeta.Labels).To(HaveKeyWithValue(k, v))
	}

	return daemonSet
}

func (d *Driver) AssertNoDaemonSet(name string) {
	_, err := d.GetDaemonSet(name)
	Expect(errors.IsNotFound(err)).To(BeTrue(), "IsNotFound error")
	Expect(err).To(HaveOccurred())
}

func (d *Driver) GetDeployment(name string) (*appsv1.Deployment, error) {
	found := &appsv1.Deployment{}
	err := d.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: d.Namespace}, found)

	return found, err
}

func (d *Driver) AssertDeployment(name string) *appsv1.Deployment {
	deployment, err := d.GetDeployment(name)
	Expect(err).To(Succeed())

	Expect(deployment.ObjectMeta.Labels).To(HaveKeyWithValue("app", name))
	Expect(deployment.Spec.Selector.MatchLabels).To(HaveKeyWithValue("app", name))
	Expect(deployment.Spec.Replicas).ToNot(BeNil())
	Expect(int(*deployment.Spec.Replicas)).To(Equal(1))

	return deployment
}

func (d *Driver) AssertNoDeployment(name string) {
	_, err := d.GetDeployment(name)
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

func (d *Driver) UpdateDaemonSetToReady(daemonSet *appsv1.DaemonSet) {
	d.UpdateDaemonSetToScheduled(daemonSet)
	daemonSet.Status.NumberReady = daemonSet.Status.DesiredNumberScheduled
	Expect(d.Client.Update(context.TODO(), daemonSet)).To(Succeed())
}

func (d *Driver) UpdateDaemonSetToObserved(daemonSet *appsv1.DaemonSet) {
	daemonSet.Generation = 1
	daemonSet.Status.ObservedGeneration = daemonSet.Generation
	Expect(d.Client.Update(context.TODO(), daemonSet)).To(Succeed())
}

func (d *Driver) UpdateDaemonSetToScheduled(daemonSet *appsv1.DaemonSet) {
	daemonSet.Generation = 1
	daemonSet.Status.ObservedGeneration = daemonSet.Generation
	daemonSet.Status.DesiredNumberScheduled = 1
	Expect(d.Client.Update(context.TODO(), daemonSet)).To(Succeed())
}

func (d *Driver) UpdateDeploymentToReady(deployment *appsv1.Deployment) {
	deployment.Status.ReadyReplicas = *deployment.Spec.Replicas
	deployment.Status.AvailableReplicas = *deployment.Spec.Replicas
	Expect(d.Client.Update(context.TODO(), deployment)).To(Succeed())
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

func (d *Driver) DeletePods(label, value string) {
	err := d.Client.DeleteAllOf(context.TODO(), &corev1.Pod{}, client.InNamespace(d.Namespace),
		client.MatchingLabelsSelector{Selector: labels.SelectorFromSet(map[string]string{label: value})})
	Expect(err).To(Succeed())
}

func (d *Driver) AwaitFinalizer(obj client.Object, finalizer string) {
	admtest.AwaitFinalizer(resource.ForControllerClient(d.Client, d.Namespace, obj), obj.GetName(), finalizer)
}

func (d *Driver) AwaitNoResource(obj client.Object) {
	admtest.AwaitNoResource(resource.ForControllerClient(d.Client, d.Namespace, obj), obj.GetName())
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
