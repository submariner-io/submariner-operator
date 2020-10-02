package submariner

import (
	"fmt"

	submopv1a1 "github.com/submariner-io/submariner-operator/apis/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
)

func (r *SubmarinerReconciler) getClusterNetwork(submariner *submopv1a1.Submariner) (*network.ClusterNetwork, error) {
	const UnknownPlugin = "unknown"

	// If a previously cached discovery exists, use that
	if r.clusterNetwork != nil && r.clusterNetwork.NetworkPlugin != UnknownPlugin {
		return r.clusterNetwork, nil
	}

	clusterNetwork, err := network.Discover(r.dynClient, r.clientSet, r.submClient, submariner.Namespace)

	if clusterNetwork != nil {
		r.clusterNetwork = clusterNetwork
		log.Info("Cluster network discovered")
		clusterNetwork.Log(log)
	} else {
		r.clusterNetwork = &network.ClusterNetwork{NetworkPlugin: UnknownPlugin}
		log.Info("No cluster network discovered")
	}

	return r.clusterNetwork, err
}

func (r *SubmarinerReconciler) discoverNetwork(submariner *submopv1a1.Submariner) error {
	clusterNetwork, err := r.getClusterNetwork(submariner)
	submariner.Status.ClusterCIDR = getCIDR(
		"Cluster",
		submariner.Spec.ClusterCIDR,
		clusterNetwork.PodCIDRs)

	submariner.Status.ServiceCIDR = getCIDR(
		"Service",
		submariner.Spec.ServiceCIDR,
		clusterNetwork.ServiceCIDRs)

	submariner.Status.NetworkPlugin = clusterNetwork.NetworkPlugin

	//TODO: globalCIDR allocation if no global CIDR is assigned and enabled.
	//      currently the clusterNetwork discovers any existing operator setting,
	//      but that's not really helpful here
	return err
}

func getCIDR(cidrType, currentCIDR string, detectedCIDRs []string) string {
	detected := getFirstCIDR(detectedCIDRs)

	if currentCIDR == "" {
		if detected != "" {
			log.Info("Using detected CIDR", "type", cidrType, "CIDR", detected)
		} else {
			log.Info("No detected CIDR", "type", cidrType)
		}
		return detected
	}

	if detected != "" && detected != currentCIDR {
		log.Error(
			fmt.Errorf("there is a mismatch between the detected and configured CIDRs"),
			"The configured CIDR will take precedence",
			"type", cidrType, "configured", currentCIDR, "detected", detected)
	}
	return currentCIDR
}

func getFirstCIDR(detectedCIDRs []string) string {
	CIDRlen := len(detectedCIDRs)

	if CIDRlen > 1 {
		log.Error(fmt.Errorf("detected > 1 CIDRs"),
			"we currently support only one", "detectedCIDRs", detectedCIDRs)
	}
	if CIDRlen > 0 {
		return detectedCIDRs[0]
	}
	return ""
}
