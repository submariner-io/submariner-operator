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

package serviceaccount

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/resource"
	"github.com/submariner-io/submariner-operator/pkg/embeddedyamls"
	resourceutil "github.com/submariner-io/submariner-operator/pkg/resource"
	"github.com/submariner-io/submariner-operator/pkg/secret"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

const (
	createdByAnnotation = "kubernetes.io/created-by"
	creatorName         = "subctl"
)

// ensureFromYAML creates the given service account.
// nolint:wrapcheck // No need to wrap errors here.
func ensureFromYAML(kubeClient kubernetes.Interface, namespace, yaml string) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}

	err := embeddedyamls.GetObject(yaml, sa)
	if err != nil {
		return nil, err
	}

	err = ensure(kubeClient, namespace, sa, true)
	if err != nil {
		return nil, err
	}

	return sa, err
}

// nolint:wrapcheck // No need to wrap errors here.
func ensure(kubeClient kubernetes.Interface, namespace string, sa *corev1.ServiceAccount, onlyCreate bool) error {
	if onlyCreate {
		_, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), sa.Name, metav1.GetOptions{})

		if err == nil || !apierrors.IsNotFound(err) {
			return err
		}
	}

	_, err := resourceutil.CreateOrUpdate(context.TODO(), resource.ForServiceAccount(kubeClient, namespace), sa)

	return err
}

// nolint:wrapcheck // No need to wrap errors here.
func Ensure(kubeClient kubernetes.Interface, namespace string, sa *corev1.ServiceAccount, onlyCreate bool) (*corev1.ServiceAccount, error) {
	err := ensure(kubeClient, namespace, sa, onlyCreate)
	if err != nil {
		return nil, err
	}

	_, err = EnsureSecretFromSA(kubeClient, sa.Name, namespace)

	if err != nil {
		return nil, errors.Wrap(err, "failed to get secret from broker SA")
	}

	return kubeClient.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), sa.Name, metav1.GetOptions{})
}

// EnsureFromYAML creates the given service account and secret for it.
func EnsureFromYAML(kubeClient kubernetes.Interface, namespace, yaml string) (bool, error) {
	sa, err := ensureFromYAML(kubeClient, namespace, yaml)
	if err != nil {
		return false, errors.Wrap(err, "error provisioning the ServiceAccount resource")
	}

	saSecret, err := EnsureSecretFromSA(kubeClient, sa.Name, namespace)
	if err != nil {
		return false, errors.Wrap(err, "error creating secret for ServiceAccount resource")
	}

	return sa != nil && saSecret != nil, nil
}

func EnsureSecretFromSA(client kubernetes.Interface, saName, namespace string) (*corev1.Secret, error) {
	sa, err := client.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), saName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get ServiceAccount %s/%s", namespace, saName)
	}

	saSecret := getSecretFromSA(client, sa)

	if saSecret != nil {
		return saSecret, nil
	}

	// We couldn't find right secret from this SA, search all Secrets
	saSecret, err = getSecretForSA(client, sa)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil {
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-token-%s", sa.Name, generateRandomString(5)),
				Namespace: namespace,
				Annotations: map[string]string{
					corev1.ServiceAccountNameKey: saName,
					createdByAnnotation:          creatorName,
				},
			},
			Type: corev1.SecretTypeServiceAccountToken,
		}

		saSecret, err = secret.Ensure(client, newSecret.Namespace, newSecret)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create secret for ServiceAccount %v", saName)
		}
	}

	secretRef := corev1.ObjectReference{
		Name: saSecret.Name,
	}

	sa.Secrets = append(sa.Secrets, secretRef)
	err = ensure(client, namespace, sa, false)

	if err != nil {
		return nil, errors.Wrapf(err, "failed to update ServiceAccount %v with Secret reference %v", saName, secretRef.Name)
	}

	return saSecret, nil
}

func getSecretFromSA(client kubernetes.Interface, sa *corev1.ServiceAccount) *corev1.Secret {
	secretNamePrefix := fmt.Sprintf("%s-token-", sa.Name)
	for _, saSecretRef := range sa.Secrets {
		if strings.HasPrefix(saSecretRef.Name, secretNamePrefix) {
			saSecret, _ := client.CoreV1().Secrets(sa.Namespace).Get(context.TODO(), saSecretRef.Name, metav1.GetOptions{})
			if saSecret.Annotations[corev1.ServiceAccountNameKey] == sa.Name && saSecret.Type == corev1.SecretTypeServiceAccountToken {
				return saSecret
			}
		}
	}

	return nil
}

func getSecretForSA(client kubernetes.Interface, sa *corev1.ServiceAccount) (*corev1.Secret, error) {
	saSecrets, err := client.CoreV1().Secrets(sa.Namespace).List(context.TODO(), metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("type", "kubernetes.io/service-account-token").String(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get secrets of type service-account-token in %v", sa.Namespace)
	}

	for i := 0; i < len(saSecrets.Items); i++ {
		if saSecrets.Items[i].Annotations[corev1.ServiceAccountNameKey] == sa.Name {
			return &saSecrets.Items[i], nil
		}
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{
		Group:    corev1.SchemeGroupVersion.Group,
		Resource: "secrets",
	}, sa.Name)
}

// nolint:gosec // we need a pseudo random string for name.
func generateRandomString(length int) string {
	rand.Seed(time.Now().UnixNano())

	s := make([]byte, length)
	rand.Read(s)

	return fmt.Sprintf("%x", s)[:length]
}
