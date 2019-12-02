/*
© 2019 Red Hat, Inc. and others.

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

package broker

import (
	"crypto/rand"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

const ipsecPSKSecretName = "submariner-ipsec-psk"

// generateRandomPSK returns securely generated n-byte array.
func generateRandomPSK(n int) ([]byte, error) {
	psk := make([]byte, n)
	_, err := rand.Read(psk)
	return psk, err
}

func NewBrokerPSKSecret(bytes int) (*v1.Secret, error) {

	psk, err := generateRandomPSK(bytes)
	if err != nil {
		return nil, err
	}

	pskSecretData := make(map[string][]byte)
	pskSecretData["psk"] = psk

	psk_secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: ipsecPSKSecretName,
		},
		Data: pskSecretData,
	}

	return psk_secret, nil
}

func GetIPSECPSKSecret(clientSet clientset.Interface, brokerNamespace string) (*v1.Secret, error) {
	return clientSet.CoreV1().Secrets(brokerNamespace).Get(ipsecPSKSecretName, metav1.GetOptions{})
}
