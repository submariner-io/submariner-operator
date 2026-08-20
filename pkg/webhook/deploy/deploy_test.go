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

package deploy_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/submariner-io/admiral/pkg/names"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/webhook/deploy"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	fakeClient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const (
	issuerName      = "test-issuer"
	certificateName = "submariner-operator-serving-cert"
	operatorImage   = "existing-operator-image"
	brokerNamespace = "broker-namespace"
)

var _ = Describe("Webhook", func() {
	var (
		client        controllerClient.Client
		clientBuilder *fakeClient.ClientBuilder
		status        reporter.Interface
	)

	BeforeEach(func() {
		status = reporter.Stdout()
		clientBuilder = fakeClient.NewClientBuilder().
			WithStatusSubresource(&appsv1.Deployment{}, &corev1.Secret{}).
			WithInterceptorFuncs(interceptor.Funcs{
				Get: func(ctx context.Context, client controllerClient.WithWatch, key controllerClient.ObjectKey,
					obj controllerClient.Object, opts ...controllerClient.GetOption,
				) error {
					if err := client.Get(ctx, key, obj, opts...); err != nil {
						return err
					}

					// Simulate deployment controller - populate status fields
					if deployment, ok := obj.(*appsv1.Deployment); ok {
						if deployment.Generation == 0 {
							deployment.Generation = 1
						}

						deployment.Status.ObservedGeneration = deployment.Generation
						deployment.Status.UpdatedReplicas = 1
						deployment.Status.ReadyReplicas = 1
						deployment.Status.AvailableReplicas = 1
					}

					return nil
				},
			})
	})

	JustBeforeEach(func() {
		client = clientBuilder.Build()
	})

	It("should deploy the webhook with the provided operator image", func(ctx context.Context) {
		Expect(deploy.Webhook(ctx, client, "", operatorImage, brokerNamespace, status)).To(Succeed())
		verifyWebhookDeployment(ctx, client, operatorImage)
	})

	When("Submariner is deployed", func() {
		BeforeEach(func() {
			clientBuilder = clientBuilder.WithRuntimeObjects(&appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      names.OperatorComponent,
					Namespace: deploy.OperatorNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Image: operatorImage,
								},
							},
						},
					},
				},
			})
		})

		It("should deploy the webhook with the existing operator image", func(ctx context.Context) {
			Expect(deploy.Webhook(ctx, client, "", "", brokerNamespace, status)).To(Succeed())
			verifyWebhookDeployment(ctx, client, operatorImage)
		})
	})

	When("Submariner is not deployed", func() {
		It("should deploy the webhook with the provided operator image", func(ctx context.Context) {
			Expect(deploy.Webhook(ctx, client, "", operatorImage, brokerNamespace, status)).To(Succeed())
			verifyWebhookDeployment(ctx, client, operatorImage)
		})
	})

	When("an issuerName is provided", func() {
		Context("and an Issuer exists in the namespace", func() {
			BeforeEach(func() {
				clientBuilder = clientBuilder.WithRuntimeObjects(newIssuer(), newCertificateSecret())
			})

			It("should deploy the Certificate with Issuer kind", func(ctx context.Context) {
				Expect(deploy.Webhook(ctx, client, issuerName, operatorImage, brokerNamespace, status)).To(Succeed())
				verifyCertificate(ctx, client, "Issuer")
			})
		})

		Context("a ClusterIssuer exists", func() {
			BeforeEach(func() {
				clientBuilder = clientBuilder.WithRuntimeObjects(newClusterIssuer(), newCertificateSecret())
			})

			It("should deploy the Certificate with ClusterIssuer kind", func(ctx context.Context) {
				Expect(deploy.Webhook(ctx, client, issuerName, operatorImage, brokerNamespace, status)).To(Succeed())
				verifyCertificate(ctx, client, "ClusterIssuer")
			})
		})

		Context("but neither an Issuer nor ClusterIssuer exists", func() {
			It("should return an error", func(ctx context.Context) {
				Expect(deploy.Webhook(ctx, client, issuerName, operatorImage, brokerNamespace, status)).NotTo(Succeed())
			})
		})

		Context("and the Certificate already exists", func() {
			BeforeEach(func() {
				clientBuilder = clientBuilder.WithRuntimeObjects(newIssuer(), newCertificate(), newCertificateSecret())
			})

			It("should update the existing Certificate", func(ctx context.Context) {
				Expect(deploy.Webhook(ctx, client, issuerName, operatorImage, brokerNamespace, status)).To(Succeed())
				verifyCertificate(ctx, client, "Issuer")
			})
		})
	})
})

func newIssuer() *unstructured.Unstructured {
	issuer := &unstructured.Unstructured{}
	issuer.SetGroupVersionKind(deploy.CertManagerGroupVersion.WithKind("Issuer"))
	issuer.SetName(issuerName)
	issuer.SetNamespace(deploy.OperatorNamespace)

	return issuer
}

func newClusterIssuer() *unstructured.Unstructured {
	clusterIssuer := &unstructured.Unstructured{}
	clusterIssuer.SetGroupVersionKind(deploy.CertManagerGroupVersion.WithKind("ClusterIssuer"))
	clusterIssuer.SetName(issuerName)

	return clusterIssuer
}

func newCertificate() runtime.Object {
	cert := &unstructured.Unstructured{}
	cert.SetGroupVersionKind(deploy.CertManagerGroupVersion.WithKind("Certificate"))
	cert.SetName(certificateName)
	cert.SetNamespace(deploy.OperatorNamespace)

	return cert
}

func verifyCertificate(ctx context.Context, client controllerClient.Client, kind string) {
	cert := &unstructured.Unstructured{}
	cert.SetGroupVersionKind(deploy.CertManagerGroupVersion.WithKind("Certificate"))
	err := client.Get(ctx, controllerClient.ObjectKey{
		Namespace: deploy.OperatorNamespace,
		Name:      certificateName,
	}, cert)
	Expect(err).NotTo(HaveOccurred())

	issuerRef, found, err := unstructured.NestedMap(cert.Object, "spec", "issuerRef")
	Expect(err).NotTo(HaveOccurred())
	Expect(found).To(BeTrue())
	Expect(issuerRef["name"]).To(Equal(issuerName))
	Expect(issuerRef["kind"]).To(Equal(kind))
}

func newCertificateSecret() *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "submariner-operator-webhook-cert",
			Namespace: deploy.OperatorNamespace,
		},
		Data: map[string][]byte{
			"tls.crt": []byte("fake-cert"),
			"tls.key": []byte("fake-key"),
		},
	}
}

func verifyWebhookDeployment(ctx context.Context, client controllerClient.Client, image string) {
	service := &corev1.Service{}
	Expect(client.Get(ctx, controllerClient.ObjectKey{Namespace: deploy.OperatorNamespace, Name: "submariner-operator-webhook"},
		service)).NotTo(HaveOccurred())

	deployment := &appsv1.Deployment{}
	Expect(client.Get(ctx, controllerClient.ObjectKey{Namespace: deploy.OperatorNamespace, Name: "submariner-operator-webhook"},
		deployment)).NotTo(HaveOccurred())
	Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal(image))

	webhookConfig := &admissionregv1.ValidatingWebhookConfiguration{}
	Expect(client.Get(ctx, controllerClient.ObjectKey{Name: "submariner-broker-validator"},
		webhookConfig)).NotTo(HaveOccurred())
	Expect(webhookConfig.Webhooks[0].NamespaceSelector.MatchExpressions[0].Values).To(Equal([]string{brokerNamespace}))
}
