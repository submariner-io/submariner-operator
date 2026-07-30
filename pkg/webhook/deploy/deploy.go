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

package deploy

import (
	"context"

	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/admiral/pkg/util"
	webhookyaml "github.com/submariner-io/submariner-operator/config/webhook"
	admissionregv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

const OperatorNamespace = "submariner-operator"

var CertManagerGroupVersion = schema.GroupVersion{Group: "cert-manager.io", Version: "v1"}

func Webhook(ctx context.Context, client controllerClient.Client, issuerName, operatorImage, brokerNamespace string,
	status reporter.Interface,
) error {
	if operatorImage == "" {
		var err error

		operatorImage, err = getOperatorImage(ctx, client, status)
		if err != nil {
			return err
		}
	}

	if issuerName != "" {
		if err := deployCertificate(ctx, client, issuerName, status); err != nil {
			return err
		}
	}

	if err := deployService(ctx, client, status); err != nil {
		return err
	}

	if err := deployWebhookConfig(ctx, client, brokerNamespace, status); err != nil {
		return err
	}

	return deployOperator(ctx, client, operatorImage, status)
}

func getOperatorImage(ctx context.Context, client controllerClient.Client, status reporter.Interface) (string, error) {
	status.Start("Retrieving Submariner operator")
	defer status.End()

	// Get the existing operator deployment to extract its image
	existingDeployment := &appsv1.Deployment{}

	err := client.Get(ctx, controllerClient.ObjectKey{Namespace: OperatorNamespace, Name: OperatorNamespace}, existingDeployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", status.Error(err, "no existing Submariner operator deployment was found in namespace %q", OperatorNamespace)
		}

		return "", status.Error(err, "error retrieving submariner-operator deployment")
	}

	return existingDeployment.Spec.Template.Spec.Containers[0].Image, nil
}

func deployCertificate(ctx context.Context, client controllerClient.Client, issuerName string, status reporter.Interface) error {
	kind, err := getIssuerKind(ctx, client, issuerName, status)
	if err != nil {
		return err
	}

	status.Start("Deploying the Certificate")
	defer status.End()

	certificate := &unstructured.Unstructured{}
	if err := yaml.Unmarshal(webhookyaml.Certificate, certificate); err != nil {
		return status.Error(err, "error parsing Certificate YAML")
	}

	if err := unstructured.SetNestedField(certificate.Object, issuerName, "spec", "issuerRef", "name"); err != nil {
		return status.Error(err, "error setting issuerRef name")
	}

	if err := unstructured.SetNestedField(certificate.Object, kind, "spec", "issuerRef", "kind"); err != nil {
		return status.Error(err, "error setting issuerRef kind")
	}

	err = Ensure(ctx, client, certificate)

	return status.Error(err, "error deploying Certificate")
}

func getIssuerKind(ctx context.Context, client controllerClient.Client, issuerName string,
	status reporter.Interface,
) (string, error) {
	status.Start("Retrieving Issuer %q", issuerName)
	defer status.End()

	// Try to get Issuer (namespace-scoped) first
	issuer := &unstructured.Unstructured{}
	issuer.SetGroupVersionKind(CertManagerGroupVersion.WithKind("Issuer"))

	err := client.Get(ctx, controllerClient.ObjectKey{Namespace: OperatorNamespace, Name: issuerName}, issuer)
	if err == nil {
		return issuer.GetKind(), nil
	}

	if !apierrors.IsNotFound(err) {
		return "", status.Error(err, "error checking for Issuer %q in namespace %q", issuerName, OperatorNamespace)
	}

	// Issuer not found, try ClusterIssuer (cluster-scoped)
	clusterIssuer := &unstructured.Unstructured{}
	clusterIssuer.SetGroupVersionKind(CertManagerGroupVersion.WithKind("ClusterIssuer"))

	err = client.Get(ctx, controllerClient.ObjectKey{Name: issuerName}, clusterIssuer)
	if err == nil {
		return clusterIssuer.GetKind(), nil
	}

	if apierrors.IsNotFound(err) {
		return "", status.Error(err, "No Issuer or ClusterIssuer %q found", issuerName)
	}

	return "", status.Error(err, "error checking for ClusterIssuer %q", issuerName)
}

func deployService(ctx context.Context, client controllerClient.Client, status reporter.Interface) error {
	status.Start("Deploying the Service")
	defer status.End()

	service := &corev1.Service{}
	if err := yaml.Unmarshal(webhookyaml.Service, service); err != nil {
		return status.Error(err, "error parsing Service YAML")
	}

	err := Ensure(ctx, client, service)

	return status.Error(err, "error deploying Service")
}

func deployOperator(ctx context.Context, client controllerClient.Client, operatorImage string, status reporter.Interface) error {
	status.Start("Deploying the Operator")
	defer status.End()

	deployment := &appsv1.Deployment{}
	if err := yaml.Unmarshal(webhookyaml.Deployment, deployment); err != nil {
		return status.Error(err, "error parsing Deployment YAML")
	}

	deployment.Spec.Template.Spec.Containers[0].Image = operatorImage

	err := Ensure(ctx, client, deployment)

	return status.Error(err, "error deploying Deployment")
}

func deployWebhookConfig(ctx context.Context, client controllerClient.Client, brokerNamespace string,
	status reporter.Interface,
) error {
	status.Start("Deploying the ValidatingWebhookConfiguration")
	defer status.End()

	webhookConfig := &admissionregv1.ValidatingWebhookConfiguration{}
	if err := yaml.Unmarshal(webhookyaml.ValidatingWebhookConfig, webhookConfig); err != nil {
		return status.Error(err, "error parsing ValidatingWebhookConfig YAML")
	}

	webhookConfig.Webhooks[0].NamespaceSelector.MatchExpressions[0].Values = []string{brokerNamespace}

	err := Ensure(ctx, client, webhookConfig)

	return status.Error(err, "error deploying ValidatingWebhookConfiguration")
}

func Ensure[T controllerClient.Object](ctx context.Context, client controllerClient.Client, obj T) error {
	_, err := util.CreateOrUpdate(ctx, resource.ForControllerClient(client, OperatorNamespace, obj), obj, util.Replace(obj))
	return err //nolint:wrapcheck // No need to wrap.
}
