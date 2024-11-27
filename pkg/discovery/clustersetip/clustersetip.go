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

package clustersetip

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
	ClusterID        string
	ClustersetIPCIDR string
	AllocationSize   uint
}

func AllocateClustersetIPCIDR(clustersetIPInfo *Info) (string, error) {
	return cidr.Allocate(&clustersetIPInfo.Info) //nolint:wrapcheck // No need to wrap
}

func ValidateClustersetIPConfiguration(clustersetIPInfo *Info, netconfig Config, status reporter.Interface) (string, error) {
	status.Start("Validating ClustersetIP configuration")
	defer status.End()

	clustersetIPClusterSize := netconfig.AllocationSize
	clustersetIPCIDR := netconfig.ClustersetIPCIDR

	if clustersetIPClusterSize != 0 && clustersetIPClusterSize != clustersetIPInfo.AllocationSize {
		clusterSize, err := cidr.GetValidAllocationSize(clustersetIPInfo.CIDR, clustersetIPClusterSize)
		if err != nil {
			return "", status.Error(err, "invalid cluster size")
		}

		clustersetIPInfo.AllocationSize = clusterSize
	}

	if clustersetIPCIDR != "" {
		err := cidr.IsValid(clustersetIPCIDR)
		if err != nil {
			return "", errors.Wrap(err, "specified clustersetip-cidr is invalid")
		}
	}

	return clustersetIPCIDR, nil
}

func GetClustersetIPNetworks(ctx context.Context, client controllerClient.Client, brokerNamespace string) (*Info, *v1.ConfigMap, error) {
	configMap, err := GetConfigMap(ctx, client, brokerNamespace)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error retrieving clustersetip ConfigMap")
	}

	clustersetIPInfo := Info{}

	err = json.Unmarshal([]byte(configMap.Data[clustersetIPEnabledKey]), &clustersetIPInfo.Enabled)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error reading clusersetIPEnabled status")
	}

	err = json.Unmarshal([]byte(configMap.Data[clustersetIPClusterSize]), &clustersetIPInfo.AllocationSize)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error reading ClustersetIPClusterSize")
	}

	err = json.Unmarshal([]byte(configMap.Data[clustersetIPCidrRange]), &clustersetIPInfo.CIDR)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error reading ClustersetIPCidrRange")
	}

	clustersetIPInfo.Clusters, err = cidr.ExtractClusterInfo(configMap)

	return &clustersetIPInfo, configMap, err //nolint:wrapcheck // No need to wrap
}

func assignClustersetIPs(clustersetIPInfo *Info, netconfig Config, status reporter.Interface) (string, error) {
	status.Start("Assigning ClustersetIP IPs")
	defer status.End()

	clustersetIPCIDR := netconfig.ClustersetIPCIDR
	clusterID := netconfig.ClusterID
	var err error

	if clustersetIPCIDR == "" {
		// ClustersetIPCIDR not specified by the user
		if cidr.IsCIDRPreConfigured(clusterID, clustersetIPInfo.Clusters) {
			// clustersetipCidr already configured on this cluster
			clustersetIPCIDR = clustersetIPInfo.Clusters[clusterID].CIDRs[0]
			status.Success("Using pre-configured clustersetip CIDR %s", clustersetIPCIDR)
		} else {
			// no clustersetipCidr configured on this cluster
			clustersetIPCIDR, err = AllocateClustersetIPCIDR(clustersetIPInfo)
			if err != nil {
				return "", status.Error(err, "unable to allocate clustersetip CIDR")
			}

			status.Success("Allocated clustersetip CIDR " + clustersetIPCIDR)
		}
	} else {
		// ClustersetIP enabled, clustersetIPCIDR specified by user
		if cidr.IsCIDRPreConfigured(clusterID, clustersetIPInfo.Clusters) {
			// clustersetipCidr pre-configured on this cluster
			clustersetIPCIDR = clustersetIPInfo.Clusters[clusterID].CIDRs[0]
			status.Warning("A pre-configured clustersetip CIDR %s was detected - not using the specified CIDR %s",
				clustersetIPCIDR, netconfig.ClustersetIPCIDR)
		} else {
			// clustersetipCidr as specified by the user
			err := cidr.CheckForOverlappingCIDRs(clustersetIPInfo.Clusters, netconfig.ClustersetIPCIDR, netconfig.ClusterID)
			if err != nil {
				return "", status.Error(err, "error validating overlapping clustersetip CIDRs %s", clustersetIPCIDR)
			}

			status.Success("Using specified clustersetip CIDR %s", clustersetIPCIDR)
		}
	}

	return clustersetIPCIDR, nil
}

func ValidateExistingClustersetIPNetworks(ctx context.Context, client controllerClient.Client, namespace string) error {
	clustersetIPInfo, _, err := GetClustersetIPNetworks(ctx, client, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return errors.Wrap(err, "error getting existing clustersetip configmap")
	}

	if clustersetIPInfo != nil {
		if err = cidr.IsValid(clustersetIPInfo.CIDR); err != nil {
			return errors.Wrap(err, "invalid ClustersetIPCidrRange")
		}
	}

	return nil
}

func AllocateCIDRFromConfigMap(ctx context.Context, brokerAdminClient controllerClient.Client, brokerNamespace string,
	config *Config, status reporter.Interface,
) (bool, error) {
	// Setup default clustersize if nothing specified
	if config.ClustersetIPCIDR == "" && config.AllocationSize == 0 {
		config.AllocationSize = DefaultAllocationSize
	}

	enabled := false
	userClustersetIPCIDR := config.ClustersetIPCIDR

	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		status.Start("Retrieving ClustersetIP information from the Broker")
		defer status.End()

		clustersetIPInfo, clustersetIPConfigMap, err := GetClustersetIPNetworks(ctx, brokerAdminClient, brokerNamespace)
		if err != nil {
			return status.Error(err, "unable to retrieve ClustersetIP information")
		}

		config.ClustersetIPCIDR, err = ValidateClustersetIPConfiguration(clustersetIPInfo, *config, status)
		if err != nil {
			return status.Error(err, "error validating the ClustersetIP configuration")
		}

		config.ClustersetIPCIDR, err = assignClustersetIPs(clustersetIPInfo, *config, status)
		if err != nil {
			return status.Error(err, "error assigning ClustersetIP IPs")
		}

		enabled = clustersetIPInfo.Enabled

		if clustersetIPInfo.Clusters[config.ClusterID] == nil ||
			clustersetIPInfo.Clusters[config.ClusterID].CIDRs[0] != config.ClustersetIPCIDR {
			newClusterInfo := cidr.ClusterInfo{
				ClusterID: config.ClusterID,
				CIDRs:     []string{config.ClustersetIPCIDR},
			}

			status.Start("Updating the ClustersetIP information on the Broker")

			err = updateConfigMap(ctx, brokerAdminClient, clustersetIPConfigMap, newClusterInfo)
			if apierrors.IsConflict(err) {
				status.Warning("Conflict occurred updating the ClustersetIP ConfigMap - retrying")
				// Conflict with allocation, retry with user given CIDR to try reallocation
				config.ClustersetIPCIDR = userClustersetIPCIDR
			} else {
				return status.Error(err, "error updating the ClustersetIP ConfigMap")
			}

			return err
		}

		return nil
	})

	return enabled, retryErr //nolint:wrapcheck // No need to wrap here
}
