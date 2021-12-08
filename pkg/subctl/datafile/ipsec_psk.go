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

package datafile

import (
	"context"
	"crypto/rand"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

const (
	ipsecPSKSecretName = "submariner-ipsec-psk"
	ipsecSecretLength  = 48
)

// generateRandomPSK returns securely generated n-byte array.
func generateRandomPSK(n int) ([]byte, error) {
	psk := make([]byte, n)
	_, err := rand.Read(psk)
	return psk, err
}

func newIPSECPSKSecret() (*v1.Secret, error) {
	psk, err := generateRandomPSK(ipsecSecretLength)
	if err != nil {
		return nil, err
	}

	pskSecretData := make(map[string][]byte)
	pskSecretData["psk"] = psk

	pskSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: ipsecPSKSecretName,
		},
		Data: pskSecretData,
	}

	return pskSecret, nil
}

func GetIPSECPSKSecret(clientSet clientset.Interface, namespace string) (*v1.Secret, error) {
	return clientSet.CoreV1().Secrets(namespace).Get(context.TODO(), ipsecPSKSecretName, metav1.GetOptions{})
}
