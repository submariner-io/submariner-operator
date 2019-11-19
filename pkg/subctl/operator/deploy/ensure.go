package deploy

import (
	"fmt"

	submariner "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/engine"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Ensure(config *rest.Config, submarinerSpec submariner.SubmarinerSpec) error {

	err := engine.Ensure(config)
	if err != nil {
		return fmt.Errorf("error setting up the engine requirements: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("error creating the core kubernetes clientset: %s", err)
	}

	// Create the namespace
	_, err = clientset.CoreV1().Namespaces().Create(NewSubmarinerNamespace())
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("error creating the Submariner namespace %s", err)
	}

	submariner := &submariner.Submariner{
		ObjectMeta: metav1.ObjectMeta{
			Name: "submariner",
		},
		Spec: submarinerSpec,
	}

	submarinerClient, err := submarinerclientset.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	_, err = submarinerClient.SubmarinerV1alpha1().Submariners("submariner-operator").Update(submariner)
	if err != nil {
		if errors.IsNotFound(err) {
			_, err = submarinerClient.SubmarinerV1alpha1().Submariners("submariner-operator").Create(submariner)
		}
		if err != nil {
			panic(err.Error())
		}
	}

	// TODO follow ensure pattern:
	// if created, err := crs.Ensure(...); err != nil {
	//	return err
	// } else if created {
	//	fmt.Printf("* Created Submariner CR.\n")
	// }

	fmt.Printf("* Submariner is up and running\n")

	return nil
}
