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

package globalnet

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/reporter"
	"github.com/submariner-io/submariner-operator/pkg/cidr"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/util/retry"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Info struct {
	Enabled bool
	cidr.Info
}

type Config struct {
	ClusterID   string
	GlobalCIDR  string
	ClusterSize uint
}

func AllocateGlobalCIDR(globalnetInfo *Info) (string, error) {
	return cidr.Allocate(&globalnetInfo.Info) //nolint:wrapcheck // No need to wrap
}

func isCIDRPreConfigured(clusterID string, globalNetworks map[string]*cidr.ClusterInfo) bool {
	// GlobalCIDR is not pre-configured
	if globalNetworks[clusterID] == nil || globalNetworks[clusterID].CIDRs == nil || len(globalNetworks[clusterID].CIDRs) == 0 {
		return false
	}

	// GlobalCIDR is pre-configured
	return true
}

func ValidateGlobalnetConfiguration(globalnetInfo *Info, netconfig Config, status reporter.Interface) (string, error) {
	status.Start("Validating Globalnet configuration")
	defer status.End()

	globalnetClusterSize := netconfig.ClusterSize
	globalnetCIDR := netconfig.GlobalCIDR

	if globalnetInfo.Enabled && globalnetClusterSize != 0 && globalnetClusterSize != globalnetInfo.AllocationSize {
		clusterSize, err := cidr.GetValidAllocationSize(globalnetInfo.CIDR, globalnetClusterSize)
		if err != nil {
			return "", status.Error(err, "invalid cluster size")
		}

		globalnetInfo.AllocationSize = clusterSize
	}

	if globalnetCIDR != "" && globalnetClusterSize != 0 {
		status.Failure("Only one of cluster size and global CIDR can be specified")

		return "", errors.New("only one of cluster size and global CIDR can be specified")
	}

	if globalnetCIDR != "" {
		err := cidr.IsValid(globalnetCIDR)
		if err != nil {
			return "", errors.Wrap(err, "specified globalnet-cidr is invalid")
		}
	}

	if !globalnetInfo.Enabled {
		if globalnetCIDR != "" {
			status.Warning("Globalnet is not enabled on the Broker - ignoring the specified global CIDR")

			globalnetCIDR = ""
		} else if globalnetClusterSize != 0 {
			status.Warning("Globalnet is not enabled on the Broker - ignoring the specified cluster size")

			globalnetInfo.AllocationSize = 0
		}
	}

	return globalnetCIDR, nil
}

func GetGlobalNetworks(ctx context.Context, client controllerClient.Client, brokerNamespace string) (*Info, *v1.ConfigMap, error) {
	configMap, err := GetConfigMap(ctx, client, brokerNamespace)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error retrieving globalnet ConfigMap")
	}

	globalnetInfo := Info{}

	err = json.Unmarshal([]byte(configMap.Data[globalnetEnabledKey]), &globalnetInfo.Enabled)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error reading globalnetEnabled status")
	}

	if globalnetInfo.Enabled {
		err = json.Unmarshal([]byte(configMap.Data[globalnetClusterSize]), &globalnetInfo.AllocationSize)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error reading GlobalnetClusterSize")
		}

		err = json.Unmarshal([]byte(configMap.Data[globalnetCidrRange]), &globalnetInfo.CIDR)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error reading GlobalnetCidrRange")
		}
	}

	globalnetInfo.Clusters, err = cidr.ExtractClusterInfo(configMap)

	return &globalnetInfo, configMap, err //nolint:wrapcheck // No need to wrap
}

