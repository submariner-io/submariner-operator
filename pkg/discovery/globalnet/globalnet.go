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
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/bits"
	"net"

	"github.com/pkg/errors"
	"github.com/submariner-io/admiral/pkg/reporter"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/util/retry"
)

type Info struct {
	Enabled     bool
	CidrRange   string
	ClusterSize uint
	CidrInfo    map[string]*GlobalNetwork
}

type GlobalNetwork struct {
	GlobalCIDRs []string
	ClusterID   string
}

type GlobalCIDR struct {
	cidr              string
	net               *net.IPNet
	allocatedClusters []*CIDR
	allocatedCount    int
}

type CIDR struct {
	network *net.IPNet
	size    int
	lastIP  uint
}

type Config struct {
	ClusterID   string
	GlobalCIDR  string
	ClusterSize uint
}

var globalCidr = GlobalCIDR{allocatedCount: 0}

func isOverlappingCIDR(cidrList []string, cidr string) (bool, error) {
	_, newNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err // nolint:wrapcheck // No need to wrap here
	}

	for _, v := range cidrList {
		_, baseNet, err := net.ParseCIDR(v)
		if err != nil {
			return false, err // nolint:wrapcheck // No need to wrap here
		}

		if baseNet.Contains(newNet.IP) || newNet.Contains(baseNet.IP) {
			return true, nil
		}
	}

	return false, nil
}

func NewCIDR(cidr string) (CIDR, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return CIDR{}, fmt.Errorf("invalid cidr %q passed as input", cidr)
	}

	ones, total := network.Mask.Size()
	size := total - ones
	lastIP := LastIP(network)
	clusterCidr := CIDR{network: network, size: size, lastIP: lastIP}

	return clusterCidr, nil
}

func LastIP(network *net.IPNet) uint {
	ones, total := network.Mask.Size()
	clusterSize := uint(total - ones)
	firstIPInt := ipToUint(network.IP)
	lastIPUint := firstIPInt + 1<<clusterSize - 1

	return lastIPUint
}

func allocateByCidr(cidr string) (uint, error) {
	requestedIP, requestedNetwork, err := net.ParseCIDR(cidr)
	if err != nil || !globalCidr.net.Contains(requestedIP) {
		return 0, fmt.Errorf("%s not a valid subnet of %v", cidr, globalCidr.net)
	}

	var clusterCidr CIDR

	if clusterCidr, err = NewCIDR(cidr); err != nil {
		return 0, err
	}

	if !globalCidr.net.Contains(uintToIP(clusterCidr.lastIP)) {
		return 0, fmt.Errorf("%s not a valid subnet of %v", cidr, globalCidr.net)
	}

	for i := 0; i < globalCidr.allocatedCount; i++ {
		allocated := globalCidr.allocatedClusters[i]
		if allocated.network.Contains(requestedIP) {
			// subset of already allocated, try next
			return allocated.lastIP, fmt.Errorf("%s subset of already allocated globalCidr %v", cidr, allocated.network)
		}

		if requestedNetwork.Contains(allocated.network.IP) {
			// already allocated is subset of requested, no valid lastIP
			return clusterCidr.lastIP, fmt.Errorf("%s overlaps with already allocated globalCidr %s", cidr, allocated.network)
		}
	}

	globalCidr.allocatedClusters = append(globalCidr.allocatedClusters, &clusterCidr)
	globalCidr.allocatedCount++

	return 0, nil
}

func allocateByClusterSize(numSize uint) (string, error) {
	bitSize := bits.LeadingZeros(0) - bits.LeadingZeros(numSize-1)
	_, totalbits := globalCidr.net.Mask.Size()
	clusterPrefix := totalbits - bitSize
	mask := net.CIDRMask(clusterPrefix, totalbits)

	cidr := fmt.Sprintf("%s/%d", globalCidr.net.IP, clusterPrefix)

	last, err := allocateByCidr(cidr)
	if err != nil && last == 0 {
		return "", err
	}

	for err != nil {
		nextNet := net.IPNet{
			IP:   uintToIP(last + 1),
			Mask: mask,
		}
		cidr = nextNet.String()

		last, err = allocateByCidr(cidr)
		if err != nil && last == 0 {
			return "", fmt.Errorf("allocation not available")
		}
	}

	return cidr, nil
}

