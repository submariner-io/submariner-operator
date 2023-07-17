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

package apply_test

import (
	"context"
	"net/http"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/fake"
	"github.com/submariner-io/submariner-operator/controllers/apply"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

var _ = Describe("Apply", func() {
	Context("DaemonSet", testDaemonSet)
	Context("Deployment", testDeployment)
	Context("ConfigMap", testConfigMap)
	Context("Service", testService)
})

func testDaemonSet() {
	t := newTestDriver()

	var daemonSet *appsv1.DaemonSet

	BeforeEach(func() {
		daemonSet = &appsv1.DaemonSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ds",
				Namespace: submarinerNamespace,
			},
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test-container",
								Image: "test-image",
							},
						},
					},
				},
				MinReadySeconds: 10,
			},
		}
	})

	When("the DaemonSet doesn't exist", func() {
		It("should create it", func() {
			actual, err := apply.DaemonSet(context.Background(), t.owner, daemonSet, log, t.client, scheme.Scheme)
			Expect(err).To(Succeed())
			Expect(actual).To(Equal(daemonSet))
			t.verifyOwnerRef(actual)

			actual = &appsv1.DaemonSet{}
			Expect(t.client.Get(context.Background(), types.NamespacedName{
				Namespace: daemonSet.Namespace, Name: daemonSet.Name,
			}, actual)).To(Succeed())
			Expect(actual).To(Equal(daemonSet))
		})
	})

	When("the DaemonSet already exists", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, daemonSet.DeepCopy())
			daemonSet.Spec.MinReadySeconds = 20
			daemonSet.Labels = map[string]string{"foo": "bar"}
		})

		It("should update it", func() {
			actual, err := apply.DaemonSet(context.Background(), t.owner, daemonSet, log, t.client, scheme.Scheme)
			Expect(err).To(Succeed())
			Expect(actual).To(Equal(daemonSet))
		})

		Context("and it's immutable", func() {
			JustBeforeEach(func() {
				t.client = fake.NewReactingClient(t.client).AddReactor(fake.Update, &appsv1.DaemonSet{},
					fake.FailingReaction(&apierrors.StatusError{ErrStatus: metav1.Status{
						Status:  metav1.StatusFailure,
						Code:    http.StatusUnprocessableEntity,
						Reason:  metav1.StatusReasonInvalid,
						Message: "Object is immutable",
					}}))
			})

			It("should re-create it", func() {
				actual, err := apply.DaemonSet(context.Background(), t.owner, daemonSet, log, t.client, scheme.Scheme)
				Expect(err).To(Succeed())
				Expect(actual).To(Equal(daemonSet))
			})
		})
	})
}

func testDeployment() {
	t := newTestDriver()

	var deployment *appsv1.Deployment

	BeforeEach(func() {
		deployment = &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dep",
				Namespace: submarinerNamespace,
			},
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:  "test-container",
								Image: "test-image",
							},
						},
					},
				},
				MinReadySeconds: 10,
			},
		}
	})

	When("the Deployment doesn't exist", func() {
		It("should create it", func() {
			actual, err := apply.Deployment(context.Background(), t.owner, deployment, log, t.client, scheme.Scheme)
			Expect(err).To(Succeed())
			Expect(actual).To(Equal(deployment))
			t.verifyOwnerRef(actual)

			actual = &appsv1.Deployment{}
			Expect(t.client.Get(context.Background(), types.NamespacedName{
				Namespace: deployment.Namespace, Name: deployment.Name,
			}, actual)).To(Succeed())
			Expect(actual).To(Equal(deployment))
		})
	})

	When("the Deployment already exists", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, deployment.DeepCopy())
			deployment.Spec.MinReadySeconds = 20
		})

		It("should update it", func() {
			actual, err := apply.Deployment(context.Background(), t.owner, deployment, log, t.client, scheme.Scheme)
			Expect(err).To(Succeed())
			Expect(actual).To(Equal(deployment))
		})
	})
}

func testConfigMap() {
	t := newTestDriver()

	var configMap *corev1.ConfigMap

	BeforeEach(func() {
		configMap = &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-cm",
				Namespace: submarinerNamespace,
			},
			Data: map[string]string{"key1": "value1"},
		}
	})

	When("the ConfigMap doesn't exist", func() {
		It("should create it", func() {
			actual, err := apply.ConfigMap(context.Background(), t.owner, configMap, log, t.client, scheme.Scheme)
			Expect(err).To(Succeed())
			Expect(actual).To(Equal(configMap))
			t.verifyOwnerRef(actual)

			actual = &corev1.ConfigMap{}
			Expect(t.client.Get(context.Background(), types.NamespacedName{
				Namespace: configMap.Namespace, Name: configMap.Name,
			}, actual)).To(Succeed())
			Expect(actual).To(Equal(configMap))
		})
	})

	When("the ConfigMap already exists", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, configMap.DeepCopy())
			configMap.Data = map[string]string{"key2": "value2"}
		})

		It("should update it", func() {
			actual, err := apply.ConfigMap(context.Background(), t.owner, configMap, log, t.client, scheme.Scheme)
			Expect(err).To(Succeed())
			Expect(actual).To(Equal(configMap))
		})
	})
}

func testService() {
	t := newTestDriver()

	var service *corev1.Service

	BeforeEach(func() {
		service = &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-svc",
				Namespace: submarinerNamespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: "1.2.3.4",
				Type:      corev1.ServiceTypeClusterIP,
			},
		}
	})

	When("the Service doesn't exist", func() {
		It("should create it", func() {
			actual, err := apply.Service(context.Background(), t.owner, service, log, t.client, scheme.Scheme)
			Expect(err).To(Succeed())
			Expect(actual).To(Equal(service))
			t.verifyOwnerRef(actual)

			actual = &corev1.Service{}
			Expect(t.client.Get(context.Background(), types.NamespacedName{Namespace: service.Namespace, Name: service.Name}, actual)).To(Succeed())
			Expect(actual).To(Equal(service))
		})
	})

	When("the Service already exists", func() {
		BeforeEach(func() {
			t.initClientObjs = append(t.initClientObjs, service.DeepCopy())
			service.Labels = map[string]string{"foo": "bar"}
			service.Annotations = map[string]string{"foo1": "bar1"}
		})

		It("should update it", func() {
			actual, err := apply.Service(context.Background(), t.owner, service, log, t.client, scheme.Scheme)
			Expect(err).To(Succeed())
			Expect(actual).To(Equal(service))
		})
	})
}
