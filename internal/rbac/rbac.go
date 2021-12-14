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

package rbac

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// maxGeneratedNameLength is the maximum generated length for a token, excluding the random suffix
// See k8s.io/apiserver/pkg/storage/names.
const maxGeneratedNameLength = 63 - 5

func GetClientTokenSecret(kubeClient kubernetes.Interface, namespace, serviceAccountName string) (*v1.Secret, error) {
	sa, err := kubeClient.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), serviceAccountName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "ServiceAccount %s get failed", serviceAccountName)
	}

	if len(sa.Secrets) < 1 {
		return nil, fmt.Errorf("ServiceAccount %s does not have any secret", sa.Name)
	}

	tokenPrefix := fmt.Sprintf("%s-token-", serviceAccountName)
	if len(tokenPrefix) > maxGeneratedNameLength {
		tokenPrefix = tokenPrefix[:maxGeneratedNameLength]
	}

	for _, secret := range sa.Secrets {
		if strings.HasPrefix(secret.Name, tokenPrefix) {
			// nolint:wrapcheck // No need to wrap here
			return kubeClient.CoreV1().Secrets(namespace).Get(context.TODO(), secret.Name, metav1.GetOptions{})
		}
	}

	return nil, fmt.Errorf("ServiceAccount %s does not have a secret of type token", serviceAccountName)
}
