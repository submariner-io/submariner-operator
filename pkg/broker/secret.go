package broker

import (
	"crypto/rand"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GenerateRandomPSK returns securely generated n-byte array.
func GenerateRandomPSK(n int) ([]byte, error) {
	psk := make([]byte, n)
	_, err := rand.Read(psk)
	return psk, err
}

func NewBrokerPSKSecret(psk []byte) *v1.Secret {
	psk_secret_data := make(map[string][]byte)
	psk_secret_data["psk"] = psk

	psk_secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "submariner-k8s-broker-psk",
			// FIXME: Get namespace somehow
			Namespace: "submariner-k8s-broker",
		},
		Data: psk_secret_data,
	}

	return psk_secret
}
