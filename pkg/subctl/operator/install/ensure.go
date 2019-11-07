package install

import (
	"fmt"

	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/crds"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/deployment"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/namespace"
	"github.com/submariner-io/submariner-operator/pkg/subctl/operator/install/serviceaccount"
	"k8s.io/client-go/rest"
)

func Ensure(config *rest.Config, operatorNamespace string, operatorImage string) error {

	if created, err := crds.Ensure(config); err != nil {
		return err
	} else if created {
		fmt.Printf("* Created operator CRDs.\n")
	}

	if created, err := namespace.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		fmt.Printf("* Created operator namespace: %s\n", operatorNamespace)
	}

	if created, err := serviceaccount.Ensure(config, operatorNamespace); err != nil {
		return err
	} else if created {
		fmt.Printf("* Created operator service account and role\n")
	}

	if created, err := deployment.Ensure(config, operatorNamespace, operatorImage); err != nil {
		return err
	} else if created {
		fmt.Printf("* Deployed the operator successfully\n")
	}

	fmt.Printf("* The operator is up and running\n")

	return nil
}