func AssignGlobalnetIPs(globalnetInfo *Info, netconfig Config, status reporter.Interface) (string, error) {
	status.Start("Assigning Globalnet IPs")
	defer status.End()

	globalnetCIDR := netconfig.GlobalCIDR
	clusterID := netconfig.ClusterID
	var err error

	if globalnetCIDR == "" {
		// Globalnet enabled, GlobalCIDR not specified by the user
		if isCIDRPreConfigured(clusterID, globalnetInfo.Clusters) {
			// globalCidr already configured on this cluster
			globalnetCIDR = globalnetInfo.Clusters[clusterID].CIDRs[0]
			status.Success("Using pre-configured global CIDR %s", globalnetCIDR)
		} else {
			// no globalCidr configured on this cluster
			globalnetCIDR, err = AllocateGlobalCIDR(globalnetInfo)
			if err != nil {
				return "", status.Error(err, "unable to allocate global CIDR")
			}

			status.Success("Allocated global CIDR " + globalnetCIDR)
		}
	} else {
		// Globalnet enabled, globalnetCIDR specified by user
		if cidr.IsCIDRPreConfigured(clusterID, globalnetInfo.Clusters) {
			// globalCidr pre-configured on this cluster
			globalnetCIDR = globalnetInfo.Clusters[clusterID].CIDRs[0]
			status.Warning("A pre-configured global CIDR %s was detected - not using the specified CIDR %s",
				globalnetCIDR, netconfig.GlobalCIDR)
		} else {
			// globalCidr as specified by the user
			err := cidr.CheckForOverlappingCIDRs(globalnetInfo.Clusters, netconfig.GlobalCIDR, netconfig.ClusterID)
			if err != nil {
				return "", status.Error(err, "error validating overlapping global CIDRs %s", globalnetCIDR)
			}

			status.Success("Using specified global CIDR %s", globalnetCIDR)
		}
	}

	return globalnetCIDR, nil
}

func ValidateExistingGlobalNetworks(ctx context.Context, client controllerClient.Client, namespace string) error {
	globalnetInfo, _, err := GetGlobalNetworks(ctx, client, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return errors.Wrap(err, "error getting existing globalnet configmap")
	}

	if globalnetInfo != nil && globalnetInfo.Enabled {
		if err = cidr.IsValid(globalnetInfo.CIDR); err != nil {
			return errors.Wrap(err, "invalid GlobalnetCidrRange")
		}
	}

	return nil
}

func AllocateAndUpdateGlobalCIDRConfigMap(ctx context.Context, brokerAdminClient controllerClient.Client, brokerNamespace string,
	netconfig *Config, status reporter.Interface,
) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		status.Start("Retrieving Globalnet information from the Broker")
		defer status.End()

		globalnetInfo, globalnetConfigMap, err := GetGlobalNetworks(ctx, brokerAdminClient, brokerNamespace)
		if err != nil {
			return status.Error(err, "unable to retrieve Globalnet information")
		}

		netconfig.GlobalCIDR, err = ValidateGlobalnetConfiguration(globalnetInfo, *netconfig, status)
		if err != nil {
			return status.Error(err, "error validating the Globalnet configuration")
		}

		if globalnetInfo.Enabled {
			netconfig.GlobalCIDR, err = AssignGlobalnetIPs(globalnetInfo, *netconfig, status)
			if err != nil {
				return status.Error(err, "error assigning Globalnet IPs")
			}

			if globalnetInfo.Clusters[netconfig.ClusterID] == nil ||
				globalnetInfo.Clusters[netconfig.ClusterID].CIDRs[0] != netconfig.GlobalCIDR {
				newClusterInfo := cidr.ClusterInfo{
					ClusterID: netconfig.ClusterID,
					CIDRs:     []string{netconfig.GlobalCIDR},
				}

				status.Start("Updating the Globalnet information on the Broker")

				err = updateConfigMap(ctx, brokerAdminClient, globalnetConfigMap, newClusterInfo)
				if apierrors.IsConflict(err) {
					status.Warning("Conflict occurred updating the Globalnet ConfigMap - retrying")
				} else {
					return status.Error(err, "error updating the Globalnet ConfigMap")
				}

				return err
			}
		}

		return nil
	})

	return retryErr //nolint:wrapcheck // No need to wrap here
}
