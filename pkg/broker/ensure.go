package broker

import (
	"fmt"

	apiextension "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Ensure(config *rest.Config, ipsecPSKBytes int) error {
	// Create the CRDs we need
	apiext, err := apiextension.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the api extensions client: %s", err)
	}
	_, err = apiext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(NewClustersCRD())
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the Cluster CRD: %s", err)
	}
	_, err = apiext.ApiextensionsV1beta1().CustomResourceDefinitions().Create(NewEndpointsCRD())
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the Endpoint CRD: %s", err)
	}

	// Create a clientset for the other standard kubernetes resources
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the core kubernetes clientset: %s", err)
	}

	// Create the namespace
	_, err = clientset.CoreV1().Namespaces().Create(NewBrokerNamespace())
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker namespace %s", err)
	}

	// Create the SA we need for the broker
	_, err = clientset.CoreV1().ServiceAccounts(SubmarinerBrokerNamespace).Create(NewBrokerSA())
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the default broker service account: %s", err)
	}

	// Create the role
	_, err = clientset.RbacV1().Roles(SubmarinerBrokerNamespace).Create(NewBrokerRole())
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating broker role: %s", err)
	}

	// Create the role binding
	_, err = clientset.RbacV1().RoleBindings(SubmarinerBrokerNamespace).Create(NewBrokerRoleBinding())
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the broker rolebinding: %s", err)
	}

	// Generate and store a psk in secret
	pskSecret, err := NewBrokerPSKSecret(ipsecPSKBytes)
	if err != nil {
		return fmt.Errorf("error generating the IPSEC PSK secret: %s", err)
	}

	_, err = clientset.CoreV1().Secrets(SubmarinerBrokerNamespace).Create(pskSecret)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the IPSEC PSK secret: %s", err)
	}
	return nil
}
