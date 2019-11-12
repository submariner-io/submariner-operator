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
