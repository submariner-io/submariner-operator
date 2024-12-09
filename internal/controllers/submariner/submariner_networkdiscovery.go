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

package submariner

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	submopv1a1 "github.com/submariner-io/submariner-operator/api/v1alpha1"
	"github.com/submariner-io/submariner-operator/pkg/discovery/network"
)

func (r *Reconciler) getClusterNetwork(ctx context.Context, submariner *submopv1a1.Submariner) (*network.ClusterNetwork, error) {
	const UnknownPlugin = "unknown"

	// If a previously cached discovery exists, use that
	if r.config.ClusterNetwork != nil && r.config.ClusterNetwork.NetworkPlugin != UnknownPlugin {
		return r.config.ClusterNetwork, nil
	}

	clusterNetwork, err := network.Discover(ctx, r.config.GeneralClient, submariner.Namespace)
	if err != nil {
		log.Error(err, "Error trying to discover network")
	}

	if clusterNetwork != nil {
		log.Info("Cluster network discovered")

		r.config.ClusterNetwork = clusterNetwork
		clusterNetwork.Log(log)
	} else {
		log.Info("No cluster network discovered")

		r.config.ClusterNetwork = &network.ClusterNetwork{NetworkPlugin: UnknownPlugin}
	}

	return r.config.ClusterNetwork, errors.Wrap(err, "error discovering cluster network")
}

func (r *Reconciler) discoverNetwork(ctx context.Context, submariner *submopv1a1.Submariner, log logr.Logger,
) (*network.ClusterNetwork, error) {
	clusterNetwork, err := r.getClusterNetwork(ctx, submariner)
	submariner.Status.ClusterCIDR = getCIDR(
		log,
		"Cluster",
		submariner.Spec.ClusterCIDR,
		clusterNetwork.PodCIDRs)

	submariner.Status.ServiceCIDR = getCIDR(
		log,
		"Service",
		submariner.Spec.ServiceCIDR,
		clusterNetwork.ServiceCIDRs)

	submariner.Status.NetworkPlugin = clusterNetwork.NetworkPlugin

	return clusterNetwork, err
}

func getCIDR(log logr.Logger, cidrType, currentCIDR string, detectedCIDRs []string) string {
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
			errors.New("there is a mismatch between the detected and configured CIDRs"),
			"The configured CIDR will take precedence",
			"type", cidrType, "configured", currentCIDR, "detected", detected)
	}

	return currentCIDR
}

func getFirstCIDR(detectedCIDRs []string) string {
	CIDRlen := len(detectedCIDRs)

	if CIDRlen > 1 {
		log.Error(errors.New("detected > 1 CIDRs"),
			"we currently support only one", "detectedCIDRs", detectedCIDRs)
	}

	if CIDRlen > 0 {
		return detectedCIDRs[0]
	}

	return ""
}