func AllocateGlobalCIDR(globalnetInfo *Info) (string, error) {
	globalCidr = GlobalCIDR{allocatedCount: 0, cidr: globalnetInfo.CidrRange}

	_, network, err := net.ParseCIDR(globalCidr.cidr)
	if err != nil {
		return "", fmt.Errorf("invalid GlobalCIDR %s configured", globalCidr.cidr)
	}

	globalCidr.net = network

	for _, globalNetwork := range globalnetInfo.CidrInfo {
		for _, otherCluster := range globalNetwork.GlobalCIDRs {
			otherClusterCIDR, err := NewCIDR(otherCluster)
			if err != nil {
				return "", err
			}

			globalCidr.allocatedClusters = append(globalCidr.allocatedClusters, &otherClusterCIDR)
			globalCidr.allocatedCount++
		}
	}

	return allocateByClusterSize(globalnetInfo.ClusterSize)
}

func ipToUint(ip net.IP) uint {
	intIP := ip
	if len(ip) == 16 {
		intIP = ip[12:16]
	}

	return uint(binary.BigEndian.Uint32(intIP))
}

func uintToIP(ip uint) net.IP {
	netIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(netIP, uint32(ip))

	return netIP
}

func GetValidClusterSize(cidrRange string, clusterSize uint) (uint, error) {
	_, network, err := net.ParseCIDR(cidrRange)
	if err != nil {
		return 0, err // nolint:wrapcheck // No need to wrap here
	}

	ones, totalbits := network.Mask.Size()
	availableSize := 1 << uint(totalbits-ones)
	userClusterSize := clusterSize
	clusterSize = nextPowerOf2(uint32(clusterSize))

	if clusterSize > uint(availableSize/2) {
		return 0, fmt.Errorf("cluster size %d, should be <= %d", userClusterSize, availableSize/2)
	}

	if clusterSize == 0 {
		return 0, errors.New("cluster size must be > 0")
	}

	return clusterSize, nil
}

// Refer: https://graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2
func nextPowerOf2(n uint32) uint {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++

	return uint(n)
}

func CheckOverlappingCidrs(globalnetInfo *Info, netconfig Config) error {
	var cidrlist []string
	var cidr string

	for k, v := range globalnetInfo.CidrInfo {
		cidrlist = v.GlobalCIDRs
		cidr = netconfig.GlobalCIDR

		overlap, err := isOverlappingCIDR(cidrlist, cidr)
		if err != nil {
			return errors.Wrap(err, "unable to validate overlapping CIDR")
		}

		if overlap && k != netconfig.ClusterID {
			return fmt.Errorf("invalid CIDR %s overlaps with cluster %q", cidr, k)
		}
	}

	return nil
}

func isCIDRPreConfigured(clusterID string, globalNetworks map[string]*GlobalNetwork) bool {
	// GlobalCIDR is not pre-configured
	if globalNetworks[clusterID] == nil || globalNetworks[clusterID].GlobalCIDRs == nil || len(globalNetworks[clusterID].GlobalCIDRs) == 0 {
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

	if globalnetInfo.Enabled && globalnetClusterSize != 0 && globalnetClusterSize != globalnetInfo.ClusterSize {
		clusterSize, err := GetValidClusterSize(globalnetInfo.CidrRange, globalnetClusterSize)
		if err != nil {
			return "", status.Error(err, "invalid cluster size")
		}

		globalnetInfo.ClusterSize = clusterSize
	}

	if globalnetCIDR != "" && globalnetClusterSize != 0 {
		status.Failure("Only one of cluster size and global CIDR can be specified")

		return "", errors.New("only one of cluster size and global CIDR can be specified")
	}

	if globalnetCIDR != "" {
		err := IsValidCIDR(globalnetCIDR)
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

			globalnetInfo.ClusterSize = 0
		}
	}

	return globalnetCIDR, nil
}

func GetGlobalNetworks(kubeClient kubernetes.Interface, brokerNamespace string) (*Info, *v1.ConfigMap, error) {
	configMap, err := GetConfigMap(kubeClient, brokerNamespace)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error retrieving globalnet ConfigMap")
	}

	globalnetInfo := Info{}

	err = json.Unmarshal([]byte(configMap.Data[globalnetEnabledKey]), &globalnetInfo.Enabled)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error reading globalnetEnabled status")
	}

	if globalnetInfo.Enabled {
		err = json.Unmarshal([]byte(configMap.Data[globalnetClusterSize]), &globalnetInfo.ClusterSize)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error reading GlobalnetClusterSize")
		}

		err = json.Unmarshal([]byte(configMap.Data[globalnetCidrRange]), &globalnetInfo.CidrRange)
		if err != nil {
			return nil, nil, errors.Wrap(err, "error reading GlobalnetCidrRange")
		}
	}

	var clusterInfo []clusterInfo

	err = json.Unmarshal([]byte(configMap.Data[clusterInfoKey]), &clusterInfo)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error reading globalnet clusterInfo")
	}

	globalNetworks := make(map[string]*GlobalNetwork)

	for _, cluster := range clusterInfo {
		globalNetwork := GlobalNetwork{
			GlobalCIDRs: cluster.GlobalCidr,
			ClusterID:   cluster.ClusterID,
		}
		globalNetworks[cluster.ClusterID] = &globalNetwork
	}

	globalnetInfo.CidrInfo = globalNetworks

	return &globalnetInfo, configMap, nil
}

