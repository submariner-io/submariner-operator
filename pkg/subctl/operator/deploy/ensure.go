package deploy

import (
	"encoding/base64"
	"fmt"
	"strings"

	submariner "github.com/submariner-io/submariner-operator/pkg/apis/submariner/v1alpha1"
	submarinerclientset "github.com/submariner-io/submariner-operator/pkg/client/clientset/versioned"
	"github.com/submariner-io/submariner-operator/pkg/engine"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func Ensure(config *rest.Config, submarinerNamespace string, repository string, version string,
	clusterID string, serviceCIDR string, clusterCIDR string, colorCodes string, nattPort int,
	ikePort int, subctlData *datafile.SubctlData) error {

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

	brokerURL := subctlData.BrokerURL
	if idx := strings.Index(brokerURL, "://"); idx >= 0 {
		// Submariner doesn't work with a schema prefix
		brokerURL = brokerURL[(idx + 3):]
	}

	if len(repository) == 0 {
		// Default repository
		// This is handled in the operator after 0.0.1 (of the operator)
		repository = "quay.io/submariner"
	}

	if len(version) == 0 {
		// Default engine version
		// This is handled in the operator after 0.0.1 (of the operator)
		version = "0.0.2"
	}

	// Populate the Submariner CR for the operator
	submarinerSpec := submariner.SubmarinerSpec{
		Repository:               repository,
		Version:                  version,
		CeIPSecNATTPort:          nattPort,
		CeIPSecIKEPort:           ikePort,
		CeIPSecDebug:             false,
		CeIPSecPSK:               base64.StdEncoding.EncodeToString(subctlData.IPSecPSK.Data["psk"]),
		BrokerK8sCA:              base64.StdEncoding.EncodeToString(subctlData.ClientToken.Data["ca.crt"]),
		BrokerK8sRemoteNamespace: string(subctlData.ClientToken.Data["namespace"]),
		BrokerK8sApiServerToken:  string(subctlData.ClientToken.Data["token"]),
		BrokerK8sApiServer:       brokerURL,
		Broker:                   "k8s",
		NatEnabled:               false,
		Debug:                    false,
		ColorCodes:               colorCodes,
		ClusterID:                clusterID,
		ServiceCIDR:              serviceCIDR,
		ClusterCIDR:              clusterCIDR,
		Namespace:                submarinerNamespace,
		Count:                    0,
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