func AssignGlobalnetIPs(globalnetInfo *Info, netconfig Config, status reporter.Interface) (string, error) {
	status.Start("Assigning Globalnet IPs")
	defer status.End()

	globalnetCIDR := netconfig.GlobalCIDR
	clusterID := netconfig.ClusterID
	var err error

	if globalnetCIDR == "" {
		// Globalnet enabled, GlobalCIDR not specified by the user
		if isCIDRPreConfigured(clusterID, globalnetInfo.CidrInfo) {
			// globalCidr already configured on this cluster
			globalnetCIDR = globalnetInfo.CidrInfo[clusterID].GlobalCIDRs[0]
			status.Success("Using pre-configured global CIDR %s", globalnetCIDR)
		} else {
			// no globalCidr configured on this cluster
			globalnetCIDR, err = AllocateGlobalCIDR(globalnetInfo)
			if err != nil {
				return "", status.Error(err, "unable to allocate global CIDR")
			}

			status.Success(fmt.Sprintf("Allocated global CIDR %s", globalnetCIDR))
		}
	} else {
		// Globalnet enabled, globalnetCIDR specified by user
		if isCIDRPreConfigured(clusterID, globalnetInfo.CidrInfo) {
			// globalCidr pre-configured on this cluster
			globalnetCIDR = globalnetInfo.CidrInfo[clusterID].GlobalCIDRs[0]
			status.Warning("A pre-configured global CIDR %s was detected - not using the specified CIDR %s",
				globalnetCIDR, netconfig.GlobalCIDR)
		} else {
			// globalCidr as specified by the user
			err := CheckOverlappingCidrs(globalnetInfo, netconfig)
			if err != nil {
				return "", status.Error(err, "error validating overlapping global CIDRs %s", globalnetCIDR)
			}

			status.Success("Using specified global CIDR %s", globalnetCIDR)
		}
	}

	return globalnetCIDR, nil
}

func IsValidCIDR(cidr string) error {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return err // nolint:wrapcheck // No need to wrap here
	}

	if ip.IsUnspecified() {
		return fmt.Errorf("%s can't be unspecified", cidr)
	}

	if ip.IsLoopback() {
		return fmt.Errorf("%s can't be in loopback range", cidr)
	}

	if ip.IsLinkLocalUnicast() {
		return fmt.Errorf("%s can't be in link-local range", cidr)
	}

	if ip.IsLinkLocalMulticast() {
		return fmt.Errorf("%s can't be in link-local multicast range", cidr)
	}

	return nil
}

func ValidateExistingGlobalNetworks(kubeClient kubernetes.Interface, namespace string) error {
	globalnetInfo, _, err := GetGlobalNetworks(kubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}

		return errors.Wrap(err, "error getting existing globalnet configmap")
	}

	if globalnetInfo != nil && globalnetInfo.Enabled {
		if err = IsValidCIDR(globalnetInfo.CidrRange); err != nil {
			return errors.Wrap(err, "invalid GlobalnetCidrRange")
		}
	}

	return nil
}

func AllocateAndUpdateGlobalCIDRConfigMap(brokerAdminClientset kubernetes.Interface, brokerNamespace string,
	netconfig *Config, status reporter.Interface,
) error {
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {
		status.Start("Retrieving Globalnet information from the Broker")
		defer status.End()

		globalnetInfo, globalnetConfigMap, err := GetGlobalNetworks(brokerAdminClientset, brokerNamespace)
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

			if globalnetInfo.CidrInfo[netconfig.ClusterID] == nil ||
				globalnetInfo.CidrInfo[netconfig.ClusterID].GlobalCIDRs[0] != netconfig.GlobalCIDR {
				var newClusterInfo clusterInfo
				newClusterInfo.ClusterID = netconfig.ClusterID
				newClusterInfo.GlobalCidr = []string{netconfig.GlobalCIDR}

				status.Start("Updating the Globalnet information on the Broker")

				err = updateConfigMap(brokerAdminClientset, brokerNamespace, globalnetConfigMap, newClusterInfo)
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

	return retryErr // nolint:wrapcheck // No need to wrap here
}
